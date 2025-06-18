package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"ebs/src/boot"
	"ebs/src/config"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/middlewares"
	"ebs/src/models"
	"ebs/src/types"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stripe/stripe-go/v82"
	engineiotypes "github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

type Claims struct {
	Username     string   `json:"username"`
	Role         string   `json:"role"`
	Permissions  []string `json:"permissions"`
	Organization uint
	UID          string `json:"uid"`
	jwt.RegisteredClaims
}

func (c Claims) GetExpirationTime() (*jwt.NumericDate, error) {
	return c.RegisteredClaims.GetExpirationTime()
}
func (c Claims) GetIssuedAt() (*jwt.NumericDate, error) {
	return c.RegisteredClaims.GetIssuedAt()
}
func (c Claims) GetNotBefore() (*jwt.NumericDate, error) {
	return c.RegisteredClaims.GetNotBefore()
}
func (c Claims) GetIssuer() (string, error) {
	return c.RegisteredClaims.GetIssuer()
}
func (c Claims) GetSubject() (string, error) {
	return c.RegisteredClaims.GetSubject()
}
func (c Claims) GetAudience() (jwt.ClaimStrings, error) {
	return c.RegisteredClaims.GetAudience()
}

var jwtKey = []byte(os.Getenv("JWT_SECRET"))

var eventDateTimeValidatorFunc validator.Func = func(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(string)
	datetime, err := time.Parse(config.TIME_PARSE_FORMAT, date)
	if err != nil {
		return false
	}
	today := time.Now()
	log.Printf("%s: ok=%v,v=%v,n=%v", fl.FieldName(), ok, datetime, today)
	if ok {
		today := time.Now()
		if today.After(datetime) {
			return false
		}
	}
	return true
}

var gtfield validator.Func = func(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(string)
	datetime, err := time.Parse(config.TIME_PARSE_FORMAT, date)
	if err != nil {
		return false
	}
	field := fl.Parent().FieldByName(fl.Param())
	fieldValue := field.Interface().(string)
	fielddatetime, err := time.Parse(config.TIME_PARSE_FORMAT, fieldValue)
	if err != nil {
		return false
	}
	log.Printf("%s: param=%s, ok=%v,v=%v,n=%v", fl.FieldName(), fl.Param(), ok, datetime, fielddatetime)
	if ok {
		if fielddatetime.After(datetime) {
			return false
		}
	}
	return true
}

var ltfield validator.Func = func(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(string)
	datetime, err := time.Parse(config.TIME_PARSE_FORMAT, date)
	if err != nil {
		return false
	}
	field := fl.Parent().FieldByName(fl.Param())
	fieldValue := field.Interface().(string)
	fielddatetime, err := time.Parse(config.TIME_PARSE_FORMAT, fieldValue)
	if err != nil {
		return false
	}
	log.Printf("%s: param=%s, ok=%v,v=%v,n=%v", fl.FieldName(), fl.Param(), ok, datetime, fielddatetime)
	if ok {
		if datetime.After(fielddatetime) {
			return false
		}
	}
	return true
}

var betweenfields validator.Func = func(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(string)
	datetime, err := time.Parse(config.TIME_PARSE_FORMAT, date)
	if err != nil {
		return false
	}
	log.Printf("param: %s\n", fl.Param())
	field1 := fl.Parent().FieldByName(fl.Param())
	fieldValue1 := field1.Interface().(string)
	fielddatetime1, err := time.Parse(config.TIME_PARSE_FORMAT, fieldValue1)
	if err != nil {
		return false
	}
	field2 := fl.Parent().FieldByName(fl.Param())
	fieldValue2 := field2.Interface().(string)
	fielddatetime2, err := time.Parse(config.TIME_PARSE_FORMAT, fieldValue2)
	if err != nil {
		return false
	}
	log.Printf("%s: ok=%v,v1=%v,v2=%v", fl.FieldName(), ok, fielddatetime1, fielddatetime2)
	if ok {
		if fielddatetime1.After(datetime) || datetime.After(fielddatetime2) {
			return false
		}
	}
	return true
}

