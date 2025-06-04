package main

import (
	"ebs/src/db"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func eventHandlers(g *gin.RouterGroup) *gin.RouterGroup {
	g.
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
		GET("/events", func(ctx *gin.Context) {
			var events []models.Event
			db := db.GetDb()
			if err := db.Exec("select set_tenant(?)", "1").Error; err != nil {
				log.Printf("Error on Exec: %s\n", err.Error())
			}
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
					// Preload("Event").
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
			newId, err := utils.CreateNewTicket(ctx.Copy(), &body)
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
			id, err := utils.CreateNewEvent(ctx.Copy(), &body, orgId, userId)
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
				if errors.Is(gorm.ErrRecordNotFound, err) {
					ctx.JSON(http.StatusOK, gin.H{"data": 0})
					return
				}
				ctx.Status(http.StatusBadRequest)
				return
			}
			log.Printf("[sub]: %v\n", sub.ID)
			ctx.JSON(http.StatusOK, gin.H{"data": sub.ID})
		})

	return g
}
