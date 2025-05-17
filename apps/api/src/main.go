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
	"ebs/src/utils"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stripe/stripe-go/v82"
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
	go boot.InitScheduler()
	go boot.InitBroker()

	/* app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}
	auth, _ := app.Auth(context.Background())
	user, _ := auth.GetUserByEmail(context.Background(), "") */

	router := gin.Default()

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

	guest := router.Group("/")
	guest.
		GET("/", func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
		}).
		POST("/login", func(ctx *gin.Context) {
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
				log.Printf("error from Firebase: %s\n", err.Error())
				ctx.JSON(http.StatusNotFound, gin.H{"error": "No user account is associated with this email"})
				return
			}

			db := db.GetDb()
			var muser models.User
			if err := db.Model(&models.User{}).Select("id").Where(&models.User{Email: user.Email}).First(&muser).Error; err != nil {
				log.Printf("error: %s\n", err.Error())
				ctx.JSON(http.StatusNotFound, gin.H{"error": "No user account is associated with this email"})
				return
			}
			log.Println("id:", muser.ID, muser.Email, muser.UID)

			token, _ := generateJWT(user.Email, muser.ID, muser.ActiveOrg)
			tokens = append(tokens, token)

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
			err = db.Model(&models.User{}).Select("id").Where(&models.User{Email: user.Email}).Find(&muser).Error

			if err != nil {
				log.Printf("error: %s\n", err.Error())
				err := errors.New("user already exists")
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			newUser := models.User{
				Email: user.Email,
				UID:   user.UID,
				Role:  "owner",
			}
			db.Create(&newUser)

			newOrg := models.Organization{
				Name:    "Default",
				OwnerID: newUser.ID,
				Country: "",
				Type:    "personal",
			}
			log.Println("new user:", newUser.ID)
			db.Create(&newOrg)

			log.Println("new org:", newOrg.ID)
			db.Model(&models.User{}).Where(&models.User{ID: newUser.ID}).Update("active_org", newOrg.ID)

			ctx.JSON(http.StatusOK, gin.H{"uid": user.UID})
		})

	stripeWebhook := router.Group("/webhook/stripe")
	stripeWebhook.POST("/", func(ctx *gin.Context) {
		payload := make([]byte, 65536)
		ctx.Request.Body.Read(payload)

		ctx.Status(http.StatusNoContent)
	})

	authorized := router.Group("/")
	authorized.Use(middlewares.AuthMiddleware)
	{
		authorized.
			GET("/test", func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

		/* authorized.
			GET("/bookings", func(ctx *gin.Context) {
				ctx.Status(200)
			})

		authorized.
			GET("/admissions", func(ctx *gin.Context) {
				ctx.Status(200)
			}) */

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
				log.Printf("filters: %v\n", filters)
				orgs := make([]models.Organization, 0)
				// TODO: remove when ADMIN feature is implemented
				if (filters.Type == "standard" && !filters.Owned) || filters.Type != "standard" {
					ctx.JSON(http.StatusOK, gin.H{"data": orgs})
					return
				}
				userId := ctx.GetUint("id")
				conds := &models.Organization{Type: filters.Type}
				if filters.Owned {
					conds.OwnerID = userId
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
				ctx.SetCookie("token", tokenCookie, 3600, "/", os.Getenv("APP_HOST"), false, true)
				ctx.JSON(http.StatusOK, gin.H{"access_token": tokenCookie})
			}).
			GET("/organizations/:orgId/reservations", func(ctx *gin.Context) {
				idParam := ctx.Params.ByName("orgId")
				atoi, err := strconv.Atoi(idParam)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "The supplied ID is not a valid format"})
					return
				}
				orgId := uint(atoi)
				data, err := utils.GetOrgReservations(orgId)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusOK, gin.H{"data": data})
			}).
			GET("/organizations/:orgId/events", func(ctx *gin.Context) {
				id := ctx.Params.ByName("orgId")
				atoi, err := strconv.Atoi(id)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "The supplied ID is not a valid format"})
					return
				}
				userId := ctx.GetUint("id")
				orgId := uint(atoi)
				var organization models.Organization
				db := db.GetDb()
				db.Where(&models.Organization{ID: orgId, OwnerID: userId}).Find(&organization)
				if organization.ID < 1 {
					err := errors.New("Organization not found")
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				activeOrgId := ctx.GetUint("org")
				if activeOrgId != orgId {
					err := errors.New("Event created must be for active organization")
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				var events []models.Event
				db.Where(&models.Event{OrganizerID: orgId}).Order("created_at desc").Find(&events)

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
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				activeOrgId := ctx.GetUint("org")
				if activeOrgId != orgId {
					err := errors.New("Event created must be for active organization")
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				var event models.Event
				db.
					Where(&models.Event{ID: eventId}).
					First(&event)

				ctx.JSON(http.StatusOK, gin.H{"data": event})
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
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				if activeOrgId != orgId && organization.OwnerID != userId {
					err := errors.New("Organization not found")
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
				ctx.JSON(http.StatusOK, gin.H{"account_id": stripeAccount.ID, "url": org.ConnectOnboardingURL, "completed": completed})
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
				shared := org.Type == "standard"
				ctx.JSON(http.StatusOK, gin.H{"type": org.Type, "shared": shared})
			}).
			GET("/organizations/active", func(ctx *gin.Context) {
				userId := ctx.GetUint("id")
				var org models.Organization
				var user models.User
				db := db.GetDb()
				ss := db.Session(&gorm.Session{PrepareStmt: true})
				err := ss.Where(&models.User{ID: userId}).First(&user).Error
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				err = ss.Where(&models.Organization{ID: user.ActiveOrg}).First(&org).Error
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
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
				userId := ctx.GetUint("id")
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					today := time.Now()
					var subscriptions []any
					err := tx.
						Model(&models.EventSubscription{}).
						Where(&models.EventSubscription{SubscriberID: userId}).
						Pluck("event_id", &subscriptions).Error
					if err != nil {
						return err
					}
					in1h := today.Add(1 * time.Hour)
					in3months := today.Add((24 * 30 * 3) * time.Hour)
					err = tx.
						Where(tx.
							Where("status", "open").
							Where("date_time BETWEEN ? AND ?", in1h, in3months),
						).
						Or(tx.
							Where(&models.Event{Status: "notify"}).
							Where("opens_at BETWEEN ? AND ?", in1h, in3months),
						).
						Where(tx.Not(clause.IN{Column: "id", Values: subscriptions})).
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
						Select("id", "status", "event_id", "created_at").
						Where(&models.EventSubscription{SubscriberID: userId}).
						Preload("Event").
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
			})

		authorized.
			GET("/tickets", func(ctx *gin.Context) {
				orgId := ctx.GetUint("org")
				var tickets []models.Ticket
				db := db.GetDb()
				db.Where(&models.Ticket{Event: models.Event{OrganizerID: orgId}}).Order("created_at desc").Find(&tickets)
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
				err := db.Transaction(func(tx *gorm.DB) error {
					var ticket models.Ticket
					if err := tx.Where(&models.Ticket{ID: ticketId}).First(&ticket).Error; err != nil {
						return err
					}
					var reservation models.Reservation
					if err := tx.
						Model(&models.Reservation{}).
						Where(&models.Reservation{
							ID:      params.ReservationID,
							Booking: models.Booking{TicketID: params.TicketID},
						}).
						Preload("Booking").
						First(&reservation).Error; err != nil {
						return err
					}
					apiHost := os.Getenv("API_HOST")
					qrc, err := qrcode.New(fmt.Sprintf("%s/tickets/%d/download/%d/code", apiHost, params.TicketID, params.ReservationID))
					if err != nil {
						return err
					}
					cwd, err := os.Getwd()
					if err != nil {
						return err
					}
					filepath = fmt.Sprintf("%s/../assets/%s.jpeg", cwd, filename)
					if err = qrc.Save(filepath); err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.FileAttachment(filepath, "ticket.jpeg")
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
				if ticket.Event.Status != "open" && (ticket.Event.Status != "notify" && ticket.Event.Mode != "scheduled") {
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
			})

		authorized.
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
			POST("/bookings/reserve", func(ctx *gin.Context) {
				var body types.CreateBookingRequestBody
				if err := ctx.ShouldBindJSON(&body); err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				userId := ctx.GetUint("id")
				url, err := utils.CreateReservation(&body, userId)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.JSON(http.StatusCreated, gin.H{"url": url})
			})

		authorized.
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
				db := db.GetDb()
				err = db.Transaction(func(tx *gorm.DB) error {
					var reservation models.Reservation
					err := tx.
						Where(&models.Reservation{ID: body.ReservationID}).
						Preload("Ticket").
						Preload("Booking").
						Preload("Booking.Event").
						First(&reservation).Error
					if err != nil {
						return err
					}
					eventWhen := reservation.Booking.Event.DateTime
					after := time.Now().After(eventWhen)
					if after {
						return errors.New("Cannot admit reservations for events in the past")
					}
					admission := models.Admission{
						ReservationID: body.ReservationID,
						Type:          "single",
						Status:        "completed",
					}
					err = tx.
						Where(models.Admission{ReservationID: body.ReservationID}).
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
					return nil
				})
				if err != nil {
					err := errors.New("Failed to claim reservation")
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx.Status(http.StatusOK)
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
				url, err := utils.CreateStripeCheckout(&body)
				if err != nil {
					log.Printf("error on checkout: %s\n", err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				log.Printf("URL: %s\n", *url)
				userId := ctx.GetUint("id")
				go utils.CreateReservation(&body, userId)
				ctx.JSON(http.StatusOK, gin.H{"url": url})
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
		"organizations:read",
		"organizations:write",
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