func setupRouter() *gin.Engine {
	router := gin.Default()
	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "ok")
	})
	return router
}

func maintenanceModeMiddleware(g *gin.Engine) *gin.Engine {
	g.Use(func(ctx *gin.Context) {
		mm := os.Getenv("MAINTENANCE_MODE")
		atoi, err := strconv.ParseBool(mm)
		if err != nil || atoi {
			err := errors.New("server is under maintenance")
			log.Println(err.Error())
			ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, err.Error())
			return
		}
	})
	return g
}

func apiv1Group(g *gin.Engine) *gin.RouterGroup {
	apiv1 := g.Group("/api/v1")
	return apiv1
}

func guestAuthRoutes(g *gin.Engine) *gin.RouterGroup {
	apiv1 := apiv1Group(g)
	guest := apiv1.Group("/auth")
	guest.Use(middlewares.VerifyIdToken)
	guest.
		POST("/login", func(ctx *gin.Context) {
			var body types.RegisterUserRequestBody
			if err := ctx.ShouldBindJSON(&body); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			auth, err := lib.GetFirebaseAuth()
			if err != nil {
				log.Printf("Error initializing FirebaseAuth client: %s\n", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			user, err := auth.GetUserByEmail(context.Background(), body.Email)
			if err != nil {
				log.Printf("error from Firebase: %s\n", err.Error())
				ctx.JSON(http.StatusNotFound, gin.H{"error": "No user account is associated with this email"})
				return
			}

			db := db.GetDb()
			var muser models.User
			if err := db.
				Model(&models.User{}).
				Select("id", "name", "email").
				Where(&models.User{Email: user.Email}).
				First(&muser).
				Error; err != nil {
				log.Printf("error: %s\n", err.Error())
				ctx.JSON(http.StatusNotFound, gin.H{"error": "No user account is associated with this email"})
				return
			}

			err = db.Transaction(func(tx *gorm.DB) error {
				if err := db.
					Model(&models.User{}).
					Where("id", muser.ID).
					Update("last_active", time.Now()).
					Error; err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return err
				}
				return nil
			})
			if err != nil {
				log.Printf("Error logging in user [%d]: %s\n", muser.ID, err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}

			token, _ := generateJWT(user.Email, muser.ID, muser.ActiveOrg)

			uid := ctx.GetString("uid")
			go func() {
				rd := lib.GetRedisClient()
				_, err = rd.JSONSet(ctx, fmt.Sprintf("%d:user", muser.ID), "$", &muser).Result()
				if err != nil {
					log.Printf("[redis] Error updating user cache: %s\n", err.Error())
				}
				_, err = rd.JSONSet(ctx, fmt.Sprintf("%d:meta", muser.ID), "$", map[string]string{"photoURL": user.PhotoURL}).Result()
				if err != nil {
					log.Printf("[redis] Error updating user cache: %s\n", err.Error())
				}
				token := rd.JSONGet(context.Background(), fmt.Sprintf("%s:fcm", uid), "$.token").Val()
				fcm, _ := lib.GetFirebaseMessaging()
				fcm.SubscribeToTopic(ctx.Copy(), []string{token}, "Notifications")
			}()

			ctx.JSON(http.StatusOK, gin.H{
				"token": token,
			})
		}).
		POST("/register", func(ctx *gin.Context) {
			var body types.RegisterUserRequestBody
			if err := ctx.ShouldBindJSON(&body); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			auth, err := lib.GetFirebaseAuth()
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			user, err := auth.GetUserByEmail(context.Background(), body.Email)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			db := db.GetDb()
			var muser models.User
			err = db.Transaction(func(tx *gorm.DB) error {
				err := tx.
					Model(&models.User{}).
					Select("id").
					Where(&models.User{Email: user.Email}).
					Find(&muser).
					Error

				if err != nil {
					log.Printf("error: %s\n", err.Error())
					err := errors.New("user already exists")
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return err
				}

				newUser := models.User{
					Email: user.Email,
					UID:   user.UID,
					Role:  types.ROLE_OWNER,
					Name:  user.DisplayName,
				}
				err = tx.Create(&newUser).Error
				if err != nil {
					return err
				}

				newOrg := models.Organization{
					Name:         fmt.Sprintf("%s's organization", user.DisplayName),
					OwnerID:      newUser.ID,
					Type:         types.ORG_PERSONAL,
					ContactEmail: user.Email,
					TenantID:     newUser.TenantID,
					Status:       "active",
				}
				err = tx.Create(&newOrg).Error
				if err != nil {
					return err
				}

				newTeam := models.Team{
					OrganizationID: newOrg.ID,
					OwnerID:        newUser.ID,
					Name:           "Default",
					Status:         "active",
				}
				err = tx.Create(&newTeam).Error
				if err != nil {
					return err
				}
				sc := lib.GetStripeClient()
				acc, err := sc.V1Accounts.Create(context.Background(), &stripe.AccountCreateParams{
					BusinessProfile: &stripe.AccountCreateBusinessProfileParams{
						Name:         stripe.String(newOrg.Name),
						SupportEmail: stripe.String(newOrg.ContactEmail),
					},
					BusinessType: stripe.String("individual"),
					Company: &stripe.AccountCreateCompanyParams{
						Name: stripe.String(newOrg.Name),
					},
					Type:     stripe.String("express"),
					Email:    stripe.String(newOrg.ContactEmail),
					Metadata: map[string]string{"organizationId": fmt.Sprintf("%d", newOrg.ID)},
					Capabilities: &stripe.AccountCreateCapabilitiesParams{
						CardPayments: &stripe.AccountCreateCapabilitiesCardPaymentsParams{
							Requested: stripe.Bool(true),
						},
						Transfers: &stripe.AccountCreateCapabilitiesTransfersParams{
							Requested: stripe.Bool(true),
						},
					},
				})
				if err != nil {
					log.Printf("Error creating account for organization: %s\n", err.Error())
					return errors.New("error creating account for organization")
				}
				if err := tx.
					Model(&models.Organization{}).
					Where("id = ?", newOrg.ID).
					Updates(&models.Organization{
						StripeAccountID: &acc.ID,
					}).Error; err != nil {
					log.Printf("Error creating Connect account: %s\n", err.Error())
				}

				err = tx.
					Model(&models.User{}).
					Where(&models.User{ID: newUser.ID}).
					Update("active_org", newOrg.ID).Error
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx.JSON(http.StatusOK, gin.H{"uid": user.UID})
		})
	return guest
}

func setupSocketServer(r *gin.Engine) *socket.Server {
	c := socket.DefaultServerOptions()
	c.SetServeClient(true)
	c.SetPingInterval(time.Second)
	c.SetPingTimeout(200 * time.Millisecond)
	c.SetMaxHttpBufferSize(1_000_000)
	c.SetConnectTimeout(time.Second)
	// c.SetTransports(engineiotypes.NewSet("polling", "websocket"))
	c.SetCors(&engineiotypes.Cors{
		Origin:      "*",
		Credentials: true,
	})

	wss := socket.NewServer(nil, nil)
	wss.On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		fmt.Println("[newclient]: ", string(client.Id()), client.Nsp().Name())
		client.On("message", func(args ...any) {
			client.Emit("message-back", args...)
		})
		// client.Emit("auth", client.Handshake().Auth)
		client.On("message-with-ack", func(args ...any) {
			ack := args[len(args)-1].(socket.Ack)
			ack(args[:len(args)-1], nil)
		})
		client.On("event", func(data ...any) {
			log.Printf("Event for client [%s]: %v\n", string(client.Id()), data)
		})
	})
	wss.Of("/sub", nil).On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		fmt.Println("[newclient]: ", string(client.Id()), client.Nsp().Name())
		// client.Emit("auth", client.Handshake().Auth)
		client.On("test", func(data ...any) {
			log.Printf("received test from client %s with data %v\n", string(client.Id()), data)
			client.EmitWithAck("test", "pong")(func(args []any, err error) {
				log.Fatal(args, err)
			})
		})
	})
	wss.Emit("test", "ping")

	r.GET("/socket.io/*any", gin.WrapH(wss.ServeHandler(c)))
	r.POST("/socket.io/*any", gin.WrapH(wss.ServeHandler(c)))
	return wss
}

