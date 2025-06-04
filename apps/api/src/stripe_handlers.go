package main

import (
	"context"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/middlewares"
	"ebs/src/models"
	"ebs/src/types"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
	"gorm.io/gorm"
)

func stripeWebhookRoute(g *gin.Engine) *gin.RouterGroup {
	apiv1 := apiv1Group(g)
	apiv1.POST("/webhook/stripe", func(ctx *gin.Context) {
		payload := make([]byte, 65536)
		payload, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			log.Printf("Error reading request body: %s\n", err.Error())
			ctx.Status(http.StatusServiceUnavailable)
			return
		}
		whsecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		event, err := webhook.ConstructEvent(payload, ctx.GetHeader("Stripe-Signature"), whsecret)
		if err != nil {
			log.Printf("Error verifying webhook signature: %s\n", err.Error())
			ctx.Status(http.StatusBadRequest)
			return
		}
		log.Printf("[StripeEvent] %s\n", event.Type)
		switch event.Type {
		case "customer.created":
			var cus stripe.Customer
			err := json.Unmarshal(event.Data.Raw, &cus)
			if err != nil {
				log.Printf("[Stripe] Error parsing Customer: %s\n", err.Error())
				break
			}
			id := cus.Metadata["id"]
			atoi, err := strconv.Atoi(id)
			if err != nil {
				log.Printf("Could not retrieve user id for customer %s: %s\n", cus.ID, err.Error())
				break
			}
			userId := uint(atoi)
			db := db.GetDb()
			err = db.Transaction(func(tx *gorm.DB) error {
				var user models.User
				if err := tx.
					Model(&models.User{}).
					Where("id = ?", userId).
					Find(&user).
					Error; err != nil {
					log.Printf("Error while retrieving user info for Customer %s: %s\n", cus.ID, err.Error())
					return errors.New("Could not retrieve user information")
				}

				if err := tx.
					Model(&models.User{}).
					Where("id = ?", userId).
					Updates(&models.User{StripeCustomerId: &cus.ID}).
					Error; err != nil {
					log.Printf("Error updating user: %s\n", err.Error())
					return err
				}
				return nil
			})
			if err != nil {
				log.Printf("Error updating user %d: %s\n", userId, err.Error())
			}
			break
		case "subscription.created":
			var sub stripe.Subscription
			err := json.Unmarshal(event.Data.Raw, &sub)
			if err != nil {
				log.Printf("[Stripe] Error parsing Customer: %s\n", err.Error())
				break
			}
			id := sub.Metadata["id"]
			atoi, err := strconv.Atoi(id)
			if err != nil {
				log.Printf("Could not retrieve user id for Subscription %s: %s\n", sub.ID, err.Error())
				break
			}
			userId := uint(atoi)
			db := db.GetDb()
			err = db.Transaction(func(tx *gorm.DB) error {
				var user models.User
				if err := tx.
					Model(&models.User{}).
					Where("id = ?", userId).
					Find(&user).
					Error; err != nil {
					log.Printf("Error while retrieving user info for Subscription %s: %s\n", sub.ID, err.Error())
					return errors.New("Could not retrieve user information")
				}

				if err := tx.
					Model(&models.User{}).
					Where("id = ?", userId).
					Updates(&models.User{StripeSubscriptionId: &sub.ID}).
					Error; err != nil {
					log.Printf("Error updating user: %s\n", err.Error())
					return err
				}
				return nil
			})
			if err != nil {
				log.Printf("Error updating user %d: %s\n", userId, err.Error())
			}
			break
		case "account.updated":
			var acc stripe.Account
			err := json.Unmarshal(event.Data.Raw, &acc)
			if err != nil {
				log.Printf("[Stripe] Error parsing Account: %s\n", err.Error())
				break
			}
			break
		case "capability.updated":
			var cap stripe.Capability
			err := json.Unmarshal(event.Data.Raw, &cap)
			if err != nil {
				log.Printf("[Stripe] Error parsing Capability: %s\n", err.Error())
				break
			}
			break
		case "payment_intent.created":
			var pi stripe.PaymentIntent
			err := json.Unmarshal(event.Data.Raw, &pi)
			if err != nil {
				log.Printf("[Stripe] Error parsing PaymentIntent: %s\n", err.Error())
				break
			}
			log.Printf("[PaymentIntent] ID: %s %s\n", pi.ID, pi.Status)
			md := pi.Metadata
			log.Printf("[%s] Metadata: %v\n", pi.ID, md)
			requestId := md["requestId"]
			go func() {
				var txn models.Transaction
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					err := tx.
						Model(&models.Transaction{}).
						Where("reference_id = ?", requestId).
						First(&txn).
						Error
					if err != nil {
						return err
					}
					txnId := txn.ID
					err = tx.
						Model(&models.Booking{}).
						Where(&models.Booking{TransactionID: &txnId}).
						Updates(&models.Booking{
							Status:          types.BOOKING_COMPLETED,
							PaymentIntentId: &pi.ID,
						}).
						Error
					if err != nil {
						log.Printf("Error updating Booking group [%s]: %s\n", requestId, err.Error())
						return err
					}
					cli := lib.AWSGetSQSClient()
					qurl, err := cli.GetQueueUrl(context.Background(), &sqs.GetQueueUrlInput{
						QueueName: aws.String("PaymentTransactionUpdates"),
					})
					bUpdates, _ := json.Marshal(&models.Transaction{
						SourceName:  "PaymentIntent",
						SourceValue: pi.ID,
						Status:      types.TRANSACTION_PROCESSING,
						Amount:      float64(pi.Amount),
						Currency:    string(pi.Currency),
					})
					updates := string(bUpdates)
					bConds, _ := json.Marshal(&models.Transaction{
						ID:     txn.ID,
						Status: types.TRANSACTION_PENDING,
					})
					conds := string(bConds)
					bPayload, _ := json.Marshal(map[string]any{
						"source":  "payment_intent.created",
						"id":      txn.ID.String(),
						"conds":   conds,
						"updates": updates,
					})
					sPayload := string(bPayload)
					out, err := cli.SendMessage(context.Background(), &sqs.SendMessageInput{
						QueueUrl:    qurl.QueueUrl,
						MessageBody: aws.String(sPayload),
					})
					if err != nil {
						log.Printf("Could not send message to queue: %s\n", err.Error())
						return err
					}
					log.Printf("Message sent to queue: %s\n", *out.MessageId)
					/* err = tx.
						Where(&models.Transaction{ReferenceID: requestId, Status: types.TRANSACTION_PENDING}).
						Updates(&models.Transaction{
							SourceName:  "PaymentIntent",
							SourceValue: pi.ID,
							Status:      types.TRANSACTION_PROCESSING,
							Amount:      float64(pi.Amount),
							Currency:    string(pi.Currency),
						}).
						Error
					if err != nil {
						return err
					} */
					return nil
				})
				if err != nil {
					log.Printf("Error processing Transaction: %s\n", err.Error())
					return
				}
			}()
			break
		case "payment_intent.succeeded":
			var pi stripe.PaymentIntent
			err := json.Unmarshal(event.Data.Raw, &pi)
			if err != nil {
				log.Printf("[Stripe] Error parsing PaymentIntent: %s\n", err.Error())
				break
			}
			log.Printf("[PaymentIntent] ID: %s %s\n", pi.ID, pi.Status)
			md := pi.Metadata
			log.Printf("[%s] Metadata: %v\n", pi.ID, md)
			requestId := md["requestId"]
			go func() {
				var txn models.Transaction
				var bookings []models.Booking
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					err := tx.
						Model(&models.Transaction{}).
						Where("reference_id = ?", requestId).
						First(&txn).
						Error
					if err != nil {
						return err
					}
					err = tx.
						Model(&models.Booking{}).
						Where("metadata ->> 'requestId' = ?", requestId).
						Preload("Event").
						Find(&bookings).
						Error
					if err != nil {
						return err
					}
					err = tx.
						Model(&models.Booking{}).
						Where("metadata ->> 'requestId' = ?", requestId).
						Updates(&models.Booking{
							Status:          types.BOOKING_COMPLETED,
							PaymentIntentId: &pi.ID,
						}).Error
					if err != nil {
						log.Printf("Error updating Booking group [%s]: %s\n", requestId, err.Error())
						return err
					}
					for _, booking := range bookings {
						err := tx.
							Model(&models.Reservation{}).
							Where("booking_id = ?", booking.ID).
							Preload("Booking").
							Updates(&models.Reservation{
								Status:     string(types.RESERVATION_PAID),
								ValidUntil: booking.Event.DateTime,
							}).
							Error
						if err != nil {
							return err
						}
					}
					cli := lib.AWSGetSQSClient()
					qurl, err := cli.GetQueueUrl(context.Background(), &sqs.GetQueueUrlInput{
						QueueName: aws.String("PaymentTransactionUpdates"),
					})
					bUpdates, _ := json.Marshal(models.Transaction{
						SourceName:  "PaymentIntent",
						SourceValue: pi.ID,
						Status:      types.TRANSACTION_COMPLETED,
						Amount:      float64(pi.Amount),
						Currency:    string(pi.Currency),
					})
					updates := string(bUpdates)
					bConds, _ := json.Marshal(&models.Transaction{
						ID:     txn.ID,
						Status: types.TRANSACTION_PROCESSING,
					})
					conds := string(bConds)
					bPayload, _ := json.Marshal(&map[string]any{
						"source":  "payment_intent.succeeded",
						"id":      txn.ID.String(),
						"conds":   conds,
						"updates": updates,
					})
					sPayload := string(bPayload)
					out, err := cli.SendMessage(context.Background(), &sqs.SendMessageInput{
						QueueUrl:     qurl.QueueUrl,
						MessageBody:  aws.String(sPayload),
						DelaySeconds: 10,
					})
					if err != nil {
						log.Printf("Could not send message to queue: %s\n", err.Error())
						return err
					}
					log.Printf("Message sent to queue: %s\n", *out.MessageId)
					/* err = tx.
						Where(&models.Transaction{ReferenceID: requestId, Status: types.TRANSACTION_PROCESSING}).
						Updates(&models.Transaction{
							Status: types.TRANSACTION_COMPLETED,
						}).
						Error
					if err != nil {
						return err
					} */
					return nil
				})
				if err != nil {
					log.Printf("Error processing Transaction: %s\n", err.Error())
					return
				}
			}()
			break
		case "checkout.session.completed":
			var cs stripe.CheckoutSession
			err := json.Unmarshal(event.Data.Raw, &cs)
			if err != nil {
				log.Printf("[Stripe] Error parsing CheckoutSession: %s\n", err.Error())
				break
			}
			log.Printf("[CheckoutSession] ID: %s %s\n", cs.ID, cs.Status)
			md := cs.Metadata
			log.Printf("[%s] Metadata: %v\n", cs.ID, md)
			requestId := md["requestId"]
			go func() {
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					err := tx.
						Model(&models.Booking{}).
						Where("metadata ->> 'requestId' = ?", requestId).
						Updates(&models.Booking{
							CheckoutSessionId: &cs.ID,
						}).
						Error
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					log.Printf("Error updating Booking records: %s\n", err.Error())
					return
				}
			}()
			break
		}
		ctx.Status(http.StatusNoContent)
	})

	stripeAuth := apiv1.Group("/stripe")
	stripeAuth.Use(middlewares.AuthMiddleware)
	stripeAuth.
		GET("/account", func(ctx *gin.Context) {
			userId := ctx.GetUint("id")
			var user struct {
				StripeAccountId *string `json:"account_id,omitempty"`
			}
			db := db.GetDb()
			if err := db.
				Model(&models.User{}).
				Where(&models.User{ID: userId}).
				Select("StripeAccountId").
				Scan(&user).
				Error; err != nil {
				if errors.Is(gorm.ErrRecordNotFound, err) {
					ctx.Status(http.StatusNotFound)
					return
				}
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": &user.StripeAccountId})
		}).
		GET("/customer", func(ctx *gin.Context) {
			userId := ctx.GetUint("id")
			var user struct {
				StripeCustomerId *string `json:"customer_id,omitempty"`
			}
			db := db.GetDb()
			if err := db.
				Model(&models.User{}).
				Where(&models.User{ID: userId}).
				Select("StripeCustomerId").
				Scan(&user).
				Error; err != nil {
				if errors.Is(gorm.ErrRecordNotFound, err) {
					ctx.Status(http.StatusNotFound)
					return
				}
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": &user.StripeCustomerId})
		}).
		GET("/subscription", func(ctx *gin.Context) {
			userId := ctx.GetUint("id")
			var user struct {
				StripeSubscriptionId *string `json:"subscription_id,omitempty"`
			}
			db := db.GetDb()
			if err := db.
				Model(&models.User{}).
				Where(&models.User{ID: userId}).
				Select("StripeSubscriptionId").
				Scan(&user).
				Error; err != nil {
				if errors.Is(gorm.ErrRecordNotFound, err) {
					ctx.Status(http.StatusNotFound)
					return
				}
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{"data": &user.StripeSubscriptionId})
		}).
		GET("/payment_methods", func(ctx *gin.Context) {
			userId := ctx.GetUint("id")
			var user struct {
				StripeCustomerId *string `json:"customer_id,omitempty"`
			}
			db := db.GetDb()
			if err := db.
				Model(&models.User{}).
				Where(&models.User{ID: userId}).
				Select("StripeCustomerId").
				Scan(&user).
				Error; err != nil {
				if errors.Is(gorm.ErrRecordNotFound, err) {
					ctx.Status(http.StatusNotFound)
					return
				}
				ctx.Status(http.StatusBadRequest)
				return
			}
			sc := lib.GetStripeClient()
			list := sc.V1PaymentMethods.List(context.Background(), &stripe.PaymentMethodListParams{
				Customer: user.StripeCustomerId,
			})
			paymentMethods := make([]*stripe.PaymentMethod, 0)
			for pm, err := range list {
				if err != nil {
					log.Printf("Expected a list but got error: %s\n", err.Error())
					break
				}
				paymentMethods = append(paymentMethods, pm)
			}
			ctx.JSON(http.StatusOK, gin.H{"data": &paymentMethods})
		})
	return apiv1
}
