package main

import (
	"context"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func eventHandlers(g *gin.RouterGroup) *gin.RouterGroup {
	g.
		PATCH("/events/:id/status", func(ctx *gin.Context) {
			var body struct {
				NewStatus types.EventStatus `json:"new_status" binding:"required"`
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
			tenantId := ctx.GetString("tenant_id")
			tid, _ := uuid.Parse(tenantId)
			db := db.GetDb()
			if err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.
					Model(&models.Event{}).
					Where(&models.Event{
						ID:       params.ID,
						TenantID: &tid,
					}).
					Updates(&models.Event{Status: body.NewStatus, Mode: "manual"}).
					Error; err != nil {
					log.Printf("Error updating event status: %s\n", err.Error())
					return err
				}
				return nil
			}); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					ctx.Status(http.StatusNotFound)
					return
				}
				ctx.Status(http.StatusForbidden)
				return
			}
			ctx.Status(http.StatusNoContent)
		}).
		GET("/events", func(ctx *gin.Context) {
			var events []models.Event
			tenantId := ctx.GetString("tenant_id")
			db := db.GetDb()
			if err := db.Exec("select set_tenant(?)", tenantId).Error; err != nil {
				log.Printf("Error on Exec: %s\n", err.Error())
			}
			err := db.Transaction(func(tx *gorm.DB) error {
				today := time.Now()
				in1m := today.Add(1 * time.Minute)
				in3months := today.Add((24 * 30 * 3) * time.Hour)
				err := tx.
					Where(tx.
						Where("status", types.EVENT_REGISTRATION).
						Where("deadline > ?", time.Now()).
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
				Where(db.Where(&models.Event{ID: eventId})).
				Where(db.
					Or(db.Where("opens_at > ? AND status = ?", time.Now(), types.EVENT_TICKETS_NOTIFY)).
					Or(db.Where("deadline > ? AND status = ?", time.Now(), types.EVENT_REGISTRATION)).
					Or(db.Where("date_time > ? AND status = ?", time.Now(), types.EVENT_ADMISSION)),
				).
				Preload("Organization").
				First(&event).Error
			if err != nil {
				log.Printf("Error finding event %d: %s\n", eventId, err.Error())
				err := errors.New("event does not exist")
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
			uid := ctx.GetString("uid")
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
			go func() {
				rd := lib.GetRedisClient()
				token := rd.JSONGet(context.Background(), fmt.Sprintf("%s:fcm", uid), "$.token").Val()
				log.Printf("[%d] retrieved token from cache: %s", userId, token)
				topic := fmt.Sprintf("EventsToOpen_%d", eventId)
				fcm, _ := lib.GetFirebaseMessaging()
				res, err := fcm.SubscribeToTopic(context.Background(), []string{token}, topic)
				if err != nil {
					log.Printf("[FCM] error subcribing to event [%s]: %s\n", topic, err.Error())
					ctx.Status(http.StatusInternalServerError)
					return
				}
				log.Printf("[FCM] subscribed to event [%s]: %v", topic, res)
			}()

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
				err := errors.New("organization does not exist")
				log.Printf("error creating ticket for event: %s", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			db.Where(&models.Event{ID: eventId, OrganizerID: orgId}).Find(&event)
			log.Printf("evt: %s", event.Title)
			if event.ID < 1 || (event.OrganizerID > 0 && orgId != event.OrganizerID) {
				err := errors.New("event does not exist")
				log.Printf("error creating ticket for event: %s", err.Error())
				ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}

			log.Printf("evt: %d", eventId)
			newId, err := utils.CreateNewTicket(ctx.Copy(), &body)
			log.Printf("newId: %d\n", newId)
			if err != nil {
				log.Printf("error creating ticket: %s", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			ctx.JSON(http.StatusCreated, gin.H{"id": newId})
		}).
		POST("/events/:id/coupon", func(ctx *gin.Context) {
			var params types.SimpleRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			var body struct {
				Name           string   `json:"name" binding:"required"`
				Duration       string   `json:"duration"`
				DiscountType   string   `json:"discount_type" binding:"required"`
				PercentOff     *float64 `json:"percentage_discount"`
				TicketIDs      []*uint  `json:"ticket_ids" binding:"required"`
				AmountOff      *int64   `json:"discount_amount"`
				PromoCode      string   `json:"promo_code" binding:"required"`
				MaxRedemptions *int64   `json:"max_redemptions"`
			}
			if err := ctx.ShouldBindJSON(&body); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			userId := ctx.GetUint("id")
			var user models.User
			var priceIDs []string
			db := db.GetDb()
			if err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Where(&models.User{ID: userId}).First(&user).Error; err != nil {
					log.Printf("could not find user with ID %d: %s\n", userId, err.Error())
					return err
				}
				rows, err := tx.
					Model(&models.Event{}).
					Where(&models.Event{ID: params.ID}).
					Preload("Tickets").
					Rows()
				if err != nil {
					log.Printf("could not retrieve Event %d with Tickets: %s\n", params.ID, err.Error())
					return err
				}
				defer rows.Close()
				for rows.Next() {
					var stripePriceId string
					rows.Scan(&stripePriceId)
					priceIDs = append(priceIDs, stripePriceId)
				}
				return nil
			}); err != nil {
				log.Printf("transaction returned an error: %s\n", err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}
			sc := lib.GetStripeClient()
			couponCreateParams := &stripe.CouponCreateParams{
				Currency: stripe.String("USD"),
				Duration: stripe.String(body.Duration),
				Params: stripe.Params{
					StripeAccount: user.StripeAccountId,
				},
				Metadata: map[string]string{
					"event_id": fmt.Sprint(params.ID),
				},
				MaxRedemptions: body.MaxRedemptions,
				Name:           stripe.String(body.Name),
				AppliesTo: &stripe.CouponCreateAppliesToParams{
					Products: stripe.StringSlice([]string{}),
				},
			}
			if body.DiscountType == "percent" {
				couponCreateParams.PercentOff = body.PercentOff
			} else if body.DiscountType == "amount" {
				couponCreateParams.AmountOff = body.AmountOff
			}
			coupon, err := sc.V1Coupons.Create(context.Background(), couponCreateParams)
			if err != nil {
				log.Printf("Error creating coupon: %s\n", err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}
			log.Printf("Created coupon with ID %s\n", coupon.ID)
			ctx.Status(http.StatusCreated)
		}).
		POST("/events", func(ctx *gin.Context) {
			var body types.CreateEventRequestBody
			if err := ctx.ShouldBindJSON(&body); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			orgId := ctx.GetUint("org")
			userId := ctx.GetUint("id")
			id, err := utils.CreateNewEvent(ctx.Copy(), &body, orgId, userId)
			if err != nil {
				log.Printf("error creating event: %s", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			uid := ctx.GetString("uid")
			rd := lib.GetRedisClient()
			fcmToken := rd.JSONGet(context.Background(), fmt.Sprintf("%s:fcm", uid), "$.token").Val()
			topic := fmt.Sprintf("EventsToClose_%d", id)
			fcm, _ := lib.GetFirebaseMessaging()
			_, err = fcm.SubscribeToTopic(ctx.Copy(), []string{fcmToken}, topic)
			if err != nil {
				log.Printf("Could not subscribe to topic [%s]: %s\n", topic, err.Error())
			}
			ctx.JSON(http.StatusCreated, gin.H{"id": id})
		}).
		GET("/events/:id/subscription", func(ctx *gin.Context) {
			var params types.SimpleRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			var sub struct {
				ID uint
			}
			subscriber := ctx.GetUint("id")
			db := db.GetDb()
			if err := db.
				Model(&models.EventSubscription{}).
				Where(&models.EventSubscription{EventID: params.ID, SubscriberID: subscriber, Status: types.EVENT_SUBSCRIPTION_NOTIFY}).
				Select("id").
				Scan(&sub).
				Error; err != nil {
				log.Printf("Error retrieving EventSubscription: %s\n", err.Error())
				if errors.Is(err, gorm.ErrRecordNotFound) {
					ctx.JSON(http.StatusOK, gin.H{"data": 0})
					return
				}
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": sub.ID})
		}).
		DELETE("/events/:id/subscription", func(ctx *gin.Context) {
			var params types.SimpleRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			tenantId := ctx.GetString("tenant_id")
			subscriber := ctx.GetUint("id")
			db := db.GetDb()
			if err := db.Exec("select set_tenant(?)", tenantId).Error; err != nil {
				log.Printf("Error on Exec: %s\n", err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}
			if err := db.
				Model(&models.EventSubscription{}).
				Delete(&models.EventSubscription{EventID: params.ID, SubscriberID: subscriber}).
				Error; err != nil {
				log.Printf("Error retrieving EventSubscription: %s\n", err.Error())
				if errors.Is(err, gorm.ErrRecordNotFound) {
					ctx.JSON(http.StatusOK, gin.H{"data": 0})
					return
				}
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.Status(http.StatusNoContent)
		}).
		DELETE("/events/:id", func(ctx *gin.Context) {
			var params types.SimpleRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			tenantId := ctx.GetString("tenant_id")
			db := db.GetDb()
			if err := db.Exec("select set_tenant(?)", tenantId).Error; err != nil {
				log.Printf("Error on Exec: %s\n", err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}
			if err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.
					Where(&models.Event{ID: params.ID}).
					First(&models.Event{}).
					Error; err != nil {
					return err
				}
				if err := tx.
					Where(&models.Event{ID: params.ID}).
					Update("status", types.EVENT_ARCHIVED).
					Error; err != nil {
					return err
				}
				if err := tx.
					Model(&models.Event{}).
					Select(clause.Associations).
					Delete(&models.Event{ID: params.ID}).
					Error; err != nil {
					return err
				}
				return nil
			}); err != nil {
				log.Printf("DELETE Event transaction failed: %s\n", err.Error())
				if errors.Is(err, gorm.ErrRecordNotFound) {
					ctx.Status(http.StatusNotFound)
					return
				}
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.Status(http.StatusNoContent)
		})

	return g
}