func main() {
	boot.InitDb()

	go boot.DownloadSDKFileFromS3()
	go lib.StripeInitialize()
	// go boot.InitScheduler()
	go boot.InitBroker()

	router := setupRouter()
	wss := setupSocketServer(router)
	if wss != nil {
		log.Println("WS server listening for connections...")
	}

	appEnv := os.Getenv("APP_ENV")
	appHost := os.Getenv("APP_HOST")
	if appEnv == "local" {
		router.Use(cors.Default())
	} else {
		cc := cors.DefaultConfig()
		cc.AllowMethods = append(cc.AllowMethods, "GET", "POST", "PATCH", "PUT", "DELETE", "HEAD")
		cc.AllowHeaders = append(cc.AllowHeaders, "Origin", "Authorization", "x-secret")
		cc.AllowOriginFunc = func(origin string) bool {
			match, _ := regexp.MatchString(`(\w+.?)+\.amazonaws\.com$`, origin)
			log.Printf("Origin matches %s: %v\n", origin, match)
			if match {
				return true
			}
			match, _ = regexp.MatchString(appHost, origin)
			if match {
				return true
			}
			match, _ = regexp.MatchString("app:mobile", origin)
			return match
		}
		cc.AllowCredentials = true
		cc.AllowAllOrigins = false
		router.Use(cors.New(cc))
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("bookabledate", eventDateTimeValidatorFunc)
		v.RegisterValidation("gtdate", gtfield)
		v.RegisterValidation("ltdate", ltfield)
		v.RegisterValidation("betweenfields", betweenfields)
	}

	router = maintenanceModeMiddleware(router)

	guestAuthRoutes(router)

	stripeWebhookRoute(router)

	authorized := router.Group("/api/v1")
	authorized.Use(middlewares.AuthMiddleware)
	{

		authorized.
			POST("/fcm", func(ctx *gin.Context) {
				var body struct {
					Token  string   `json:"token" binding:"required"`
					Topics []string `json:"topics" binding:"required"`
				}
				if err := ctx.ShouldBindJSON(&body); err != nil {
					log.Printf("[FCM] error: %v\n", err)
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				fcm, err := lib.GetFirebaseMessaging()
				if err != nil {
					log.Printf("Could not retrieve FCM instance: %v\n", err)
					ctx.Status(http.StatusInternalServerError)
					return
				}
				for _, topic := range body.Topics {
					_, err := fcm.SubscribeToTopic(ctx, []string{body.Token}, topic)
					if err != nil {
						log.Printf("[FCM] error subscribing to topic [%s]: %v\n", topic, err)
						ctx.Status(http.StatusBadRequest)
						return
					}
				}
				uid := ctx.GetString("uid")
				rd := lib.GetRedisClient()
				rd.JSONSet(context.Background(), fmt.Sprintf("%s:fcm", uid), "$", map[string]any{
					"token":  body.Token,
					"topics": body.Topics,
				})

				ctx.Status(http.StatusOK)
			}).
			POST("/fcm/send", func(ctx *gin.Context) {
				var body struct {
					Topic string `json:"topic" binding:"required"`
				}
				if err := ctx.ShouldBindJSON(&body); err != nil {
					log.Printf("[FCM] error: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				fcm, err := lib.GetFirebaseMessaging()
				if err != nil {
					log.Printf("Could not retrieve FCM instance: %v\n", err)
					ctx.Status(http.StatusInternalServerError)
					return
				}
				res, err := fcm.Send(context.Background(), &messaging.Message{
					Data: map[string]string{
						"test": "abc",
					},
					Topic: body.Topic,
				})
				if err != nil {
					log.Fatalln(err)
				}
				log.Println("successfully sent message:", res)
				ctx.Status(http.StatusOK)
			})

		authorized.
			POST("/auth/logout", func(ctx *gin.Context) {
				db := db.GetDb()
				if err := db.Transaction(func(tx *gorm.DB) error {
					userId := ctx.GetUint("id")
					err := tx.Model(&models.User{}).Where(userId).Update("last_active", time.Now()).Error
					if err != nil {
						return err
					}
					return nil
				}); err != nil {
					log.Printf("Error on user logout: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				uid := ctx.GetString("uid")

				go func() {
					rd := lib.GetRedisClient()
					token := rd.JSONGet(context.Background(), fmt.Sprintf("%s:fcm", uid), "$.token").Val()
					fcm, _ := lib.GetFirebaseMessaging()
					fcm.SubscribeToTopic(ctx.Copy(), []string{token}, "Notifications")
				}()

				ctx.Status(http.StatusOK)
			})

		authorized.
			POST("/users", func(ctx *gin.Context) {}).
			GET("/users/:id", func(ctx *gin.Context) {
				id, found := ctx.Params.Get("id")
				ctx.JSON(http.StatusOK, gin.H{
					"id":    id,
					"found": found,
				})
			})

		authorized = organizationHandlers(authorized)
		authorized = eventHandlers(authorized)
		authorized = ticketHandlers(authorized)
		authorized = bookingHandlers(authorized)
		authorized = reservationHandlers(authorized)
		authorized = admissionHandlers(authorized)
		authorized = transactionHandlers(authorized)

		authorized.
			GET("/me", func(ctx *gin.Context) {
				rd := lib.GetRedisClient()
				userId := ctx.GetUint("id")
				cacheKey := fmt.Sprintf("%d:user", userId)
				res := rd.JSONGet(context.Background(), cacheKey).Val()
				if res == "" {
					log.Printf("content not found [%s]\n", cacheKey)
					auth, err := lib.GetFirebaseAuth()
					if err != nil {
						log.Printf("Error initializing FirebaseAuth client: %s\n", err.Error())
						ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
						return
					}
					email := ctx.GetString("email")
					user, err := auth.GetUserByEmail(context.Background(), email)
					if err != nil {
						log.Printf("error from Firebase: %s\n", err.Error())
						ctx.JSON(http.StatusNotFound, gin.H{"error": "No user account is associated with this email"})
						return
					}
					db := db.GetDb()
					var muser models.User
					if err := db.
						Model(&models.User{}).
						Select("id", "name", "email").
						Where(&models.User{Email: user.Email}).
						First(&muser).
						Error; err != nil {
						log.Printf("error: %s\n", err.Error())
						ctx.JSON(http.StatusNotFound, gin.H{"error": "No user account is associated with this email"})
						return
					}

					mm := map[string]string{"photoURL": user.PhotoURL}
					go func() {
						rd := lib.GetRedisClient()
						_, err = rd.JSONSet(ctx, fmt.Sprintf("%d:user", muser.ID), "$", &muser).Result()
						if err != nil {
							log.Printf("[redis] Error updating user cache: %s\n", err.Error())
						}
						_, err = rd.JSONSet(ctx, fmt.Sprintf("%d:meta", muser.ID), "$", &mm).Result()
						if err != nil {
							log.Printf("[redis] Error updating user cache: %s\n", err.Error())
						}
					}()

					ctx.JSON(http.StatusOK, gin.H{"data": map[string]any{
						"me": map[string]string{
							"name":   muser.Name,
							"email":  muser.Email,
							"avatar": user.PhotoURL,
						},
						"md": mm,
					}})
					return
				}
				var user models.User
				err := json.Unmarshal([]byte(res), &user)
				if err != nil {
					log.Printf("Error on json unmarshal: %s\n", err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				var mm map[string]string
				res = rd.JSONGet(context.Background(), fmt.Sprintf("%d:meta", userId)).Val()
				err = json.Unmarshal([]byte(res), &mm)
				if err != nil {
					log.Printf("Error on json unmarshal: %s\n", err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": map[string]any{
					"me": map[string]string{
						"name":   user.Name,
						"email":  user.Email,
						"avatar": mm["photoURL"],
					},
					"md": mm,
				}})
			}).
			POST("/settings", func(ctx *gin.Context) {
				var body types.CreateSettingRequestBody
				err := ctx.ShouldBindJSON(&body)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				db := db.GetDb()
				err = db.Transaction(func(tx *gorm.DB) error {
					setting := models.Setting{
						SettingKey:   body.Key,
						SettingValue: body.Value,
						Group:        body.Group,
					}
					err := tx.Create(&setting).Error
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.Status(http.StatusOK)
			}).
			GET("/settings", func(ctx *gin.Context) {
				var settings []models.Setting
				db := db.GetDb()
				err := db.Find(&settings).Error
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": settings})
			})

		authorized.
			POST("/stripe/onboarding", func(ctx *gin.Context) {
				userId := ctx.GetUint("id")
				var user models.User
				var accLinkURL string
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					err := tx.Model(&models.User{}).Where("id = ?", userId).First(&user).Error
					if err != nil {
						return err
					}
					sc := lib.GetStripeClient()
					acc, err := sc.V1Accounts.Create(context.Background(), &stripe.AccountCreateParams{
						Type:  stripe.String("express"),
						Email: stripe.String(user.Email),
					})
					if err != nil {
						return err
					}
					accLink, err := sc.V1AccountLinks.Create(context.Background(), &stripe.AccountLinkCreateParams{
						Account:    stripe.String(acc.ID),
						Type:       stripe.String("account_onboarding"),
						ReturnURL:  stripe.String(fmt.Sprint(os.Getenv("APP_HOST"), "/dashboard")),
						RefreshURL: stripe.String(fmt.Sprint(os.Getenv("APP_HOST"), "/callback/account/refresh")),
					})
					if err != nil {
						return err
					}
					err = tx.Model(&models.User{}).Where("id = ?", user.ID).Update("stripe_account_id", acc.ID).Error
					if err != nil {
						return err
					}
					accLinkURL = accLink.URL
					return nil
				})
				if err != nil {
					log.Printf("Error while setting up Stripe Account: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"url": accLinkURL})
			})

		authorized.
			GET("/keys", func(ctx *gin.Context) {
				bytes := make([]byte, 32)
				if _, err := rand.Read(bytes); err != nil {
					ctx.Status(http.StatusInternalServerError)
					return
				}

				key := hex.EncodeToString(bytes)
				ctx.JSON(http.StatusOK, gin.H{"key": key})
			}).
			POST("/encrypt", func(ctx *gin.Context) {
				var body EncryptRequestBody
				err := ctx.ShouldBindJSON(&body)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				key, _ := hex.DecodeString(body.Key)

				c, err := aes.NewCipher(key)
				if err != nil {
					log.Printf("Error creating cipher: %s\n", err.Error())
					ctx.Status(http.StatusInternalServerError)
					return
				}

				gcm, err := cipher.NewGCM(c)
				if err != nil {
					log.Printf("Error in GCM: %s\n", err.Error())
					ctx.Status(http.StatusInternalServerError)
					return
				}

				nonce := make([]byte, gcm.NonceSize())
				if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
					log.Printf("Error creating nonce: %s\n", err.Error())
					ctx.Status(http.StatusInternalServerError)
					return
				}

				plainTextBytes := []byte(body.PlainText)
				encryptedText := gcm.Seal(nonce, nonce, plainTextBytes, nil)

				ctx.JSON(http.StatusOK, gin.H{"encrypted_text": encryptedText})
			}).
			GET("/jwt", func(ctx *gin.Context) {})
	}

	if err := router.Run(":9090"); err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
}

type EncryptRequestBody struct {
	Key       string `json:"key" binding:"required"`
	PlainText string `json:"plain_text" binding:"required"`
}

func generateJWT(email string, uid uint, orgId uint) (string, error) {
	now := time.Now()
	expirationTime := now.Add(24 * time.Hour)
	permissionClaims := []string{
		"user:read",
		"user:update",
	}
	claims := &Claims{
		Permissions:  permissionClaims,
		Username:     email,
		Organization: orgId,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", uid),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(jwtKey)
}
