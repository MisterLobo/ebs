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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
				err := errors.New("reservation not found")
				ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": reservation})
		}).
		GET("/reservations/:id/admission", func(ctx *gin.Context) {
			var params types.SimpleRequestParams
			if err := ctx.ShouldBindUri(&params); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
			tenantId := ctx.GetString("tenant_id")
			db := db.GetDb()
			if err := db.Exec("select set_tenant(?)", tenantId).Error; err != nil {
				log.Printf("Error on setting tenant %s: %s\n", tenantId, err.Error())
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
				return
			}
			var adm models.Admission
			uu, _ := uuid.Parse(tenantId)
			if err := db.
				Where(&models.Admission{ReservationID: params.ID, TenantID: &uu}).
				Select("id").
				First(&adm).
				Error; err != nil {
				ctx.Status(http.StatusNotFound)
				return
			}
			ctx.Status(http.StatusOK)
		})
	return g
}
