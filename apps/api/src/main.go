package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"ebs/src/boot"
	"ebs/src/config"
	"ebs/src/controllers"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/lib/mailer"
	"ebs/src/middlewares"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/covalenthq/lumberjack"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gookit/goutil/dump"
	"github.com/grokify/go-pkce"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
	"github.com/stripe/stripe-go/v82"
	engineiotypes "github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io/v2/socket"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
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

const (
	apiPrefix string = "/api/v1"
)

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
	router.Use(middlewares.SecureHeaders)
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
	apiv1 := g.Group(apiPrefix)
	return apiv1
}

func publicRoutes(g *gin.Engine) *gin.RouterGroup {
	apiv1 := apiv1Group(g)
	apiv1.
		GET("/share/:filename", func(ctx *gin.Context) {
			apiEnv := os.Getenv("API_ENV")
			if apiEnv != "local" {
				ctx.Status(http.StatusNotFound)
				return
			}
			var params struct {
				Filename string `uri:"filename" binding:"required"`
			}
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			assets := os.Getenv("TEMP_DIR")
			filePath := path.Join(assets, fmt.Sprintf("%s.jpeg", params.Filename))
			log.Printf("filePath: %s", filePath)
			ctx.File(filePath)
		})

	passkey := apiv1.Group("/passkey")
	passkey.
		POST("/login/start", func(ctx *gin.Context) {
			opts, status, err := controllers.PasskeyLoginStart(ctx.Copy())
			if err != nil {
				log.Printf("Error on PasskeyLoginStart: %s\n", err.Error())
				ctx.Status(status)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"publicKey": opts.Response})
		}).
		POST("/login/finish", func(ctx *gin.Context) {
			token, status, err := controllers.PasskeyLoginFinish(ctx.Copy())
			if err != nil {
				log.Printf("Error on PasskeyLoginFinish: %s\n", err.Error())
				ctx.Status(status)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{
				"token": token,
			})
		})

	oauthcb := apiv1.Group("/oauth")
	oauthcb.
		GET("/google/callback", func(ctx *gin.Context) {
			var query struct {
				State    *string `form:"state" binding:"required"`
				Code     *string `form:"code" binding:"required"`
				Scope    *string `form:"scope" binding:"required"`
				AuthUser int     `form:"authuser"`
				Prompt   string  `form:"prompt"`
			}
			if err := ctx.BindQuery(&query); err != nil {
				log.Printf("Error while parsing request params: %s\n", err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}
			dump.P(query)
			// Decrypt state
			key, err := hex.DecodeString(config.API_SECRET)
			if err != nil {
				log.Printf("Error while retrieving key: %s\n", err.Error())
				ctx.Status(http.StatusInternalServerError)
				return
			}
			dec, err := utils.DecryptMessage(key, *query.State)
			if err != nil {
				log.Printf("Error while decrypting message: %s\n", err.Error())
				ctx.Status(http.StatusInternalServerError)
				return
			}
			// Deserialize JSON
			var state types.Oauth2FlowState
			if err := json.Unmarshal([]byte(*dec), &state); err != nil {
				ctx.Status(http.StatusInternalServerError)
				return
			}
			dump.P(state)
			db := db.GetDb()
			var uc int64
			model := db.Model(&models.User{})
			if state.AccountType == "org" {
				model = db.Model(&models.Organization{})
			}
			if err := model.Where("id = ?", state.AccountID).Count(&uc).Error; err != nil {
				log.Printf("Error retrieving user info: %s\n", err.Error())
				ctx.Status(http.StatusInternalServerError)
				return
			}
			if uc == 0 {
				err := fmt.Errorf("could not find user with ID [%d]", state.AccountID)
				log.Printf("Error verifying user: %s\n", err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}
			// Decode nonce
			dnonce, err := hex.DecodeString(state.Nonce)
			if err != nil {
				log.Printf("Could not read nonce: %s\n", err.Error())
				ctx.Status(http.StatusInternalServerError)
				return
			}
			// Read generated nonce
			rd := lib.GetRedisClient()
			nonceKey := fmt.Sprintf("org::%d:oauth:nonce", state.AccountID)
			cache := rd.Get(context.Background(), nonceKey).Val()
			nonce, err := hex.DecodeString(cache)
			if err != nil {
				log.Printf("Error while decoding hex value: %s\n", err.Error())
				ctx.Status(http.StatusInternalServerError)
				return
			}
			// Subtle compare
			if subtle.ConstantTimeCompare(dnonce, nonce) != 1 {
				log.Println("Data mismatch: the supplied values do not match")
				ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Access denied"})
				return
			}
			oauthcfg := &oauth2.Config{
				RedirectURL:  config.API_HOST + "/api/v1/oauth/google/callback",
				ClientID:     config.OAUTH_CLIENT_ID,
				ClientSecret: config.OAUTH_CLIENT_SECRET,
				Scopes:       strings.Split(*query.Scope, " "),
				Endpoint:     google.Endpoint,
			}
			// Create code challenge and verifier
			cv := pkce.NewCodeVerifierBytes(nonce)
			token, err := oauthcfg.Exchange(
				context.Background(),
				*query.Code,
				oauth2.SetAuthURLParam(pkce.ParamCodeVerifier, cv),
			)
			if err != nil {
				log.Printf("Error while exchanging authorization code for token: %s\n", err.Error())
				ctx.Status(http.StatusInternalServerError)
				return
			}
			dump.P(token)
			go func() {
				t := &models.Token{
					RequestedBy:   state.AccountID,
					RequesterType: state.AccountType,
					Type:          "AccessToken",
					TokenName:     "calendar_token",
					TokenValue: types.JSONB{
						"access_token":  token.AccessToken,
						"refresh_token": token.RefreshToken,
						"exp":           token.Expiry,
						"ttl":           token.ExpiresIn,
					},
					TTL:    uint(token.ExpiresIn),
					Status: "active",
					Metadata: &types.Metadata{
						"state": query.State,
						"raw":   token,
					},
				}
				tx := db.Begin()
				if err := tx.Model(&models.Token{}).Where(&models.Token{
					Type:          "AccessToken",
					TokenName:     "calendar_token",
					RequestedBy:   state.AccountID,
					RequesterType: state.AccountType,
					Status:        "active",
				}).Update("status", "invalid").Error; err != nil {
					log.Printf("Error invalidating tokens: %s\n", err.Error())
					tx.Rollback()
					return
				}
				if err := tx.Create(t).Error; err != nil {
					log.Printf("Error saving token to database: %s\n", err.Error())
					tx.Rollback()
					return
				}
				tx.Commit()
			}()
			go func() {
				//Create a calendar named after the Account
				if state.AccountType == "org" {
					aid := state.AccountID
					var org models.Organization
					if err := db.Where(&models.Organization{ID: aid}).First(&org).Error; err != nil {
						log.Printf("Failed to retrieve information for Organization [%d]: %s\n", aid, err.Error())
						return
					}
					svc, err := lib.GAPICreateCalendarService(context.Background(), token, nil)
					if err != nil {
						log.Printf("Failed to create Calendar service: %s\n", err.Error())
						return
					}
					cal, err := lib.GAPIAddCalendar(org.Name, svc)
					if err != nil {
						log.Printf("Failed to create Calendar for [%s]: %s\n", org.Name, err.Error())
						return
					}
					evtId := strings.ReplaceAll(uuid.NewString(), "-", "")
					today := time.Now()
					err = lib.GAPIAddEvent(cal.Id, &calendar.Event{
						Id:      evtId,
						Summary: "Test",
						Start: &calendar.EventDateTime{
							DateTime: today.Add(time.Hour).Format("2006-01-02T15:04:05-0700"),
							TimeZone: "Asia/Manila",
						},
						End: &calendar.EventDateTime{
							DateTime: today.Add(5 * time.Hour).Format("2006-01-02T15:04:05-0700"),
							TimeZone: "Asia/Manila",
						},
						Description: "just a test",
						Attendees: []*calendar.EventAttendee{
							{
								Email:       org.ContactEmail,
								DisplayName: org.Name,
								Organizer:   true,
							},
						},
					}, svc)
					if err != nil {
						log.Printf("Failed to add Event to Calendar: %s\n", err.Error())
						return
					}
					log.Println("Event has been added to Calendar")
					tx := db.Begin()
					if err := tx.Model(&models.Organization{}).Where("id = ?", aid).Update("calendar_id", base64.RawURLEncoding.EncodeToString([]byte(cal.Id))).Error; err != nil {
						tx.Rollback()
						return
					}
					tx.Commit()
				}
			}()
			ex := time.Duration(token.ExpiresIn) * time.Second
			go rd.SetEx(context.Background(), fmt.Sprintf("%s::%d:calendar:token", state.AccountType, state.AccountID), token.AccessToken, ex)
			go rd.Del(context.Background(), nonceKey)
			ctx.Redirect(http.StatusTemporaryRedirect, state.Redirect)
		})
	return apiv1
}

func guestAuthRoutes(g *gin.Engine) *gin.RouterGroup {
	apiv1 := apiv1Group(g)
	guest := apiv1.Group("/auth")
	guest.Use(middlewares.VerifyIdToken)
	guest.
		POST("/login", func(ctx *gin.Context) {
			token, status, err := controllers.AuthLogin(ctx)
			if err != nil {
				log.Printf("[AuthLogin] error: %s\n", err.Error())
				ctx.Status(status)
				return
			}

			ctx.JSON(http.StatusOK, gin.H{
				"token": token,
			})
		}).
		POST("/register", func(ctx *gin.Context) {
			uid, status, err := controllers.AuthRegister(ctx)
			if err != nil {
				log.Printf("[AuthRegister] error: %s\n", err.Error())
				ctx.JSON(status, gin.H{"error": err.Error()})
				return
			}

			ctx.JSON(http.StatusOK, gin.H{"uid": uid})
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

type Country struct {
	Cca2      string   `json:"cca2"`
	Flag      string   `json:"flag"`
	Timezones []string `json:"timezones"`
	Name      struct {
		Common     string            `json:"common"`
		NativeName map[string]string `json:"nativeName"`
		Official   string            `json:"official"`
	} `json:"name"`
}

func cacheCountries() []Country {
	rd := lib.GetRedisClient()
	var rjson []Country
	val := rd.JSONGet(context.Background(), "countries").Val()
	if val != "" {
		json.Unmarshal([]byte(val), &rjson)
		return rjson
	}
	res, err := http.Get("https://restcountries.com/v3.1/all?fields=name,cca2,flag,timezones")
	if err != nil {
		log.Printf("Error response from API: %s\n", err.Error())
		return []Country{}
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading response: %s\n", err.Error())
		return []Country{}
	}
	json.Unmarshal(body, &rjson)
	sort.Slice(rjson, func(i, j int) bool {
		return rjson[i].Name.Common < rjson[j].Name.Common
	})
	rd.JSONSet(context.Background(), "countries", "$", rjson)

	return rjson
}

func initLogger() {
	cwd, _ := os.Getwd()
	serverLogs := path.Join(cwd, "logs", "server.log")
	apiLogs := path.Join(cwd, "logs", "api.log")
	gin.ForceConsoleColor()

	f, _ := os.Create(apiLogs)
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)
	log.SetOutput(&lumberjack.Logger{
		Filename:   serverLogs,
		MaxSize:    500,
		MaxBackups: 3,
		MaxAge:     30,
		Compress:   true,
	})
}

func main() {
	apiEnv := os.Getenv("API_ENV")
	if apiEnv == "local" {
		cwd, _ := os.Getwd()
		if err := godotenv.Load(path.Join(cwd, ".env")); err != nil {
			panic(err)
		}
	}
	initLogger()

	boot.InitDb()
	boot.InitScheduler()
	lib.InitWebAuthn(time.Hour, !utils.IsProd())

	go boot.DownloadSDKFileFromS3()
	go boot.DownloadServiceKeyFromS3()
	go lib.StripeInitialize()
	go boot.InitBroker()

	go cacheCountries()

	router := setupRouter()
	wss := setupSocketServer(router)
	if wss != nil {
		log.Println("WS server listening for connections...")
	}

	appHost := os.Getenv("APP_HOST")
	if apiEnv == "local" {
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

	publicRoutes(router)

	guestAuthRoutes(router)

	stripeWebhookRoute(router)

	authorized := router.Group(apiPrefix)
	authorized.Use(middlewares.AuthMiddleware)
	{
		authorized.GET("/countries", func(ctx *gin.Context) {
			countries := cacheCountries()
			ctx.JSON(http.StatusOK, gin.H{"countries": countries})
		})

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

		authorized = organizationHandlers(authorized)
		authorized = eventHandlers(authorized)
		authorized = ticketHandlers(authorized)
		authorized = bookingHandlers(authorized)
		authorized = reservationHandlers(authorized)
		authorized = admissionHandlers(authorized)
		authorized = transactionHandlers(authorized)

		authorized.
			GET("/users/me", func(ctx *gin.Context) {
				var user models.User
				userId := ctx.GetUint("id")
				db := db.GetDb()
				if err := db.Transaction(func(tx *gorm.DB) error {
					if err := tx.
						Where(&models.User{ID: userId}).
						First(&user).
						Error; err != nil {
						return err
					}
					return nil
				}); err != nil {
					ctx.Status(http.StatusBadRequest)
				}

				ctx.Status(http.StatusOK)
			}).
			PUT("/users/:id", func(ctx *gin.Context) {
				ctx.Status(http.StatusNoContent)
			})

		accounts := authorized.Group("/accounts")
		accounts.
			GET("/:id/calendar", func(ctx *gin.Context) {
				var params struct {
					AccountID uint `uri:"id" binding:"required"`
				}
				if err := ctx.ShouldBindUri(&params); err != nil {
					ctx.Status(http.StatusBadRequest)
					return
				}
				var query struct {
					AccountType string `form:"type" binding:"required"`
				}
				if err := ctx.ShouldBindQuery(&query); err != nil {
					ctx.Status(http.StatusBadRequest)
					return
				}
				var org models.Organization
				var tok models.Token
				db := db.GetDb()
				if err := db.Transaction(func(tx *gorm.DB) error {
					if err := tx.Model(&models.Organization{}).Where("id = ?", params.AccountID).First(&org).Error; err != nil {
						return nil
					}
					if err := tx.
						Where(&models.Token{
							Type:          "AccessToken",
							TokenName:     "calendar_token",
							RequesterType: query.AccountType,
							RequestedBy:   params.AccountID,
							Status:        "active",
						}).
						First(&tok).
						Error; err != nil {
						return nil
					}
					return nil
				}); err != nil {
					ctx.Status(http.StatusBadRequest)
					return
				}
				var token oauth2.Token
				val, err := json.Marshal(tok.TokenValue)
				if err != nil {
					ctx.Status(http.StatusInternalServerError)
					return
				}
				err = json.Unmarshal(val, &token)
				if err != nil {
					ctx.Status(http.StatusInternalServerError)
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"url": org.CalendarID})
			}).
			POST("/calendar/connect", func(ctx *gin.Context) {
				var body struct {
					Redirect string `json:"redirect" binding:"required"`
				}
				if err := ctx.ShouldBindJSON(&body); err != nil {
					ctx.Status(http.StatusBadRequest)
					return
				}

				orgId := ctx.GetUint("org")
				oauthcfg := &oauth2.Config{
					RedirectURL:  config.API_HOST + "/api/v1/oauth/google/callback",
					ClientID:     config.OAUTH_CLIENT_ID,
					ClientSecret: config.OAUTH_CLIENT_SECRET,
					Scopes: []string{
						calendar.CalendarCalendarsScope,
						calendar.CalendarEventsScope,
					},
					Endpoint: google.Endpoint,
				}
				// Generate nonce
				nonce := make([]byte, 32)
				rand.Read(nonce)
				hnonce := hex.EncodeToString(nonce)
				go func() {
					ex := 3600 * time.Second
					rd := lib.GetRedisClient()
					rd.SetEx(
						context.Background(),
						fmt.Sprintf("org::%d:oauth:nonce", orgId),
						hnonce,
						ex,
					)
				}()

				// Create code challenge and verifier
				cv := pkce.NewCodeVerifierBytes(nonce)
				cc := pkce.CodeChallengeS256(cv)

				// Build state
				state := &types.Oauth2FlowState{
					AccountID:   orgId,
					AccountType: "org",
					Nonce:       hnonce,
					Redirect:    body.Redirect,
				}
				// Serialize JSON
				b, err := json.Marshal(state)
				if err != nil {
					ctx.Status(http.StatusInternalServerError)
					return
				}
				keyBytes, err := hex.DecodeString(config.API_SECRET)
				if err != nil {
					log.Printf("Error while reading secret key: %s\n", err.Error())
					ctx.Status(http.StatusInternalServerError)
					return
				}
				enc, err := utils.EncryptMessage(keyBytes, string(b))
				if err != nil {
					log.Printf("Error while encrypting message: %s\n", err.Error())
					ctx.Status(http.StatusInternalServerError)
					return
				}
				authurl := oauthcfg.AuthCodeURL(
					enc,
					oauth2.AccessTypeOffline,
					oauth2.SetAuthURLParam(pkce.ParamCodeChallenge, cc),
					oauth2.SetAuthURLParam(pkce.ParamCodeChallengeMethod, pkce.MethodS256),
				)
				ctx.JSON(http.StatusOK, gin.H{"url": authurl})
			}).
			POST("/passkey/register/start", func(ctx *gin.Context) {
				opts, status, err := controllers.AccountsPasskeyRegisterStart(ctx.Copy())
				if err != nil {
					log.Printf("[AccountsPasskeyRegisterStart] error: %s\n", err.Error())
					ctx.Status(status)
					return
				}
				ctx.JSON(http.StatusOK, opts.Response)
			}).
			POST("/passkey/register/finish", func(ctx *gin.Context) {
				status, err := controllers.AccountsPasskeyRegisterFinish(ctx.Copy())
				if err != nil {
					log.Printf("[AccountsPasskeyRegisterFinish] error: %s\n", err.Error())
					ctx.Status(status)
					return
				}
				ctx.Status(http.StatusOK)
			}).
			GET("/devices", func(ctx *gin.Context) {
				userId := ctx.GetUint("id")
				devices, err := utils.GetCredentialsByUser(userId)
				if err != nil {
					ctx.Status(http.StatusBadRequest)
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": devices})
			}).
			PUT("/devices/revoke", func(ctx *gin.Context) {
				var body struct {
					DeviceName string `json:"name" binding:"required"`
				}
				if err := ctx.ShouldBindJSON(&body); err != nil {
					log.Printf("Error validating request: %s\n", err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				userId := ctx.GetUint("id")
				err := utils.RevokeCredential(userId, body.DeviceName)
				if err != nil {
					log.Printf("Error revoking credential: %s\n", err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				ctx.Status(http.StatusOK)
			}).
			POST("/verification/request_code", func(ctx *gin.Context) {
				var body struct {
					Email string `json:"email" binding:"required"`
				}
				if err := ctx.ShouldBindJSON(&body); err != nil {
					ctx.Status(http.StatusBadRequest)
					return
				}
				bi, err := rand.Int(rand.Reader, big.NewInt(999_999))
				if err != nil {
					ctx.Status(http.StatusInternalServerError)
					return
				}
				go func() {
					db := db.GetDb()
					if err := db.Transaction(func(tx *gorm.DB) error {
						var user models.User
						if err := tx.Where(&models.User{Email: body.Email}).Select("id").First(&user).Error; err != nil {
							return err
						}
						var del models.Token
						if err := tx.Model(&models.Token{}).Where("user_id = ? AND status = ?", user.ID, "pending").Update("status", "invalid").Error; err != nil {
							return err
						}
						if err := tx.Model(&models.Token{}).Where("user_id = ? AND status = ?", user.ID, "invalid").Delete(&del).Error; err != nil {
							return err
						}
						tok := &models.Token{
							RequestedBy: user.ID,
							Type:        "verification",
							TokenName:   "mfa_verification_code",
							TokenValue: types.JSONB{
								"code": bi,
							},
							TTL: 600,
						}
						if err := tx.Create(tok).Error; err != nil {
							return err
						}
						return nil
					}); err != nil {
						log.Printf("Error storing generated token: %s\n", err.Error())
					}
				}()
				if err := mailer.NewMailerMessage(&lib.SendMailInput{
					From:     config.SMTP_FROM,
					FromName: "noreply",
					Subject:  "Verify Authentication Code",
					To:       []string{body.Email},
					Body: fmt.Sprintf(`
					<p>You have requested a verification code</p>
					<p>Your verfication code: %d</p>
				`, bi),
					Html: true,
				}); err != nil {
					log.Printf("Could not send verification email to [%s]: %s\n", body.Email, err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				ctx.Status(http.StatusOK)
			}).
			POST("/verification/verify_code", func(ctx *gin.Context) {
				var body struct {
					Email string `json:"email" binding:"required"`
					Code  string `json:"code" binding:"required"`
				}
				if err := ctx.ShouldBindJSON(&body); err != nil {
					log.Printf("Error validating request: %s\n", err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				var token models.Token
				db := db.GetDb()
				tx := db.Begin()
				var user models.User
				if err := tx.
					Model(&models.User{}).
					Where("email = ?", body.Email).
					First(&user).
					Error; err != nil {
					tx.Rollback()
					log.Printf("Error retrieving user [%s]: %s\n", body.Email, err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				if err := tx.
					Model(&models.Token{}).
					Where("user_id = ? AND token_name = ? AND token_value ->> 'code' = ?", user.ID, "mfa_verification_code", body.Code).
					First(&token).
					Error; err != nil {
					tx.Rollback()
					log.Printf("Error retrieving token: %s\n", err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				if token.ExpiresAt.Before(time.Now()) {
					tx.Rollback()
					if err := tx.Model(&models.Token{}).Where("id = ?", token.ID).Update("status", "expired").Error; err != nil {
						tx.Rollback()
						log.Printf("Error updating expired token: %s\n", err.Error())
						ctx.Status(http.StatusBadRequest)
						return
					}
					err := errors.New("code has expired")
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				if err := tx.Model(&models.Token{}).Where("id = ?", token.ID).Update("status", "done").Error; err != nil {
					tx.Rollback()
					log.Printf("Error updating token status: %s\n", err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				tx.Commit()
				ctx.Status(http.StatusOK)
			}).
			POST("/verify", func(ctx *gin.Context) {
				status, err := controllers.AccountsVerify(ctx.Copy())
				if err != nil {
					log.Printf("[AccountsVerify] error: %s\n", err.Error())
					ctx.Status(status)
					return
				}
				ctx.Status(http.StatusOK)
			})

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
						Select("id", "name", "email", "phone", "email_verified", "phone_verified").
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
			})
	}

	if os.Getenv("TLS_ENABLE") == "true" {
		cwd, _ := os.Getwd()
		certpath := path.Join(cwd, "certificates", "localhost.pem")
		keypath := path.Join(cwd, "certificates", "localhost-key.pem")
		if err := router.RunTLS(":9090", certpath, keypath); err != nil {
			log.Fatalf("Failed to start server: %s", err)
		}
	}
	if err := router.Run(":9090"); err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
}

type EncryptRequestBody struct {
	Key       string `json:"key" binding:"required"`
	PlainText string `json:"plain_text" binding:"required"`
}
