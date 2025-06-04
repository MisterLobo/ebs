package main

import (
	"ebs/src/db"
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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func admissionHandlers(g *gin.RouterGroup) *gin.RouterGroup {
	g.
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

			tenantId, _ := uuid.Parse(ctx.GetString("tenant_id"))
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
					TenantID:      &tenantId,
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
		})
	return g
}
