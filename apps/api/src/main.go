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
	awslib "ebs/src/lib/aws"
	"ebs/src/middlewares"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
	"github.com/tidwall/gjson"
	"github.com/yeqown/go-qrcode"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func main() {
	go boot.DownloadSDKFileFromS3()
	go boot.InitDb()
	go boot.InitBroker()
	go boot.InitScheduler()

	router := gin.Default()

	appEnv := os.Getenv("APP_ENV")
	appHost := os.Getenv("APP_HOST")
	if appEnv == "local" {
		router.Use(cors.Default())
	} else {
		cc := cors.DefaultConfig()
		cc.AllowMethods = append(cc.AllowMethods, "GET", "POST", "PATCH", "PUT", "DELETE", "HEAD")
		cc.AllowHeaders = append(cc.AllowHeaders, "Origin")
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

	router.Use(func(ctx *gin.Context) {
		mm := os.Getenv("MAINTENANCE_MODE")
		atoi, err := strconv.ParseBool(mm)
		if err != nil || atoi {
			err := errors.New("Server is under maintenance")
			log.Println(err.Error())
			ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, err.Error())
			return
		}
	})

	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "ok")
	})

	apiv1 := router.Group("/api/v1")
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
				Select("id").
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
				_, err = rd.JSONSet(ctx, fmt.Sprintf("%d:user", muser.ID), "$", muser).Result()
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

	apiv1.POST("/webhook/stripe", func(ctx *gin.Context) {
		payload := make([]byte, 65536)
		payload, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			log.Printf("Error reading request body: %s\n", err.Error())
			ctx.Status(http.StatusServiceUnavailable)
			return
		}
		whsecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		event, err := webhook.ConstructEvent(payload, ctx.GetHeader("Stripe-Signature"), whsecret)
		if err != nil {
			log.Printf("Error verifying webhook signature: %s\n", err.Error())
			ctx.Status(http.StatusBadRequest)
			return
		}
		log.Printf("[StripeEvent] %s\n", event.Type)
		switch event.Type {
		case "customer.created":
			var cus stripe.Customer
			err := json.Unmarshal(event.Data.Raw, &cus)
			if err != nil {
				log.Printf("[Stripe] Error parsing Customer: %s\n", err.Error())
				break
			}
			id := cus.Metadata["id"]
			atoi, err := strconv.Atoi(id)
			if err != nil {
				log.Printf("Could not retrieve user id for customer %s: %s\n", cus.ID, err.Error())
				break
			}
			userId := uint(atoi)
			db := db.GetDb()
			err = db.Transaction(func(tx *gorm.DB) error {
				var user models.User
				if err := tx.
					Model(&models.User{}).
					Where("id = ?", userId).
					Find(&user).
					Error; err != nil {
					log.Printf("Error while retrieving user info for Customer %s: %s\n", cus.ID, err.Error())
					return errors.New("Could not retrieve user information")
				}

				if err := tx.
					Model(&models.User{}).
					Where("id = ?", userId).
					Updates(&models.User{StripeCustomerId: &cus.ID}).
					Error; err != nil {
					log.Printf("Error updating user: %s\n", err.Error())
					return err
				}
				return nil
			})
			if err != nil {
				log.Printf("Error updating user %d: %s\n", userId, err.Error())
			}
			break
		case "account.updated":
			var acc stripe.Account
			err := json.Unmarshal(event.Data.Raw, &acc)
			if err != nil {
				log.Printf("[Stripe] Error parsing Account: %s\n", err.Error())
				break
			}
			break
		case "capability.updated":
			var cap stripe.Capability
			err := json.Unmarshal(event.Data.Raw, &cap)
			if err != nil {
				log.Printf("[Stripe] Error parsing Capability: %s\n", err.Error())
				break
			}
			break
		case "payment_intent.created":
			var pi stripe.PaymentIntent
			err := json.Unmarshal(event.Data.Raw, &pi)
			if err != nil {
				log.Printf("[Stripe] Error parsing PaymentIntent: %s\n", err.Error())
				break
			}
			log.Printf("[PaymentIntent] ID: %s %s\n", pi.ID, pi.Status)
			md := pi.Metadata
			log.Printf("[%s] Metadata: %v\n", pi.ID, md)
			requestId := md["requestId"]
			go func() {
				var txn models.Transaction
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					err := tx.
						Model(&models.Transaction{}).
						Where("reference_id = ?", requestId).
						First(&txn).
						Error
					if err != nil {
						return err
					}
					err = tx.
						Model(&models.Booking{}).
						Where("transaction_id = ?", &txn.ID).
						Updates(&models.Booking{
							Status:          types.BOOKING_COMPLETED,
							PaymentIntentId: &pi.ID,
							// TransactionID:   &txn.ID,
						}).
						Error
					if err != nil {
						log.Printf("Error updating Booking group [%s]: %s\n", requestId, err.Error())
						return err
					}
					cli := lib.AWSGetSQSClient()
					qurl, err := cli.GetQueueUrl(context.Background(), &sqs.GetQueueUrlInput{
						QueueName: aws.String("PaymentTransactionUpdates"),
					})
					bUpdates, _ := json.Marshal(&models.Transaction{
						SourceName:  "PaymentIntent",
						SourceValue: pi.ID,
						Status:      types.TRANSACTION_PROCESSING,
						Amount:      float64(pi.Amount),
						Currency:    string(pi.Currency),
					})
					updates := string(bUpdates)
					bConds, _ := json.Marshal(&models.Transaction{
						ID:     txn.ID,
						Status: types.TRANSACTION_PENDING,
					})
					conds := string(bConds)
					bPayload, _ := json.Marshal(map[string]any{
						"source":  "payment_intent.created",
						"id":      txn.ID.String(),
						"conds":   conds,
						"updates": updates,
					})
					sPayload := string(bPayload)
					out, err := cli.SendMessage(context.Background(), &sqs.SendMessageInput{
						QueueUrl:    qurl.QueueUrl,
						MessageBody: aws.String(sPayload),
					})
					if err != nil {
						log.Printf("Could not send message to queue: %s\n", err.Error())
						return err
					}
					log.Printf("Message sent to queue: %s\n", *out.MessageId)
					/* err = tx.
						Where(&models.Transaction{ReferenceID: requestId, Status: types.TRANSACTION_PENDING}).
						Updates(&models.Transaction{
							SourceName:  "PaymentIntent",
							SourceValue: pi.ID,
							Status:      types.TRANSACTION_PROCESSING,
							Amount:      float64(pi.Amount),
							Currency:    string(pi.Currency),
						}).
						Error
					if err != nil {
						return err
					} */
					return nil
				})
				if err != nil {
					log.Printf("Error processing Transaction: %s\n", err.Error())
					return
				}
			}()
			break
		case "payment_intent.succeeded":
			var pi stripe.PaymentIntent
			err := json.Unmarshal(event.Data.Raw, &pi)
			if err != nil {
				log.Printf("[Stripe] Error parsing PaymentIntent: %s\n", err.Error())
				break
			}
			log.Printf("[PaymentIntent] ID: %s %s\n", pi.ID, pi.Status)
			md := pi.Metadata
			log.Printf("[%s] Metadata: %v\n", pi.ID, md)
			requestId := md["requestId"]
			go func() {
				var txn models.Transaction
				var bookings []models.Booking
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					err := tx.
						Model(&models.Transaction{}).
						Where("reference_id = ?", requestId).
						First(&txn).
						Error
					if err != nil {
						return err
					}
					err = tx.
						Model(&models.Booking{}).
						Where("metadata ->> 'requestId' = ?", requestId).
						Preload("Event").
						Find(&bookings).
						Error
					if err != nil {
						return err
					}
					err = tx.
						Model(&models.Booking{}).
						Where("metadata ->> 'requestId' = ?", requestId).
						Updates(&models.Booking{
							Status:          types.BOOKING_COMPLETED,
							PaymentIntentId: &pi.ID,
						}).Error
					if err != nil {
						log.Printf("Error updating Booking group [%s]: %s\n", requestId, err.Error())
						return err
					}
					for _, booking := range bookings {
						err := tx.
							Model(&models.Reservation{}).
							Where("booking_id = ?", booking.ID).
							Preload("Booking").
							Updates(&models.Reservation{
								Status:     string(types.RESERVATION_PAID),
								ValidUntil: booking.Event.DateTime,
							}).
							Error
						if err != nil {
							return err
						}
					}
					cli := lib.AWSGetSQSClient()
					qurl, err := cli.GetQueueUrl(context.Background(), &sqs.GetQueueUrlInput{
						QueueName: aws.String("PaymentTransactionUpdates"),
					})
					bUpdates, _ := json.Marshal(models.Transaction{
						SourceName:  "PaymentIntent",
						SourceValue: pi.ID,
						Status:      types.TRANSACTION_COMPLETED,
						Amount:      float64(pi.Amount),
						Currency:    string(pi.Currency),
					})
					updates := string(bUpdates)
					bConds, _ := json.Marshal(&models.Transaction{
						ID:     txn.ID,
						Status: types.TRANSACTION_PROCESSING,
					})
					conds := string(bConds)
					bPayload, _ := json.Marshal(&map[string]any{
						"source":  "payment_intent.succeeded",
						"id":      txn.ID.String(),
						"conds":   conds,
						"updates": updates,
					})
					sPayload := string(bPayload)
					out, err := cli.SendMessage(context.Background(), &sqs.SendMessageInput{
						QueueUrl:     qurl.QueueUrl,
						MessageBody:  aws.String(sPayload),
						DelaySeconds: 10,
					})
					if err != nil {
						log.Printf("Could not send message to queue: %s\n", err.Error())
						return err
					}
					log.Printf("Message sent to queue: %s\n", *out.MessageId)
					/* err = tx.
						Where(&models.Transaction{ReferenceID: requestId, Status: types.TRANSACTION_PROCESSING}).
						Updates(&models.Transaction{
							Status: types.TRANSACTION_COMPLETED,
						}).
						Error
					if err != nil {
						return err
					} */
					return nil
				})
				if err != nil {
					log.Printf("Error processing Transaction: %s\n", err.Error())
					return
				}
			}()
			break
		case "checkout.session.completed":
			var cs stripe.CheckoutSession
			err := json.Unmarshal(event.Data.Raw, &cs)
			if err != nil {
				log.Printf("[Stripe] Error parsing CheckoutSession: %s\n", err.Error())
				break
			}
			log.Printf("[CheckoutSession] ID: %s %s\n", cs.ID, cs.Status)
			md := cs.Metadata
			log.Printf("[%s] Metadata: %v\n", cs.ID, md)
			requestId := md["requestId"]
			go func() {
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					err := tx.
						Model(&models.Booking{}).
						Where("metadata ->> 'requestId' = ?", requestId).
						Updates(&models.Booking{
							CheckoutSessionId: &cs.ID,
						}).
						Error
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					log.Printf("Error updating Booking records: %s\n", err.Error())
					return
				}
			}()
			break
		}
		ctx.Status(http.StatusNoContent)
	})

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

		authorized.
			GET("/organizations", func(ctx *gin.Context) {
				var filters types.OrganizationsQueryFilters
				if err := ctx.ShouldBindQuery(&filters); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				orgs := make([]models.Organization, 0)
				userId := ctx.GetUint("id")
				conds := &models.Organization{}
				if filters.Type == types.ORG_STANDARD || filters.Type == types.ORG_PERSONAL {
					conds.Type = filters.Type
				}
				role := ctx.GetString("role")
				if filters.Owned {
					conds.OwnerID = userId
				} else {
					if role != string(types.ROLE_ADMIN) {
						conds.OwnerID = userId
					}
				}
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					err := tx.
						Select("id", "name", "type", "contact_email", "stripe_account_id", "connect_onboarding_url", "status").
						Where(conds).
						Order("created_at desc").
						Find(&orgs).
						Error
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": orgs})
			}).
			GET("/organizations/:orgId", func(ctx *gin.Context) {
				orgIdParam := ctx.Params.ByName("orgId")
				atoi, err := strconv.Atoi(orgIdParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				orgId := uint(atoi)
				var org models.Organization
				db := db.GetDb()
				err = db.Where(&models.Organization{ID: orgId}).First(&org).Error
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": org})
			}).
			GET("/organizations/about", func(ctx *gin.Context) {
				var query struct {
					Slug *string `form:"slug"`
					ID   *uint   `form:"id"`
				}
				if err := ctx.ShouldBindQuery(&query); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				var org models.Organization
				db := db.GetDb()
				if query.Slug != nil {
					db = db.Where(&models.Organization{Slug: *query.Slug})
				}
				if query.ID != nil {
					db = db.Where(&models.Organization{ID: *query.ID})
				}
				if err := db.
					Omit("ConnectOnboardingURL", "StripeAccountID", "OwnerID").
					First(&org).
					Error; err != nil {
					if errors.Is(gorm.ErrRecordNotFound, err) {
						ctx.Status(http.StatusNotFound)
						return
					}
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": org})
			}).
			POST("/organizations/:orgId/switch", func(ctx *gin.Context) {
				orgIdParam := ctx.Params.ByName("orgId")
				atoi, err := strconv.Atoi(orgIdParam)
				if err != nil {
					log.Printf("Error switching to organization %s: %s\n", orgIdParam, err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				userId := ctx.GetUint("id")
				orgId := uint(atoi)
				var org models.Organization
				db := db.GetDb()
				err = db.Transaction(func(tx *gorm.DB) error {
					err := db.Where(&models.Organization{ID: orgId, OwnerID: userId}).First(&org).Error
					if err != nil {
						return err
					}
					err = tx.
						Model(&models.User{}).
						Where(&models.User{ID: userId}).
						Update("active_org", orgId).
						Error
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					log.Printf("Error switching to organization %d: %s\n", orgId, err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				if org.Type != types.ORG_STANDARD {
					err := errors.New("Switching to a non-Standard organization type is not allowed")
					log.Printf("Error switching to organization: %s\n", err.Error())
					ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
					return
				}
				tokenCookie, err := ctx.Cookie("token")
				email := ctx.GetString("email")
				if err != nil {
					tokenCookie, err = generateJWT(email, userId, orgId)
				}
				if err != nil {
					log.Printf("Error switching to organization %d: %s\n", orgId, err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "Error switching to organization"})
					return
				}
				rd := lib.GetRedisClient()
				err = rd.JSONSet(context.Background(), fmt.Sprintf("%d:active", userId), "$", org).Err()
				if err != nil {
					log.Printf("Error update cache: %s\n", err.Error())
					ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
					return
				}
				sc := lib.GetStripeClient()
				accountId := org.StripeAccountID
				if accountId == nil {
					log.Printf("Error while retrieving account information for Organization [%d]: account is not set up\n", orgId)
					ctx.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
					return
				}
				accId := *accountId
				log.Println("AccountID:", org.ID, accId)
				stripeAccount, err := sc.V1Accounts.GetByID(context.Background(), accId, nil)
				if err != nil {
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				completed := stripeAccount != nil && len(stripeAccount.Requirements.Errors) == 0 &&
					stripeAccount.ChargesEnabled &&
					stripeAccount.PayoutsEnabled &&
					stripeAccount.DetailsSubmitted

				onboarding_status := "incomplete"
				if completed {
					onboarding_status = "complete"
				}
				onboardingStatus := map[string]any{
					"url":              org.ConnectOnboardingURL,
					"accountId":        stripeAccount.ID,
					"status":           onboarding_status,
					"errors":           stripeAccount.Requirements.Errors,
					"chargesEnabled":   stripeAccount.ChargesEnabled,
					"payoutsEnabled":   stripeAccount.PayoutsEnabled,
					"detailsSubmitted": stripeAccount.DetailsSubmitted,
				}
				jbytes, _ := json.Marshal(&onboardingStatus)
				value := string(jbytes)
				key := fmt.Sprintf("%d:onboarding_status", userId)
				_, err = rd.JSONSet(context.Background(), key, "$", value).Result()
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				rd.Expire(context.Background(), key, time.Hour)
				appEnv := os.Getenv("APP_ENV")
				secure := appEnv == "prod"
				ctx.SetCookie("token", tokenCookie, 3600, "/", os.Getenv("APP_HOST"), secure, true)
				ctx.JSON(http.StatusOK, gin.H{"access_token": tokenCookie})
			}).
			GET("/organizations/:orgId/reservations", func(ctx *gin.Context) {
				idParam := ctx.Params.ByName("orgId")
				atoi, err := strconv.Atoi(idParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "The supplied ID is not a valid format"})
					return
				}
				db := db.GetDb()
				userId := ctx.GetUint("id")
				claims := ctx.GetStringSlice("perms")
				log.Printf("claims: %v\n", claims)
				allowed := slices.IndexFunc(claims, func(c string) bool { return c == "reservations:list" || c == "reservations*" || c == "*" }) > -1
				if !allowed {
					err := errors.New("Not enough permissions to perform this action")
					ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
					return
				}
				var user models.User
				err = db.
					Model(&models.User{}).
					Where("id = ?", userId).
					First(&user).
					Error
				if err != nil {
					log.Println(err.Error())
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				orgId := uint(atoi)
				var org models.Organization
				err = db.
					Model(&models.Organization{}).
					Where(&models.Organization{ID: orgId, OwnerID: userId}).
					First(&org).
					Error
				if err != nil {
					log.Println(err.Error())
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				data, err := utils.GetOrgReservations(orgId)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": data})
			}).
			GET("/organizations/:orgId/bookings", func(ctx *gin.Context) {
				var params types.SimpleOrganizationRequestParams
				if err := ctx.ShouldBindUri(&params); err != nil {
					log.Printf("Error while validating request: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				db := db.GetDb()
				orgId := params.ID
				var org models.Organization
				if err := db.Model(&models.Organization{}).Where("id = ?", orgId).First(&org).Error; err != nil {
					log.Printf("Error retrieving Organization [%d]: %s\n", orgId, err.Error())
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				var bookings []models.Booking
				var events int64
				if err := db.
					Model(&models.Event{}).
					Preload("Organization", "organizer_id = ?", orgId).
					Where(&models.Event{OrganizerID: orgId}).
					Count(&events).
					Error; err != nil {
					ctx.Status(http.StatusBadRequest)
					return
				}
				if events == 0 {
					log.Printf("No events for Organization [%d]\n", orgId)
					ctx.JSON(http.StatusOK, gin.H{"data": []map[string]any{}})
					return
				}
				log.Printf("Events: %d\n", events)
				sub := db.Preload("Organization").Where(&models.Event{OrganizerID: orgId})
				if err := db.
					Joins("Event", sub).
					// Preload("Event").
					Preload("Ticket").
					Preload("User").
					Order("created_at DESC").
					Find(&bookings).
					Error; err != nil {
					log.Printf("Error retrieving Booking: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": bookings})
			}).
			GET("/organizations/:orgId/events", func(ctx *gin.Context) {
				var query struct {
					Public bool `form:"public,omitempty"`
				}
				if err := ctx.ShouldBindQuery(&query); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				id := ctx.Params.ByName("orgId")
				atoi, err := strconv.Atoi(id)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "The supplied ID is not a valid format"})
					return
				}
				orgId := uint(atoi)
				events := make([]models.Event, 0)
				var organization models.Organization
				db := db.GetDb()
				if query.Public {
					if err := db.Where(&models.Organization{ID: orgId}).First(&organization).Error; err != nil {
						log.Printf("Error retrieving Organization [%d]: %s\n", orgId, err.Error())
						if errors.Is(gorm.ErrRecordNotFound, err) {
							ctx.Status(http.StatusNotFound)
							return
						}
						ctx.Status(http.StatusBadRequest)
						return
					}
					if err := db.
						Where(&models.Event{OrganizerID: orgId}).
						Where("status IN (?)", []types.EventStatus{
							types.EVENT_TICKETS_NOTIFY,
							types.EVENT_REGISTRATION,
						}).
						Where("date_time > ?", time.Now()).
						Find(&events).
						Limit(100).
						Order("date_time DESC").
						Error; err != nil {
						if len(events) == 0 {
							ctx.JSON(http.StatusOK, gin.H{"data": events})
							return
						}
						ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
						return
					}
					ctx.JSON(http.StatusOK, gin.H{"data": events})
					return
				}
				userId := ctx.GetUint("id")
				if err := db.
					Where(&models.Organization{ID: orgId, OwnerID: userId}).
					Find(&organization).
					Error; err != nil {
					if errors.Is(gorm.ErrRecordNotFound, err) {
						err := errors.New("Organization not found")
						ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
						return
					}
				}

				activeOrgId := ctx.GetUint("org")
				if activeOrgId != orgId {
					err := errors.New("Event created must be for active organization")
					ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
					return
				}
				db.
					Where(&models.Event{OrganizerID: orgId}).
					Limit(100).
					Order("date_time DESC").
					Find(&events)

				ctx.JSON(http.StatusOK, gin.H{"data": events, "count": len(events)})
			}).
			GET("/organizations/:orgId/events/:eventId", func(ctx *gin.Context) {
				orgIdParam := ctx.Params.ByName("orgId")
				atoi, err := strconv.Atoi(orgIdParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "The supplied ID is not a valid format"})
					return
				}
				orgId := uint(atoi)

				eventIdParam := ctx.Params.ByName("eventId")
				atoi, err = strconv.Atoi(eventIdParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "The supplied ID is not a valid format"})
					return
				}
				eventId := uint(atoi)

				userId := ctx.GetUint("id")
				var organization models.Organization
				db := db.GetDb()
				db.
					Where(&models.Organization{ID: orgId, OwnerID: userId}).
					First(&organization)
				if organization.ID < 1 {
					err := errors.New("Organization not found")
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				activeOrgId := ctx.GetUint("org")
				if activeOrgId != orgId {
					err := errors.New("Event created must be for active organization")
					ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
					return
				}
				var event models.Event
				db.
					Where(&models.Event{ID: eventId}).
					First(&event)

				ctx.JSON(http.StatusOK, gin.H{"data": event})
			}).
			GET("/organizations/:orgId/tickets", func(ctx *gin.Context) {
				var params types.SimpleOrganizationRequestParams
				if err := ctx.ShouldBindUri(&params); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				var org models.Organization
				db := db.GetDb()
				if err := db.Model(&models.Organization{}).Where("id = ?", params.ID).First(&org).Error; err != nil {
					log.Printf("Error retrieving Organization: %s\n", err.Error())
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				var events int64
				if err := db.Model(&models.Event{}).Where("organizer_id = ?", params.ID).Count(&events).Error; err != nil {
					log.Printf("Error retrieving Events: %s\n", err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				if events == 0 {
					log.Printf("No events for Organization [%d]\n", params.ID)
					ctx.JSON(http.StatusOK, gin.H{"data": []map[string]any{}})
					return
				}
				sub := db.Model(&models.Event{}).Preload("Organization", "id = ?", params.ID).Where(&models.Event{OrganizerID: params.ID})
				var tickets []models.Ticket
				if err := db.
					Model(&models.Ticket{}).
					// Preload("Event", "organizer_id = ?", params.ID).
					Joins("Event", sub).
					Find(&tickets).
					Error; err != nil {
					log.Printf("Error retrieving Tickets for Organization [%d]: %s\n", params.ID, err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": tickets})
			}).
			GET("/organizations/:orgId/tickets/sold", func(ctx *gin.Context) {
				var params types.SimpleOrganizationRequestParams
				if err := ctx.ShouldBindUri(&params); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				var org models.Organization
				var sales map[string]any
				db := db.GetDb()
				if err := db.Model(&models.Organization{}).Where("id = ?", params.ID).First(&org).Error; err != nil {
					log.Printf("Error retrieving Organization: %s\n", err.Error())
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				now := time.Now()
				past1month := now.AddDate(0, 0, -30)
				bookingSubquery := db.
					Model(&models.Booking{}).
					Select("SUM(subtotal) as total_revenue", "SUM(qty) as total_sold").
					Where("status = ?", "completed").
					Where("created_at BETWEEN ? AND ?", past1month, now)
				if err := bookingSubquery.
					Scan(&sales).
					Error; err != nil {
					log.Printf("Error executing query: %s\n", err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": map[string]any{
					"currency":  "usd",
					"sales":     sales,
					"from_date": past1month,
					"to_date":   now,
				}})
			}).
			GET("/organizations/:orgId/admissions", func(ctx *gin.Context) {
				var params types.SimpleOrganizationRequestParams
				if err := ctx.ShouldBindUri(&params); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				var admissions []models.Admission
				db := db.GetDb()
				if err := db.Transaction(func(tx *gorm.DB) error {
					var org models.Organization
					if err := tx.
						Model(&models.Organization{}).
						Where("id = ?", params.ID).
						First(&org).
						Error; err != nil {
						log.Printf("Could not retrieve Organization [%d]\n", params.ID)
						return err
					}
					var eventIDs []uint
					if err := tx.
						Model(&models.Event{}).
						Where(&models.Event{OrganizerID: params.ID}).
						Select("id").
						Pluck("id", &eventIDs).
						Error; err != nil {
						log.Printf("Error on finding Events: %s\n", err.Error())
						return err
					}
					var bookingIDs []uint
					if err := tx.
						Model(&models.Booking{}).
						Where("event_id IN (?)", eventIDs).
						Select("id").
						Pluck("id", &bookingIDs).
						Error; err != nil {
						log.Printf("Error on finding Bookings: %s\n", err.Error())
						return err
					}
					var ticketIDs []uint
					if err := tx.
						Model(&models.Ticket{}).
						Where("event_id IN (?)", eventIDs).
						Select("id").
						Pluck("id", &ticketIDs).
						Error; err != nil {
						log.Printf("Error on finding Tickets: %s\n", err.Error())
						return err
					}
					var resIDs []uint
					if err := tx.
						Model(&models.Reservation{}).
						Where("ticket_id IN (?)", ticketIDs).
						Where("booking_id IN (?)", bookingIDs).
						Select("id").
						Pluck("id", &resIDs).
						Error; err != nil {
						log.Printf("Error on finding Reservations: %s\n", err.Error())
						return err
					}
					if err := tx.
						Model(&models.Admission{}).
						Where("reservation_id IN (?)", resIDs).
						Preload("Reservation").
						Preload("Reservation.Ticket").
						Preload("Reservation.Ticket.Event").
						Limit(100).
						Order("created_at DESC").
						Find(&admissions).
						Error; err != nil {
						log.Printf("Error on finding Admissions: %s\n", err.Error())
						return err
					}
					return nil
				}); err != nil {
					log.Printf("Error retrieving Admissions: %s\n", err.Error())
					if errors.Is(err, gorm.ErrRecordNotFound) {
						ctx.Status(http.StatusNotFound)
						return
					}
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "There was an error"})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": admissions})
			}).
			GET("/admissions/:id", func(ctx *gin.Context) {
				var params types.SimpleRequestParams
				if err := ctx.ShouldBindUri(&params); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				orgId := ctx.GetUint("org")
				var res models.Reservation
				var adm models.Admission
				db := db.GetDb()
				if err := db.
					Where(&models.Admission{ID: params.ID}).
					Preload("Reservation").
					Preload("Reservation.Booking").
					Preload("Reservation.Ticket").
					First(&adm).
					Error; err != nil {
					log.Printf("Error retrieving Admission [%d]: %s\n", params.ID, err.Error())
					if errors.Is(err, gorm.ErrRecordNotFound) {
						ctx.Status(http.StatusNotFound)
						return
					}
					ctx.Status(http.StatusNotFound)
					return
				}
				if err := db.
					Where(&models.Reservation{ID: adm.ReservationID}).
					First(&res).
					Error; err != nil {
					log.Printf("Error retrieving Reservation for Admission [%d]: %s\n", params.ID, err.Error())
					if errors.Is(err, gorm.ErrRecordNotFound) {
						ctx.Status(http.StatusNotFound)
						return
					}
					ctx.Status(http.StatusBadRequest)
					return
				}
				var ticket models.Ticket
				if err := db.
					Where(&models.Ticket{ID: res.TicketID}).
					Preload("Event", "organizer_id = ?", orgId).
					First(&ticket).
					Error; err != nil {
					log.Printf("Error retrieving Ticket for Admission [%d]: %s\n", params.ID, err.Error())
					if errors.Is(err, gorm.ErrRecordNotFound) {
						ctx.Status(http.StatusNotFound)
						return
					}
					ctx.Status(http.StatusBadRequest)
					return
				}
				var evt models.Event
				if err := db.Where(&models.Event{ID: ticket.EventID, OrganizerID: orgId}).First(&evt).Error; err != nil {
					log.Printf("Error retrieving Event for Admission [%d]: %s\n", params.ID, err.Error())
					if errors.Is(err, gorm.ErrRecordNotFound) {
						ctx.Status(http.StatusNotFound)
						return
					}
					ctx.Status(http.StatusBadRequest)
					return
				}

				ctx.JSON(http.StatusOK, gin.H{"data": adm})
			}).
			GET("/organizations/:orgId/events/:eventId/tickets", func(ctx *gin.Context) {
				orgIdParam := ctx.Params.ByName("orgId")
				atoi, err := strconv.Atoi(orgIdParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "The supplied ID is not a valid format"})
					return
				}
				orgId := uint(atoi)

				eventIdParam := ctx.Params.ByName("eventId")
				atoi, err = strconv.Atoi(eventIdParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "The supplied ID is not a valid format"})
					return
				}
				eventId := uint(atoi)

				userId := ctx.GetUint("id")
				var organization models.Organization
				db := db.GetDb()
				db.Where(&models.Organization{ID: orgId, OwnerID: userId}).First(&organization)
				if organization.ID < 1 {
					err := errors.New("Organization not found")
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				activeOrgId := ctx.GetUint("org")
				if activeOrgId != orgId {
					err := errors.New("Event created must be for active organization")
					ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
					return
				}
				var event models.Event
				err = db.Where(&models.Event{ID: eventId, OrganizerID: orgId}).First(&event).Error
				if err != nil {
					log.Printf("Event %d not found: %s\n", eventId, err.Error())
					ctx.Status(http.StatusNotFound)
					return
				}
				var tickets []models.Ticket
				db.Where(&models.Ticket{EventID: eventId}).Order("created_at desc").Find(&tickets)

				ctx.JSON(http.StatusOK, gin.H{"data": tickets, "count": len(tickets)})
			}).
			POST("/organizations/:orgId/events", func(ctx *gin.Context) {
				id := ctx.Params.ByName("orgId")
				atoi, err := strconv.Atoi(id)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "The supplied ID is not a valid format"})
					return
				}
				userId := ctx.GetUint("id")
				activeOrgId := ctx.GetUint("org")
				orgId := uint(atoi)
				var organization models.Organization
				db := db.GetDb()
				db.Where(&models.Organization{ID: orgId, OwnerID: userId}).First(&organization)
				if organization.ID < 1 {
					err := errors.New("Organization not found")
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				if activeOrgId != orgId && organization.OwnerID != userId {
					err := errors.New("Organization not found")
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				body := types.CreateEventRequestBody{
					Organization: orgId,
				}
				if err := ctx.ShouldBindJSON(&body); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				newId, err := utils.CreateNewEvent(&body, orgId, userId)
				if err != nil {
					log.Printf("error creating event: %s", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "Error creating event"})
					return
				}
				ctx.JSON(http.StatusCreated, gin.H{"id": newId})
			}).
			POST("/organizations/:orgId/account/refresh", func(ctx *gin.Context) {
				orgIdParam := ctx.Params.ByName("orgId")
				atoi, err := strconv.Atoi(orgIdParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				userId := ctx.GetUint("id")
				orgId := uint(atoi)
				var accLinkURL string
				db := db.GetDb()
				var org models.Organization
				err = db.Transaction(func(tx *gorm.DB) error {
					err := tx.
						Model(&models.Organization{}).
						Select("stripe_account_id").
						Where(&models.Organization{ID: orgId, OwnerID: userId}).
						First(&org).
						Error
					if err != nil {
						return err
					}
					if org.StripeAccountID == nil {
						err := fmt.Errorf("Organization is not setup properly: %d", orgId)
						return err
					}
					sc := lib.GetStripeClient()
					acc, err := sc.V1Accounts.GetByID(context.Background(), *org.StripeAccountID, nil)
					if err != nil {
						log.Printf("Error retrieving Account: %s\n", err.Error())
						return err
					}
					if acc == nil {
						err := errors.New("Account not found")
						return err
					}
					accLink, err := sc.V1AccountLinks.Create(context.Background(), &stripe.AccountLinkCreateParams{
						Account:    org.StripeAccountID,
						Type:       stripe.String("account_onboarding"),
						ReturnURL:  stripe.String(fmt.Sprint(os.Getenv("APP_HOST"), "/dashboard")),
						RefreshURL: stripe.String(fmt.Sprint(os.Getenv("APP_HOST"), "/callback/account/refresh")),
					})
					if err != nil {
						return err
					}
					err = tx.Model(&models.Organization{}).Where("id = ?", org.ID).Updates(&models.Organization{
						ConnectOnboardingURL: &accLink.URL,
						Status:               "onboarding",
					}).Error
					if err != nil {
						return err
					}
					accLinkURL = accLink.URL
					return nil
				})
				if err != nil {
					log.Printf("Error while processing request: %s\n", err.Error())
					ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"url": accLinkURL})
			}).
			POST("/organizations/:orgId/onboarding", func(ctx *gin.Context) {
				orgIdParam := ctx.Params.ByName("orgId")
				atoi, err := strconv.Atoi(orgIdParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				orgId := uint(atoi)
				userId := ctx.GetUint("id")
				var user models.User
				var accLinkURL string
				db := db.GetDb()
				var org models.Organization
				err = db.Transaction(func(tx *gorm.DB) error {
					err := tx.
						Model(&models.Organization{}).
						Where(&models.Organization{ID: orgId, OwnerID: userId}).
						First(&org).
						Error
					if err != nil {
						return err
					}
					if org.ConnectOnboardingURL != nil {
						accLinkURL = *org.ConnectOnboardingURL
						return nil
					}
					if org.Status != "pending" {
						ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Organization account was not created via the API"})
					}
					err = tx.Model(&models.User{}).Where("id = ?", userId).First(&user).Error
					if err != nil {
						return err
					}
					if org.StripeAccountID == nil {
						err := fmt.Errorf("Organization is not setup properly: %d", orgId)
						return err
					}
					sc := lib.GetStripeClient()
					acc, err := sc.V1Accounts.GetByID(context.Background(), *org.StripeAccountID, nil)
					if err != nil {
						log.Printf("Error retrieving Account: %s\n", err.Error())
						return err
					}
					if acc == nil {
						err := errors.New("Account not found")
						return err
					}
					accLink, err := sc.V1AccountLinks.Create(context.Background(), &stripe.AccountLinkCreateParams{
						Account:    org.StripeAccountID,
						Type:       stripe.String("account_onboarding"),
						ReturnURL:  stripe.String(fmt.Sprint(os.Getenv("APP_HOST"), "/dashboard")),
						RefreshURL: stripe.String(fmt.Sprint(os.Getenv("APP_HOST"), "/callback/account/refresh")),
					})
					if err != nil {
						return err
					}
					err = tx.Model(&models.Organization{}).Where("id = ?", org.ID).Updates(&models.Organization{
						ConnectOnboardingURL: &accLink.URL,
						Status:               "onboarding",
					}).Error
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
				ctx.JSON(http.StatusOK, gin.H{"url": accLinkURL, "account_id": org.StripeAccountID})
			}).
			GET("/organizations/:orgId/onboarding", func(ctx *gin.Context) {
				orgIdParam := ctx.Params.ByName("orgId")
				atoi, err := strconv.Atoi(orgIdParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				orgId := uint(atoi)
				userId := ctx.GetUint("id")
				key := fmt.Sprintf("%d:onboarding_status", userId)
				rd := lib.GetRedisClient()
				val, err := rd.JSONGet(context.Background(), key).Result()
				if err != nil {
					if !errors.Is(err, redis.Nil) {
						log.Printf("Could not read value from cache: %s\n", err.Error())
						ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
						return
					}
				}
				if gjson.Valid(val) {
					var raw map[string]any
					json.Unmarshal([]byte(val), &raw)
					accountId := raw["accountId"].(string)
					onboardingUrl := raw["url"].(string)
					status := gjson.Get(val, "status").String()
					ctx.JSON(http.StatusOK, gin.H{"completed": status == "complete", "account_id": accountId, "url": onboardingUrl, "data": raw})
					return
				}

				db := db.GetDb()
				var org models.Organization
				ss := db.Session(&gorm.Session{PrepareStmt: true})
				err = ss.Where(&models.Organization{ID: orgId, OwnerID: userId}).First(&org).Error
				if err != nil {
					log.Printf("Error retrieving information: %s\n", err.Error())
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				sc := lib.GetStripeClient()
				accountId := org.StripeAccountID
				if accountId == nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "Account not found"})
					return
				}
				log.Println("AccountID:", org.ID, *accountId)
				stripeAccount, err := sc.V1Accounts.GetByID(context.Background(), *accountId, nil)
				if err != nil {
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				completed := stripeAccount != nil && len(stripeAccount.Requirements.Errors) == 0 &&
					stripeAccount.ChargesEnabled &&
					stripeAccount.PayoutsEnabled &&
					stripeAccount.DetailsSubmitted

				onboarding_status := "incomplete"
				if completed {
					onboarding_status = "complete"
				}
				onboardingStatus := map[string]any{
					"url":              org.ConnectOnboardingURL,
					"accountId":        stripeAccount.ID,
					"status":           onboarding_status,
					"errors":           stripeAccount.Requirements.Errors,
					"chargesEnabled":   stripeAccount.ChargesEnabled,
					"payoutsEnabled":   stripeAccount.PayoutsEnabled,
					"detailsSubmitted": stripeAccount.DetailsSubmitted,
				}
				jbytes, _ := json.Marshal(&onboardingStatus)
				value := string(jbytes)
				_, err = rd.JSONSet(context.Background(), key, "$", value).Result()
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"account_id": stripeAccount.ID, "url": org.ConnectOnboardingURL, "completed": completed, "data": onboardingStatus})
			}).
			GET("/organizations/check", func(ctx *gin.Context) {
				orgId := ctx.GetUint("org")
				var org models.Organization
				db := db.GetDb()
				db.Where(&models.Organization{ID: orgId}).First(&org)
				if org.ID < 1 {
					err := errors.New("Organization not found")
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				shared := org.Type == types.ORG_STANDARD
				ctx.JSON(http.StatusOK, gin.H{"type": org.Type, "shared": shared})
			}).
			PATCH("/events/:id/status", func(ctx *gin.Context) {
				var body struct {
					NewStatus *types.EventStatus `json:"new_status" binding:"required"`
				}
				var params types.SimpleRequestParams
				if err := ctx.ShouldBindUri(&params); err != nil {
					ctx.Status(http.StatusBadRequest)
					return
				}
				if err := ctx.ShouldBindJSON(&body); err != nil {
					ctx.Status(http.StatusBadRequest)
					return
				}
				userId := ctx.GetUint("id")
				db := db.GetDb()
				if err := db.Transaction(func(tx *gorm.DB) error {
					var event models.Event
					if err := tx.
						Where(&models.Event{Organization: models.Organization{OwnerID: userId}}).
						Error; err != nil {
						return err
					}
					if err := tx.
						Model(&models.Event{}).
						Where("id = ?", event.ID).
						Updates(&models.Event{Status: *body.NewStatus, Mode: "manual"}).
						Error; err != nil {
						return err
					}
					return nil
				}); err != nil {
					if errors.Is(gorm.ErrRecordNotFound, err) {
						ctx.Status(http.StatusNotFound)
						return
					}
					ctx.Status(http.StatusForbidden)
					return
				}
				ctx.Status(http.StatusNoContent)
			}).
			GET("/organizations/active", func(ctx *gin.Context) {
				userId := ctx.GetUint("id")
				rd := lib.GetRedisClient()
				val, err := rd.JSONGet(context.Background(), fmt.Sprintf("%d:active", userId)).Result()
				if err != nil {
					log.Printf("Could not retrieve cache value: %s\n", err.Error())
					ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
					return
				}
				if gjson.Valid(val) {
					var orgval models.Organization
					err := json.Unmarshal([]byte(val), &orgval)
					if err != nil {
						ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
						return
					}
					ctx.JSON(http.StatusOK, gin.H{"active": orgval})
					return
				}
				var org models.Organization
				var user models.User
				db := db.GetDb()
				ss := db.Session(&gorm.Session{PrepareStmt: true})
				err = ss.Where(&models.User{ID: userId}).First(&user).Error
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				err = ss.Where(&models.Organization{ID: user.ActiveOrg}).First(&org).Error
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				rd.JSONSet(context.Background(), fmt.Sprintf("%d:active", userId), "$", &org)
				ctx.JSON(http.StatusOK, gin.H{"active": org})
			}).
			POST("/organizations", func(ctx *gin.Context) {
				user := ctx.GetUint("id")
				body := types.CreateOrganizationRequestBody{OwnerID: user}
				if err := ctx.ShouldBindJSON(&body); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				id, err := utils.CreateNewOrganization(&body)
				if err != nil {
					log.Printf("Error creating organization: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusCreated, gin.H{"id": id})
			})

		authorized.
			GET("/events", func(ctx *gin.Context) {
				var events []models.Event
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					today := time.Now()
					in1m := today.Add(1 * time.Minute)
					in3months := today.Add((24 * 30 * 3) * time.Hour)
					err := tx.
						Where(tx.
							Where("status", types.EVENT_REGISTRATION).
							Where("date_time BETWEEN ? AND ?", in1m, in3months),
						).
						Or(tx.
							Where("status", types.EVENT_TICKETS_NOTIFY).
							Where("opens_at BETWEEN ? AND ?", in1m, in3months),
						).
						Order("date_time asc").
						Limit(20).
						Find(&events).
						Error
					if err != nil {
						return err
					}
					return nil
				})

				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				ctx.JSON(http.StatusOK, gin.H{"data": events})
			}).
			GET("/events/waitlist", func(ctx *gin.Context) {
				userId := ctx.GetUint("id")
				var subscriptions []models.EventSubscription
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					err := tx.
						Model(&models.EventSubscription{}).
						Preload("Event").
						Where(tx.
							Model(&models.EventSubscription{}).
							Where(&models.EventSubscription{SubscriberID: userId}).
							Where(clause.IN{Column: "status", Values: []any{
								types.EVENT_SUBSCRIPTION_NOTIFY,
								types.EVENT_SUBSCRIPTION_ACTIVE,
							}})).
						Select("id", "status", "event_id", "created_at").
						Order("created_at DESC").
						Find(&subscriptions).
						Error
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					log.Printf("Error retrieving waitlist: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": subscriptions, "count": len(subscriptions)})
			}).
			GET("/events/:id", func(ctx *gin.Context) {
				id := ctx.Params.ByName("id")
				log.Println("event:", id)
				atoi, err := strconv.Atoi(id)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				eventId := uint(atoi)
				var event models.Event
				db := db.GetDb()
				err = db.
					Model(&models.Event{}).
					Where(&models.Event{ID: eventId}).
					Preload("Organization").
					First(&event).Error
				if err != nil {
					log.Printf("Error finding event %d: %s\n", eventId, err.Error())
					err := errors.New("Event does not exist")
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				log.Printf("%d\n", event.Seats)
				/* orgId := ctx.GetUint("org")
				if event.ID < 1 {
					err := errors.New("Event does not exist")
					log.Printf("Error retrieving info: %s\n", err.Error())
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				} */
				ctx.JSON(http.StatusOK, gin.H{"data": event})
			}).
			POST("/events/:id/subscribe", func(ctx *gin.Context) {
				id := ctx.Params.ByName("id")
				atoi, err := strconv.Atoi(id)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				eventId := uint(atoi)
				userId := ctx.GetUint("id")
				db := db.GetDb()
				var subscription models.EventSubscription
				err = db.Transaction(func(tx *gorm.DB) error {
					var event models.Event
					err := tx.Where(&models.Event{ID: eventId}).First(&event).Error
					if err != nil {
						return err
					}

					err = tx.FirstOrCreate(&subscription, &models.EventSubscription{
						EventID:      eventId,
						SubscriberID: userId,
					}).Error
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					log.Printf("Error creating subscription for event %d: %s\n", eventId, err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				ctx.JSON(http.StatusCreated, gin.H{"id": subscription.ID})
			}).
			PATCH("/events/:id/publish", func(ctx *gin.Context) {
				id := ctx.Params.ByName("id")
				atoi, err := strconv.Atoi(id)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				eventId := uint(atoi)
				err = utils.PublishEvent(eventId)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"id": eventId})
			}).
			GET("/events/:id/tickets", func(ctx *gin.Context) {
				id := ctx.Params.ByName("id")
				atoi, err := strconv.Atoi(id)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				eventId := uint(atoi)
				tickets, err := utils.GetTicketsForEvent(eventId, false)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": tickets})
			}).
			POST("/events/:id/tickets", func(ctx *gin.Context) {
				id := ctx.Params.ByName("id")
				atoi, err := strconv.Atoi(id)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				eventId := uint(atoi)
				var body types.CreateTicketRequestBody
				if err := ctx.ShouldBindJSON(&body); err != nil {
					log.Printf("eeeee: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				orgId := ctx.GetUint("org")
				db := db.GetDb()
				var org models.Organization
				var event models.Event
				db.Where(&models.Organization{ID: orgId}).Find(&org)
				log.Printf("org: %d", org.ID)
				if org.ID < 1 {
					err := errors.New("Organization does not exist")
					log.Printf("error creating ticket for event: %s", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				db.Where(&models.Event{ID: eventId, OrganizerID: orgId}).Find(&event)
				log.Printf("evt: %s", event.Title)
				if event.ID < 1 || (event.OrganizerID > 0 && orgId != event.OrganizerID) {
					err := errors.New("Event does not exist")
					log.Printf("error creating ticket for event: %s", err.Error())
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}

				log.Printf("evt: %d", eventId)
				newId, err := utils.CreateNewTicket(&body)
				log.Printf("newId: %d\n", newId)
				if err != nil {
					log.Printf("error creating ticket: %s", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusCreated, gin.H{"id": newId})
			}).
			POST("/events", func(ctx *gin.Context) {
				var body types.CreateEventRequestBody
				if err := ctx.ShouldBindJSON(&body); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				orgId := ctx.GetUint("org")
				userId := ctx.GetUint("id")
				id, err := utils.CreateNewEvent(&body, orgId, userId)
				if err != nil {
					log.Printf("error creating event: %s", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusCreated, gin.H{"id": id})
			}).
			GET("/events/:id/subscription", func(ctx *gin.Context) {
				var params types.SimpleRequestParams
				if err := ctx.ShouldBindUri(&params); err != nil {
					ctx.Status(http.StatusBadRequest)
					return
				}
				var sub models.EventSubscription
				subscriber := ctx.GetUint("id")
				db := db.GetDb()
				if err := db.
					Model(&models.EventSubscription{}).
					Where(&models.EventSubscription{EventID: params.ID, SubscriberID: subscriber}).
					Select("id").
					First(sub).
					Error; err != nil {
					log.Printf("Error retrieving EventSubscription: %s\n", err.Error())
					if errors.Is(gorm.ErrRecordNotFound, err) {
						ctx.JSON(http.StatusOK, gin.H{"data": 0})
						return
					}
					ctx.Status(http.StatusBadRequest)
					return
				}
				log.Printf("[sub]: %v\n", sub.ID)
				ctx.JSON(http.StatusOK, gin.H{"data": sub.ID})
			}).
			GET("/tickets", func(ctx *gin.Context) {
				orgId := ctx.GetUint("org")
				var tickets []models.Ticket
				db := db.GetDb()
				if err := db.
					Where(&models.Ticket{Event: models.Event{OrganizerID: orgId}}).
					Order("created_at desc").
					Find(&tickets).Error; err != nil {
					log.Printf("Error retrieving Events: %s\n", err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": tickets})
			}).
			POST("/tickets", func(ctx *gin.Context) {
				var body types.CreateTicketRequestBody
				if err := ctx.ShouldBindJSON(&body); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				id, err := utils.CreateNewTicket(&body)
				if err != nil {
					log.Printf("error creating ticket: %s", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusCreated, gin.H{"id": id})
			}).
			POST("/tickets/:id/download/:resId/code", func(ctx *gin.Context) {
				var params types.TicketDownloadURIParams
				if err := ctx.ShouldBindUri(&params); err != nil {
					log.Printf("Error in validating request: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ticketId := params.TicketID
				db := db.GetDb()
				var filepath string
				filename := fmt.Sprintf("ticketcode_%d-%d", ticketId, params.ReservationID)
				var signedURL string
				err := db.Transaction(func(tx *gorm.DB) error {
					var ticket models.Ticket
					if err := tx.
						Where(&models.Ticket{ID: ticketId}).
						First(&ticket).
						Error; err != nil {
						return err
					}
					var reservation models.Reservation
					if err := tx.
						Model(&models.Reservation{}).
						Where(&models.Reservation{
							ID:       params.ReservationID,
							TicketID: params.TicketID,
						}).
						Preload("Booking").
						Preload("Booking.Event").
						First(&reservation).Error; err != nil {
						return err
					}
					now := time.Now()
					if now.After(reservation.Booking.Event.DateTime) {
						err := errors.New("Ticket is no longer valid")
						log.Printf("Error: %s\n", err.Error())
						return err
					}

					rawData := map[string]any{
						"ticketId":      params.TicketID,
						"reservationId": params.ReservationID,
					}
					rawBytes, _ := json.Marshal(rawData)
					rawText := string(rawBytes)

					keyEnv := os.Getenv("API_QRC_SECRET")
					key, err := hex.DecodeString(keyEnv)
					if err != nil {
						log.Printf("Could not read key from string: %s\n", err.Error())
						return err
					}

					encryptedMessage, err := utils.EncryptMessage(key, rawText)
					if err != nil {
						log.Printf("Error encrypting message: %s\n", err.Error())
						return err
					}
					log.Printf("Encrypted message: %s\n", encryptedMessage)
					qrc, err := qrcode.New(encryptedMessage)
					if err != nil {
						return err
					}
					wd, err := os.Getwd()
					if err != nil {
						log.Printf("Could not read working directory: %s\n", err.Error())
						return err
					}
					tempdir := os.Getenv("TEMP_DIR")
					filepath = path.Join(wd, "..", tempdir, fmt.Sprintf("%s.jpeg", filename))
					if err = qrc.Save(filepath); err != nil {
						log.Printf("Could not save qrcode to file [%s]: %s\n", filepath, err.Error())
						return err
					}
					appEnv := os.Getenv("APP_ENV")
					if appEnv == "test" || appEnv == "prod" {
						url, err := awslib.S3UploadAsset(filename, filepath)
						if err != nil {
							log.Printf("Error uploading asset to S3 bucket: %s\n", err.Error())
							return err
						}
						log.Printf("Signed URL: %s\n", *url)
						signedURL = *url
						return nil
					}
					signedURL = filepath
					return nil
				})
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				log.Printf("[signedURL]: %s\n", signedURL)
				ctx.FileAttachment(signedURL, "ticket.jpeg")
			}).
			GET("/tickets/:id/reservations", func(ctx *gin.Context) {
				var params types.TicketReservationsURIParams
				if err := ctx.ShouldBindUri(&params); err != nil {
					log.Printf("Error in validating request: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ticketId := params.TicketID
				var ticket models.Ticket
				db := db.GetDb()
				db.Model(&models.Ticket{}).Where(&models.Ticket{ID: ticketId}).First(&ticket)
				if ticket.ID < 1 {
					err := errors.New("Ticket does not exist")
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

			}).
			GET("/tickets/:id/seats", func(ctx *gin.Context) {
				idParam := ctx.Params.ByName("id")
				atoi, err := strconv.Atoi(idParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ticketId := uint(atoi)
				free, reserved, err := utils.GetTicketSeats(ticketId)
				ctx.JSON(http.StatusOK, gin.H{"id": ticketId, "free": free, "reserved": reserved})
			}).
			PATCH("/tickets/:id/publish", func(ctx *gin.Context) {
				id := ctx.Params.ByName("id")
				atoi, err := strconv.Atoi(id)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ticketId := uint(atoi)
				ticket, err := utils.GetTicket(ticketId)
				if ticket.Event.Status != types.EVENT_OPEN && (ticket.Event.Status != types.EVENT_REGISTRATION && ticket.Event.Mode != "scheduled") {
					err := errors.New("Event must be either published or in scheduled mode")
					log.Printf("Error publishing ticket: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				if err != nil {
					log.Printf("error publishing ticket: %s", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				err = utils.PublishTicket(ticketId)
				if err != nil {
					log.Printf("error publishing ticket: %s", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": ticket})
			}).
			PATCH("/tickets/:id/close", func(ctx *gin.Context) {
				id := ctx.Params.ByName("id")
				atoi, err := strconv.Atoi(id)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ticketId := uint(atoi)
				var ticket models.Ticket
				db := db.GetDb()
				db.Model(&models.Ticket{}).Where(&models.Ticket{ID: ticketId, Status: "open"}).Find(&ticket)
				if ticket.ID < 1 {
					err := errors.New("Ticket not found")
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				err = utils.CloseTicket(ticketId)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": ticket})
			}).
			DELETE("/tickets/:id", func(ctx *gin.Context) {
				id := ctx.Params.ByName("id")
				atoi, err := strconv.Atoi(id)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ticketId := uint(atoi)
				var ticket models.Ticket
				db := db.GetDb()
				db.Model(&models.Ticket{}).Where(&models.Ticket{ID: ticketId}).Find(&ticket)
				if ticket.ID < 1 {
					err := errors.New("Ticket not found")
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				err = utils.CloseTicket(ticketId)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": ticket})
			}).
			GET("/reservations", func(ctx *gin.Context) {
				userId := ctx.GetUint("id")
				orgQuery := ctx.Query("org")
				forOrg, err := strconv.ParseBool(orgQuery)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				var data []models.Booking
				if forOrg {
					orgId := ctx.GetUint("org")
					data, err = utils.GetOrgReservations(orgId)
				} else {
					data, err = utils.GetOwnReservations(userId)
				}
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": data, "count": len(data)})
			}).
			GET("/reservations/:id", func(ctx *gin.Context) {
				idParam := ctx.Params.ByName("id")
				atoi, err := strconv.Atoi(idParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				resId := uint(atoi)
				var reservation models.Reservation
				db := db.GetDb()
				err = db.
					Model(&models.Reservation{}).
					Where(&models.Reservation{ID: resId}).
					Preload("Booking").
					Preload("Booking.Event").
					Preload("Booking.Tickets").
					Preload("Ticket").
					First(&reservation).Error
				if err != nil {
					err := errors.New("Reservation not found")
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": reservation})
			}).
			GET("/bookings", func(ctx *gin.Context) {
				orgId := ctx.GetUint("org")
				db := db.GetDb()
				var bookings []models.Booking
				err := db.Transaction(func(tx *gorm.DB) error {
					err := tx.
						Model(&models.Booking{}).
						// Preload("Tickets").
						Where(&models.Booking{Event: &models.Event{OrganizerID: orgId}}).
						Joins("Event.Organization").
						Find(&bookings).
						Error
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": bookings, "count": len(bookings)})
			}).
			GET("/bookings/:id/reservations", func(ctx *gin.Context) {
				idParam := ctx.Params.ByName("id")
				atoi, err := strconv.Atoi(idParam)
				if err != nil {
					log.Printf(err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				bookingId := uint(atoi)
				var booking models.Booking
				db := db.GetDb()
				err = db.
					Model(&models.Booking{}).
					Where(&models.Booking{ID: bookingId}).
					Preload("Event").
					Preload("Tickets").
					Preload("Reservations").
					Preload("Reservations.Ticket").
					Limit(100).
					First(&booking).
					Error
				if err != nil {
					log.Printf(err.Error())
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": booking})
			})

		authorized.
			GET("/admissions", func(ctx *gin.Context) {
				ctx.Status(http.StatusOK)
			}).
			POST("/admission", func(ctx *gin.Context) {
				var body types.CreateAdmissionRequestBody
				err := ctx.ShouldBindJSON(&body)
				if err != nil {
					log.Printf("Error validating request: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				keyEnv := os.Getenv("API_QRC_SECRET")
				key, err := hex.DecodeString(keyEnv)
				if err != nil {
					log.Printf("Could not read data from string: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				message, err := utils.DecryptMessage(key, body.Code)
				if err != nil {
					log.Printf("Error decrypting message: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				var rawData map[string]any
				json.Unmarshal([]byte(*message), &rawData)
				resIdKey := rawData["reservationId"].(float64)
				reservationId := uint(resIdKey)

				userId := ctx.GetUint("id")
				db := db.GetDb()
				err = db.Transaction(func(tx *gorm.DB) error {
					var user models.User
					if err := tx.Where(&models.User{ID: userId}).First(&user).Error; err != nil {
						return err
					}
					var reservation models.Reservation
					err := tx.
						Where(&models.Reservation{ID: reservationId}).
						Preload("Ticket").
						Preload("Booking").
						Preload("Booking.Event").
						First(&reservation).Error
					if err != nil {
						return err
					}
					if reservation.Booking.Event.Status == types.EVENT_COMPLETED {
						return errors.New("Ticket admissions are no longer accepted")
					}
					if reservation.Booking.Event.Status != types.EVENT_ADMISSION {
						return errors.New("Ticket admissions are not accepted")
					}
					admission := models.Admission{
						ReservationID: reservationId,
						Type:          "single",
						Status:        "completed",
						By:            userId,
					}
					err = tx.
						Where(models.Admission{ReservationID: reservationId}).
						FirstOrInit(&admission).
						Error
					if err != nil {
						return err
					}
					if admission.ID > 0 {
						return errors.New("Cannot admit. Reservation already completed")
					}
					err = tx.Create(&admission).Error
					if err != nil {
						return err
					}
					if err := tx.
						Model(&models.Reservation{}).
						Where("id = ?", reservationId).
						Update("status", types.RESERVATION_COMPLETED).
						Error; err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					err := fmt.Errorf("Error on Ticket admission: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.Status(http.StatusOK)
			})

		authorized.
			GET("/transactions/:id", func(ctx *gin.Context) {
				idParam := ctx.Params.ByName("id")
				atoi, err := strconv.Atoi(idParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				id := uint(atoi)
				db := db.GetDb()
				var txn models.Transaction
				if err = db.Model(&models.Transaction{}).Where("id = ?", id).First(&txn).Error; err != nil {
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": txn})
			})

		authorized.
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
			POST("/checkout", func(ctx *gin.Context) {
				var body types.CreateBookingRequestBody
				if err := ctx.ShouldBindJSON(&body); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				orgId := ctx.GetUint("org")
				userId := ctx.GetUint("id")
				requestID := uuid.New()
				url, csid, txnId, err := utils.CreateStripeCheckout(&body, map[string]string{
					"orgId":     fmt.Sprint(orgId),
					"requestId": requestID.String(),
					"userId":    fmt.Sprint(userId),
				})
				if err != nil {
					log.Printf("error on checkout: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				}
				_, errs, err := utils.CreateReservation(&body, userId, *url, txnId, csid, &requestID)
				if err != nil {
					log.Printf("Error creating Reservation: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"errors": errs})
					return
				}

				log.Printf("URL: %s\n", *url)
				ctx.JSON(http.StatusOK, gin.H{"url": url})
			}).
			POST("/transactions/checkout", func(ctx *gin.Context) {
				var body types.SimpleTransactionRequestBody
				err := ctx.ShouldBindJSON(&body)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				id := body.CheckoutID
				var booking models.Booking
				db := db.GetDb()
				err = db.
					Model(&models.Booking{}).
					Preload("Event").
					Where(&models.Booking{TransactionID: body.ID, CheckoutSessionId: &id}).
					First(&booking).
					Error
				if err != nil {
					log.Printf("Could not find record [%d] with associated id: %s\n", body.ID, err.Error())
					ctx.JSON(http.StatusNotFound, gin.H{"error": "Could not find record with associated id"})
					return
				}
				if booking.Status != types.BOOKING_PENDING {
					err := errors.New("Could not continue to checkout due to expired reservation")
					ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
					return
				}
				var org models.Organization
				err = db.
					Model(&models.Organization{}).
					Where("id", booking.Event.OrganizerID).
					First(&org).
					Error
				if err != nil {
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				cs := lib.GetStripeClient()
				data, err := cs.V1CheckoutSessions.Retrieve(context.Background(), id, &stripe.CheckoutSessionRetrieveParams{
					Params: stripe.Params{
						StripeAccount: org.StripeAccountID,
					},
				})
				if err != nil {
					log.Printf("[Stripe] Unable to retrieve information: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "Unable to retrieve information"})
					return
				}
				url := data.URL
				ctx.JSON(http.StatusOK, gin.H{"url": url})
			}).
			PUT("/bookings/:id/cancel", func(ctx *gin.Context) {
				var params types.SimpleRequestParams
				err := ctx.ShouldBindUri(&params)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				db := db.GetDb()
				err = db.Transaction(func(tx *gorm.DB) error {
					var booking models.Booking
					err := tx.
						Model(&models.Booking{}).
						Where("id = ?", params.ID).
						First(&booking).
						Error
					if err != nil {
						return err
					}
					if booking.TransactionID == nil {
						err := fmt.Errorf("No transaction found for Booking [%d]\n", params.ID)
						log.Println(err)
						return err
					}
					err = tx.
						Model(&models.Booking{}).
						Where(&models.Booking{ID: params.ID}).
						Updates(&models.Booking{Status: types.BOOKING_CANCELED}).
						Error
					if err != nil {
						return err
					}
					err = tx.
						Model(&models.Reservation{}).
						Where(&models.Reservation{BookingID: params.ID}).
						Updates(&models.Reservation{Status: string(types.RESERVATION_CANCELED)}).
						Error
					if err != nil {
						return err
					}
					err = tx.
						Model(&models.Transaction{}).
						Where(&models.Transaction{ID: *booking.TransactionID}).
						Update("status", types.TRANSACTION_CANCELED).
						Error
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					log.Printf("Could not complete request: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "Error while processing request"})
					return
				}

				ctx.Status(http.StatusNoContent)
			}).
			PUT("/bookings/cancel", func(ctx *gin.Context) {
				var body types.CancelBookingsRequestBody
				if err := ctx.ShouldBindJSON(&body); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					if body.Type == "transaction" {
						if err := tx.
							Model(&models.Transaction{}).
							Where("id = ?", body.TxnID).
							Update("status", "canceled").
							Error; err != nil {
							log.Printf("Could not update transaction %v: %s\n", body.TxnID, err.Error())
							return err
						}
						if err := tx.
							Model(&models.Booking{}).
							Where("transaction_id", body.TxnID).
							Update("status", types.BOOKING_CANCELED).
							Error; err != nil {
							log.Printf("Could not update Booking for Transaction %s: %s\n", body.TxnID, err.Error())
							return err
						}
						var bIds []uint
						txnId, _ := uuid.Parse(body.TxnID)
						if err := tx.
							Model(&models.Booking{}).
							Where(&models.Booking{TransactionID: &txnId}).
							Pluck("id", &bIds).
							Error; err != nil {
							return err
						}
						// ids := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(bIds)), ","), "[]")
						if err := tx.
							Model(&models.Reservation{}).
							Where("booking_id IN (?)", bIds).
							Update("status", "canceled").
							Error; err != nil {
							log.Printf("Could not update Reservation for Booking %v: %s\n", bIds, err.Error())
							return err
						}
					} else if body.Type == "reservation" {
						/* if err := tx.
							Model(&models.Booking{}).
							Where("id IN (?)", body.IDs).
							Update("status", "canceled").
							Error; err != nil {
							log.Printf("Could not update booking %v: %s\n", body.IDs, err.Error())
							return err
						}
						if err := tx.
							Model(&models.Reservation{}).
							Where("booking_id IN (?)", &body.IDs).
							Update("status", "canceled").
							Error; err != nil {
							log.Printf("Could not update Reservation for Booking %v: %s\n", body.IDs, err.Error())
							return err
						} */
						return errors.New("Updating status for individual Booking is not allowed")
					} else {
						err := errors.New("Invalid type")
						return err
					}
					return nil
				})
				if err != nil {
					log.Printf("Error processing Transaction: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.Status(http.StatusNoContent)
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
		"organizations:list",
		"organizations:read",
		"organizations:write",
		"teams:list",
		"teams:read",
		"teams:write",
		"user:read",
		"user:update",
		"events:*",
		"events:list",
		"events:read",
		"events:write",
		"tickets:*",
		"tickets:list",
		"tickets:read",
		"tickets:write",
		"reservations:*",
		"reservations:list",
		"reservations:read",
		"reservations:write",
		"admissions:*",
		"admissions:list",
		"admissions:read",
		"admissions:write",
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
