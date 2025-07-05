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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
)

func transactionHandlers(g *gin.RouterGroup) *gin.RouterGroup {
	g.
		POST("/checkout", func(ctx *gin.Context) {
			var body types.CreateBookingRequestBody
			if err := ctx.ShouldBindJSON(&body); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			orgId := ctx.GetUint("org")
			userId := ctx.GetUint("id")
			requestID := uuid.New()
			url, csid, txnId, err := utils.CreateStripeCheckout(ctx, &body, map[string]string{
				"orgId":     fmt.Sprint(orgId),
				"requestId": requestID.String(),
				"userId":    fmt.Sprint(userId),
			})
			if err != nil {
				log.Printf("error on checkout: %s\n", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			}
			_, errs, err := utils.CreateReservation(ctx, &body, userId, *url, txnId, csid, &requestID)
			if err != nil {
				log.Printf("Error creating Reservation: %s\n", err.Error())
				ctx.JSON(http.StatusBadRequest, gin.H{"errors": errs})
				return
			}

			log.Printf("URL: %s\n", *url)
			ctx.JSON(http.StatusOK, gin.H{"url": *url})
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
				err := errors.New("could not continue to checkout due to expired reservation")
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
			if (url == "") && data.Status == "expired" {
				err := fmt.Errorf("CheckoutSession for transaction [%s] has expired and could not be recovered. Reason: AfterExpiration URL was not configured", body.ID.String())
				log.Printf("Error on checkout: %s\n", err.Error())
				ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
				return
			}
			log.Printf("url: %s %s", url, data.Status)
			ctx.JSON(http.StatusOK, gin.H{"url": url})
		}).
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
	return g
}
