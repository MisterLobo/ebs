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

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stripe/stripe-go/v82"
	"gorm.io/gorm"
)

type Claims struct {
	Username     string   `json:"username"`
	Role         string   `json:"role"`
	Permissions  []string `json:"permissions"`
	Organization uint
	jwt.RegisteredClaims
}

var jwtKey = []byte(os.Getenv("JWT_SECRET"))
var tokens []string

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
			err := errors.New("Server is under maintenance")
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
	guest.Use(func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("origin")
		log.Printf("[origin]: %s\n", origin)
		secret := ctx.Request.Header.Get("x-secret")
		realSecret := os.Getenv("API_SECRET")
		log.Printf("[secret]: %s %s\n", secret, realSecret)
		if secret != realSecret {
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		}
		appHost := os.Getenv("APP_HOST")
		match, _ := regexp.MatchString(appHost, origin)
		if match {
			return
		}
		log.Printf("Origin matches host: %v %s\n", match, origin)
		match, _ = regexp.MatchString(`app:mobile`, origin)
		if match {
			return
		}
		log.Printf("Origin matches mobile: %v %s\n", match, origin)
		ctx.AbortWithStatus(http.StatusNotFound)
	})
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
			tokens = append(tokens, token)

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
				err := db.
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
				err = db.Create(&newUser).Error
				if err != nil {
					return err
				}

				newOrg := models.Organization{
					Name:         fmt.Sprintf("%s's organization", user.DisplayName),
					OwnerID:      newUser.ID,
					Type:         types.ORG_PERSONAL,
					ContactEmail: user.Email,
					TenantID:     newUser.TenantID,
				}
				err = db.Create(&newOrg).Error
				if err != nil {
					return err
				}

				newTeam := models.Team{
					OrganizationID: newOrg.ID,
					OwnerID:        newUser.ID,
					Name:           "Default",
					Status:         "active",
				}
				err = db.Create(&newTeam).Error
				if err != nil {
					return err
				}

				err = db.
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

func main() {
	go boot.DownloadSDKFileFromS3()
	go boot.InitDb()
	go boot.InitBroker()
	go boot.InitScheduler()

	router := setupRouter()

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
			if match {
				return true
			}
			return false
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
				log.Printf("content: %s\n", res)
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
						SettingValue: types.JSONBAny{Inner: body.Value},
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
						ReturnURL:  stripe.String(""),
						RefreshURL: stripe.String(""),
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
			})
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
