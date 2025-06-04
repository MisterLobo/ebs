package main

import (
	"ebs/src/db"
	"ebs/src/models"
	"ebs/src/utils"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func reservationHandlers(g *gin.RouterGroup) *gin.RouterGroup {
	g.
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
		})
	return g
}
