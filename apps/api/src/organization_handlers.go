package main

import (
	"context"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/middlewares"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	"github.com/stripe/stripe-go/v82/accountlink"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

func organizationHandlers(g *gin.RouterGroup) *gin.RouterGroup {
	g.Use(middlewares.AuthMiddleware)
	g.
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
			id, err := utils.CreateNewOrganization(ctx.Copy(), &body)
			if err != nil {
				log.Printf("Error creating organization: %s\n", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			ctx.JSON(http.StatusCreated, gin.H{"id": id})
		}).
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
				if errors.Is(err, gorm.ErrRecordNotFound) {
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
			if org.Type != types.ORG_STANDARD && org.Type != types.ORG_PERSONAL {
				err := errors.New("switching to a non-Standard organization type is not allowed")
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
			/* if accountId == nil {
				log.Printf("Error while retrieving account information for Organization [%d]: account is not set up\n", orgId)
				ctx.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
				return
			} */
			onboarding_status := "incomplete"
			onboardingStatus := map[string]any{
				"url":              org.ConnectOnboardingURL,
				"accountId":        nil,
				"status":           onboarding_status,
				"errors":           []*stripe.AccountRequirementsError{},
				"chargesEnabled":   false,
				"payoutsEnabled":   false,
				"detailsSubmitted": false,
			}
			if accountId != nil {
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

				if completed {
					onboarding_status = "complete"
				}
				onboardingStatus["accountId"] = stripeAccount.ID
				onboardingStatus["status"] = onboarding_status
				onboardingStatus["errors"] = stripeAccount.Requirements.Errors
				onboardingStatus["chargesEnabled"] = stripeAccount.ChargesEnabled
				onboardingStatus["payoutsEnabled"] = stripeAccount.PayoutsEnabled
				onboardingStatus["detailsSubmitted"] = stripeAccount.DetailsSubmitted
			}
			jbytes, _ := json.Marshal(&onboardingStatus)
			value := string(jbytes)
			key := fmt.Sprintf("%d:onboarding_status", userId)
			_, err = rd.JSONSet(context.Background(), key, "$", value).Result()
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			rd.JSONSet(context.Background(), fmt.Sprintf("%d:active", userId), "$", org)
			rd.Expire(context.Background(), key, time.Hour)
			appEnv := os.Getenv("APP_ENV")
			secure := appEnv != "local"
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
				err := errors.New("not enough permissions to perform this action")
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
					log.Printf("error retrieving Organization [%d]: %s\n", orgId, err.Error())
					if errors.Is(err, gorm.ErrRecordNotFound) {
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
				if errors.Is(err, gorm.ErrRecordNotFound) {
					err := errors.New("organization not found")
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
			}

			activeOrgId := ctx.GetUint("org")
			if activeOrgId != orgId {
				err := errors.New("event created must be for active organization")
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
			var params struct {
				OrgId   uint `uri:"orgId"`
				EventId uint `uri:"eventId"`
			}
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			orgId := params.OrgId
			eventId := params.EventId

			userId := ctx.GetUint("id")
			var organization models.Organization
			db := db.GetDb()
			if err := db.
				Where(&models.Organization{ID: orgId, OwnerID: userId}).
				First(&organization).Error; err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			activeOrgId := ctx.GetUint("org")
			if activeOrgId != orgId {
				err := errors.New("event created must be for active organization")
				ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				return
			}
			var event models.Event
			if err := db.
				Where(&models.Event{ID: eventId}).
				First(&event).Error; err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}

			ctx.JSON(http.StatusOK, gin.H{"data": event})
		}).
		PUT("/organizations/:orgId/events/:eventId/cancel", func(ctx *gin.Context) {
			var params struct {
				OrgId   uint `uri:"orgId"`
				EventId uint `uri:"eventId"`
			}
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			orgId := params.OrgId
			eventId := params.EventId

			userId := ctx.GetUint("id")
			var organization models.Organization
			db := db.GetDb()
			if err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.
					Where(&models.Organization{ID: orgId, OwnerID: userId}).
					First(&organization).Error; err != nil {
					return err
				}
				subq := tx.Where(&models.Event{ID: eventId})

				if err := tx.Where(subq).Update("status = ?", types.EVENT_CANCELED).Error; err != nil {
					log.Printf("Error on canceling Event [%d]: %s\n", eventId, err.Error())
					return err
				}
				return nil
			}); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.Status(http.StatusNoContent)
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
			var sales struct {
				TotalRevenue int64 `json:"total_revenue"`
				TotalSold    int64 `json:"total_sold"`
			}
			db := db.GetDb()
			if err := db.Where(&models.Organization{ID: params.ID}).First(&org).Error; err != nil {
				log.Printf("Error retrieving Organization: %s\n", err.Error())
				ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			now := time.Now()
			past1month := now.AddDate(0, 0, -30)
			eventSub := db.
				Model(&models.Event{}).
				Where(&models.Event{OrganizerID: params.ID}).
				Select("id")
			if err := db.
				Model(&models.Booking{}).
				Where("event_id IN (?)", eventSub).
				Where("status = ?", types.BOOKING_COMPLETED).
				Where("created_at BETWEEN ? AND ?", past1month, now).
				Select("SUM(subtotal) as total_revenue", "SUM(qty) as total_sold").
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
		GET("/organizations/:orgId/customers", func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
		}).
		GET("/organizations/:orgId/customers/count", func(ctx *gin.Context) {
			var params types.SimpleOrganizationRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			var org models.Organization
			db := db.GetDb()
			if err := db.Where(&models.Organization{ID: params.ID}).First(&org).Error; err != nil {
				log.Printf("Error retrieving Organization: %s\n", err.Error())
				ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			now := time.Now()
			past1month := now.AddDate(0, 0, -30)
			var count int64
			// eventSub := db.Table("events AS E").Where("organizer_id = ?", params.ID)
			if err := db.
				Model(&models.Booking{}).
				Joins("JOIN events E ON event_id=E.id").
				Where("bookings.created_at >= ? AND E.organizer_id = ?", past1month, org.ID).
				Distinct("bookings.user_id").
				Select("bookings.user_id").
				Count(&count).
				Error; err != nil {
				log.Printf("Error on query: %s\n", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx.JSON(http.StatusOK, gin.H{"data": count})
		}).
		GET("/organizations/:orgId/customers/daily", func(ctx *gin.Context) {
			var params types.SimpleOrganizationRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			db := db.GetDb()
			var stats []struct {
				TxnDate      string `json:"date"`
				CompletedTxn int    `json:"completed"`
				PendingTxn   int    `json:"pending"`
				TotalTxn     int    `json:"total"`
			}
			if err := db.Raw(`
			SELECT
				TO_CHAR(d::date, 'YYYY-MM-DD') AS txn_date,
				COUNT(CASE WHEN b.status = 'completed' THEN 1 END) AS completed_txn,
				COUNT(CASE WHEN b.status = 'pending' THEN 1 END) AS pending_txn,
				COUNT(b.id) AS total_txn
			FROM generate_series(
				CURRENT_DATE - INTERVAL '30 days',
				CURRENT_DATE,
				INTERVAL '1 day'
			) as d
			LEFT JOIN events e ON e.organizer_id = ?
			LEFT JOIN bookings b ON b.event_id = e.id AND b.created_at::date = d::date
			LEFT JOIN users u ON u.id = b.user_id
			GROUP BY d.d
			ORDER BY d.d
			`, params.ID).Scan(&stats).Error; err != nil {
				log.Printf("Error querying Transaction: %s\n", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": stats})
		}).
		GET("/organizations/:orgId/transactions/daily", func(ctx *gin.Context) {
			var params types.SimpleOrganizationRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			/* var query struct {
				FromDate *string `json:"from_date"`
			} */
			/* queryFromDate := time.Now().Format("2006-01-02")
			if query.FromDate != nil {
				fromDate, err := time.Parse("2006-01-02", *query.FromDate)
				if err != nil {
					ctx.Status(http.StatusBadRequest)
					return
				}
				queryFromDate = fromDate.Format("2006-01-02")
			} */

			db := db.GetDb()
			var stats []struct {
				TxnDate      string `json:"date"`
				CompletedTxn int    `json:"completed"`
				PendingTxn   int    `json:"pending"`
				TotalTxn     int    `json:"total"`
				TotalRev     int    `json:"total_revenue"`
				Currency     string `json:"currency"`
			}
			if err := db.Raw(`
			SELECT
				TO_CHAR(d::date, 'YYYY-MM-DD') AS txn_date,
				COUNT(DISTINCT CASE WHEN t.status = 'paid' THEN 1 END) AS completed_txn,
				COUNT(DISTINCT CASE WHEN t.status = 'pending' THEN 1 END) AS pending_txn,
				COUNT(DISTINCT t.id) AS total_txn,
				COALESCE(SUM(DISTINCT CASE WHEN t.currency = 'usd' THEN ROUND(t.amount_paid/100) ELSE t.amount_paid END),0) AS total_rev
			FROM generate_series(
				CURRENT_DATE - INTERVAL '30 days',
				CURRENT_DATE,
				INTERVAL '1 day'
			) as d
			LEFT JOIN events e ON e.organizer_id = ?
			LEFT JOIN bookings b ON b.event_id = e.id
			LEFT JOIN transactions t ON t.id = b.transaction_id AND t.created_at::date = d::date
			GROUP BY d.d
			ORDER BY d.d
			`, params.ID).Scan(&stats).Error; err != nil {
				log.Printf("Error querying Transaction: %s\n", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": stats})
		}).
		GET("/organizations/:orgId/dashboard", func(ctx *gin.Context) {
			var params types.SimpleOrganizationRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			var results []struct {
				EventID    uint   `json:"id"`
				EventTitle string `json:"event_title"`
				TotalRev   int    `json:"total_rev"`
				TotalQty   int    `json:"total_qty"`
			}
			db := db.GetDb()
			if err := db.
				Raw(`
				SELECT
					e.id AS event_id,
					e.title AS event_title,
					COALESCE(SUM(DISTINCT CASE WHEN t.currency = 'usd' THEN ROUND(t.amount_paid/100) ELSE t.amount_paid END),0) AS total_rev,
					SUM(DISTINCT b.qty) AS total_qty
				FROM transactions t
				LEFT JOIN events e ON e.organizer_id = ?
				LEFT JOIN bookings b ON b.event_id = e.id
				WHERE t.id = b.transaction_id AND t.created_at >= CURRENT_DATE - INTERVAL '30 days'
				GROUP BY t.created_at, e.id, e.title
				ORDER BY total_qty
				LIMIT 10
			`, params.ID).
				Scan(&results).
				Error; err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": results})
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
				err := errors.New("organization not found")
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			activeOrgId := ctx.GetUint("org")
			if activeOrgId != orgId {
				err := errors.New("event created must be for active organization")
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
				err := errors.New("organization not found")
				ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			if activeOrgId != orgId && organization.OwnerID != userId {
				err := errors.New("organization not found")
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
			newId, err := utils.CreateNewEvent(ctx.Copy(), &body, orgId, userId)
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
					err := fmt.Errorf("organization is not setup properly: %d", orgId)
					return err
				}
				sc := lib.GetStripeClient()
				acc, err := sc.V1Accounts.GetByID(context.Background(), *org.StripeAccountID, nil)
				if err != nil {
					log.Printf("Error retrieving Account: %s\n", err.Error())
					return err
				}
				if acc == nil {
					err := errors.New("account not found")
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
					err := fmt.Errorf("organization is not setup properly: %d", orgId)
					return err
				}
				sc := lib.GetStripeClient()
				acc, err := sc.V1Accounts.GetByID(context.Background(), *org.StripeAccountID, nil)
				if err != nil {
					log.Printf("Error retrieving Account: %s\n", err.Error())
					return err
				}
				if acc == nil {
					err := errors.New("account not found")
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
			var accountId string
			if gjson.Valid(val) {
				var raw map[string]any
				json.Unmarshal([]byte(val), &raw)
				accountId := gjson.Get(val, "accountId").String()
				onboardingUrl := gjson.Get(val, "url").String()
				status := gjson.Get(val, "status").String()
				log.Printf("accountId is nil: %v, %v", accountId, accountId == "")
				if accountId != "" {
					ctx.JSON(http.StatusOK, gin.H{"completed": status == "complete", "account_id": accountId, "url": onboardingUrl, "data": raw})
					return
				}
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
			accountId = *org.StripeAccountID
			// sc := lib.GetStripeClient()
			var acc *stripe.Account
			if org.StripeAccountID == nil && accountId == "" {
				// Create the Account
				businessType := "individual"
				acc, err = account.New(&stripe.AccountParams{
					BusinessProfile: &stripe.AccountBusinessProfileParams{
						Name:         stripe.String(org.Name),
						SupportEmail: stripe.String(org.ContactEmail),
					},
					Company: &stripe.AccountCompanyParams{
						Name: stripe.String(org.Name),
					},
					BusinessType: stripe.String(businessType),
					Type:         stripe.String("express"),
					Email:        stripe.String(org.ContactEmail),
					Metadata:     map[string]string{"organizationId": fmt.Sprintf("%d", org.ID)},
					Capabilities: &stripe.AccountCapabilitiesParams{
						CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{
							Requested: stripe.Bool(true),
						},
						Transfers: &stripe.AccountCapabilitiesTransfersParams{
							Requested: stripe.Bool(true),
						},
					},
				})
				if err != nil {
					log.Printf("[Stripe] Error creating Connect account for Organization [%d]: %s\n", orgId, err.Error())
					ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
					return
				}
				accLink, err := accountlink.New(&stripe.AccountLinkParams{
					Account:    stripe.String(acc.ID),
					Type:       stripe.String("account_onboarding"),
					ReturnURL:  stripe.String(fmt.Sprint(os.Getenv("APP_HOST"), "/dashboard")),
					RefreshURL: stripe.String(fmt.Sprint(os.Getenv("APP_HOST"), "/callback/account/refresh")),
				})
				if err != nil {
					log.Printf("[Stripe] Error creating AccountLink for Organization [%d]: %s\n", orgId, err.Error())
					ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
					return
				}
				go func() {
					if err := db.Transaction(func(tx *gorm.DB) error {
						org.StripeAccountID = &acc.ID
						org.ConnectOnboardingURL = &accLink.URL
						org.Status = "onboarding"
						if err := tx.Save(&org).Error; err != nil {
							return err
						}
						return nil
					}); err != nil {
						return
					}
				}()
				accountId = acc.ID
			}
			acc, err = account.GetByID(accountId, nil)
			if err != nil {
				if !errors.Is(err, &stripe.Error{Code: stripe.ErrorCodeResourceMissing}) {
					log.Printf("Error retrieving Account details for Organization %d: %s\n", orgId, err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
			}

			if acc == nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "Account not found"})
				return
			}
			log.Println("AccountID:", org.ID, accountId)
			stripeAccount, err := account.GetByID(accountId, nil)
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
			if err := db.Where(&models.Organization{ID: orgId}).First(&org).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					err := errors.New("organization not found")
					ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				}
				log.Printf("Error retrieving Orgnization [%d] details: %s\n", orgId, err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}
			shared := org.Type == types.ORG_STANDARD
			ctx.JSON(http.StatusOK, gin.H{"type": org.Type, "shared": shared})
		}).
		POST("/organizations/:orgId/verify/send", func(ctx *gin.Context) {
			var params types.SimpleOrganizationRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			var body struct {
				Email  string `json:"email"`
				Phone  string `json:"phone"`
				Method string `json:"method"`
			}
			if err := ctx.ShouldBindJSON(&body); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.Status(http.StatusOK)
		}).
		POST("/organizations/:orgId/verify/confirm", func(ctx *gin.Context) {
			var params types.SimpleOrganizationRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			var body struct {
				Email  string `json:"email"`
				Phone  string `json:"phone"`
				Method string `json:"method"`
			}
			if err := ctx.ShouldBindJSON(&body); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.Status(http.StatusOK)
		})
	return g
}
