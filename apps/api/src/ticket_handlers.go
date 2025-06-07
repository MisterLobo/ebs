package main

import (
	"context"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	awslib "ebs/src/lib/aws"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/yeqown/go-qrcode"
	"gorm.io/gorm"
)

func ticketHandlers(g *gin.RouterGroup) *gin.RouterGroup {
	g.
		GET("/tickets", func(ctx *gin.Context) {
			orgId := ctx.GetUint("org")
			var tickets []models.Ticket
			db := db.GetDb()
			if err := db.
				Where(&models.Ticket{Event: &models.Event{OrganizerID: orgId}}).
				Order("created_at desc").
				Find(&tickets).Error; err != nil {
				log.Printf("Error retrieving Events: %s\n", err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": tickets})
		}).
		GET("/tickets/:id", func(ctx *gin.Context) {
			var params types.SimpleRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			orgId := ctx.GetUint("org")
			var ticket models.Ticket
			db := db.GetDb()
			if err := db.
				Where(&models.Ticket{ID: params.ID, Event: &models.Event{OrganizerID: orgId}}).
				Preload("Event").
				First(&ticket).
				Error; err != nil {
				log.Printf("Error retrieving Ticket: %s\n", err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}
			free, reserved, _ := utils.GetTicketSeats(ticket.ID)
			ticket.Stats = &models.TicketStats{
				Free:     free,
				Reserved: reserved,
			}
			ctx.JSON(http.StatusOK, gin.H{"data": ticket})
		}).
		POST("/tickets", func(ctx *gin.Context) {
			var body types.CreateTicketRequestBody
			if err := ctx.ShouldBindJSON(&body); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			id, err := utils.CreateNewTicket(ctx.Copy(), &body)
			if err != nil {
				log.Printf("error creating ticket: %s", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			ctx.JSON(http.StatusCreated, gin.H{"id": id})
		}).
		POST("/tickets/:id/download/:resId/code", func(ctx *gin.Context) {
			var query struct {
				ShareLink bool `form:"share_link"`
			}
			if err := ctx.ShouldBindQuery(&query); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
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
			log.Printf("Download eticket for %s\n", filename)
			var signedURL string
			rd := lib.GetRedisClient()
			content, err := rd.Get(context.Background(), filename).Result()
			if err != nil {
				if errors.Is(redis.Nil, err) {
					log.Printf("No value for key: %s\n", filename)
				} else {
					log.Printf("Error reading from cache: %s\n", err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
			}
			wd, err := os.Getwd()
			if err != nil {
				log.Printf("Could not read working directory: %s\n", err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}
			tempdir := os.Getenv("TEMP_DIR")
			if content != "" {
				if query.ShareLink {
					ctx.JSON(http.StatusOK, gin.H{"url": content})
					return
				}
				filepath = path.Join(wd, "..", tempdir, fmt.Sprintf("%s.jpeg", filename))
				if err := awslib.S3DownloadAsset(filename); err != nil {
					log.Printf("Error downloading asset [%s] from S3 bucket: %s\n", filename, err.Error())
					ctx.Status(http.StatusBadRequest)
					return
				}
				ctx.FileAttachment(filepath, "eticket.jpeg")
				return
			}

			err = db.Transaction(func(tx *gorm.DB) error {
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
				if now.After(*reservation.Booking.Event.DateTime) {
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
				filepath = path.Join(wd, "..", tempdir, fmt.Sprintf("%s.jpeg", filename))
				if err = qrc.Save(filepath); err != nil {
					log.Printf("Could not save qrcode to file [%s]: %s\n", filepath, err.Error())
					return err
				}
				appEnv := os.Getenv("APP_ENV")
				if appEnv == "local" {
					url, err := awslib.S3UploadAsset(filename, filepath)
					if err != nil {
						log.Printf("Error uploading asset to S3 bucket: %s\n", err.Error())
						return err
					}
					signedURL = *url
					rd.SetEx(context.Background(), filename, signedURL, 2*time.Hour)
					return nil
				}
				signedURL = filepath
				return nil
			})
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			ctx.FileAttachment(signedURL, "eticket.jpeg")
		}).
		GET("/tickets/:id/reservations", func(ctx *gin.Context) {
			var params types.TicketReservationsURIParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				log.Printf("Error in validating request: %s\n", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			orgId := ctx.GetUint("org")
			ticketId := params.TicketID
			var ticket models.Ticket
			db := db.GetDb()
			if err := db.
				Model(&models.Ticket{}).
				Where(&models.Ticket{ID: ticketId, Event: &models.Event{OrganizerID: orgId}}).
				First(&ticket).
				Error; err != nil {
				log.Printf("Error retrieving Ticket [%d]: %s\n", ticketId, err.Error())
				if errors.Is(gorm.ErrRecordNotFound, err) {
					ctx.Status(http.StatusNotFound)
					return
				}
				ctx.Status(http.StatusBadRequest)
				return
			}
			var reservations []models.Reservation
			ss := db.Session(&gorm.Session{PrepareStmt: true})
			if err := ss.
				Where(&models.Reservation{TicketID: ticketId}).
				Find(&reservations).
				Limit(100).
				Order("created_at DESC").
				Error; err != nil {
				log.Printf("Error retrieving Reservations for Ticket [%d]: %s\n", ticketId, err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": reservations})
		}).
		GET("/tickets/:id/bookings", func(ctx *gin.Context) {
			var params types.TicketReservationsURIParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				log.Printf("Error in validating request: %s\n", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			orgId := ctx.GetUint("org")
			ticketId := params.TicketID
			var ticket models.Ticket
			db := db.GetDb()
			if err := db.
				Model(&models.Ticket{}).
				Where(&models.Ticket{ID: ticketId, Event: &models.Event{OrganizerID: orgId}}).
				First(&ticket).
				Error; err != nil {
				log.Printf("Error retrieving Ticket [%d]: %s\n", ticketId, err.Error())
				if errors.Is(gorm.ErrRecordNotFound, err) {
					ctx.Status(http.StatusNotFound)
					return
				}
				ctx.Status(http.StatusBadRequest)
				return
			}
			var bookings []models.Booking
			ss := db.Session(&gorm.Session{PrepareStmt: true})
			if err := ss.
				Where(&models.Booking{TicketID: ticketId}).
				Preload("User").
				Find(&bookings).
				Limit(100).
				Order("created_at DESC").
				Error; err != nil {
				log.Printf("Error retrieving Bookings for Ticket [%d]: %s\n", ticketId, err.Error())
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": bookings})
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
		})
	return g
}
