package main

import (
	"ebs/src/db"
	"ebs/src/models"
	"ebs/src/types"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func bookingHandlers(g *gin.RouterGroup) *gin.RouterGroup {
	g.
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
		GET("/bookings/:id", func(ctx *gin.Context) {
			var params types.SimpleRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			orgId := ctx.GetUint("org")
			db := db.GetDb()
			var booking models.Booking
			ss := db.Session(&gorm.Session{PrepareStmt: true})
			if err := ss.
				Model(&models.Booking{}).
				Where(&models.Booking{ID: params.ID, Event: &models.Event{OrganizerID: orgId}}).
				Preload("Event").
				Preload("Ticket").
				Preload("User").
				First(&booking).
				Error; err != nil {
				ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": booking})
		}).
		GET("/bookings/:id/reservations", func(ctx *gin.Context) {
			idParam := ctx.Params.ByName("id")
			atoi, err := strconv.Atoi(idParam)
			if err != nil {
				log.Print(err.Error())
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
				log.Print(err.Error())
				ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": booking})
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
					err := fmt.Errorf("no transaction found for Booking [%d]", params.ID)
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
					return errors.New("updating status for individual Booking is not allowed")
				} else {
					err := errors.New("invalid type")
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
	return g
}
