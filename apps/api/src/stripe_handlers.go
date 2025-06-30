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
	"fmt"
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
	"gorm.io/gorm/clause"
)

func stripeWebhookRoute(g *gin.Engine) *gin.RouterGroup {
	apiv1 := apiv1Group(g)
	apiv1.POST("/webhook/stripe", func(ctx *gin.Context) {
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
					return errors.New("could not retrieve user information")
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
		case "customer.subscription.created":
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
					return errors.New("could not retrieve user information")
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
		case "account.updated":
			var acc stripe.Account
			err := json.Unmarshal(event.Data.Raw, &acc)
			if err != nil {
				log.Printf("[Stripe] Error parsing Account: %s\n", err.Error())
				break
			}
			md := acc.Metadata
			organizationId := md["organizationId"]
			orgId, err := strconv.Atoi(organizationId)
			if err != nil {
				log.Printf("Error reading property organizationId from Metadata: %s\n", err.Error())
				return
			}
			completed := len(acc.Requirements.Errors) == 0 &&
				acc.ChargesEnabled &&
				acc.PayoutsEnabled &&
				acc.DetailsSubmitted
			if completed {
				db := db.GetDb()
				db.Transaction(func(tx *gorm.DB) error {
					if err := tx.
						Model(&models.Organization{}).
						Where("id = ?", orgId).
						Updates(&models.Organization{
							Verified:        completed,
							PaymentVerified: acc.ChargesEnabled,
						}).
						Error; err != nil {
						return err
					}
					return nil
				})
			}
		case "capability.updated":
			var cap stripe.Capability
			err := json.Unmarshal(event.Data.Raw, &cap)
			if err != nil {
				log.Printf("[Stripe] Error parsing Capability: %s\n", err.Error())
				break
			}
		case "invoice.paid":
			var inv stripe.Invoice
			if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
				log.Printf("[Stripe] Error parsing Invoice: %s\n", err.Error())
				break
			}
			md := inv.Metadata
			requestId := md["requestId"]
			go func() {
				var bookings []models.Booking
				var txn models.Transaction
				db := db.GetDb()
				if err := db.Transaction(func(tx *gorm.DB) error {
					err := tx.
						Model(&models.Transaction{}).
						Where("reference_id = ?", requestId).
						First(&txn).
						Error
					if err != nil {
						return err
					}
					txnId := txn.ID
					if err := tx.
						Model(&models.Booking{}).
						Preload("User").
						Where("transaction_id = ?", txnId).
						Select("event_id", "user_id").
						Find(&bookings).
						Error; err != nil {
						log.Printf("Could not retrieve Event IDs for selected Booking: %s\n", err.Error())
						return err
					}
					user := bookings[0].User
					fcm, _ := lib.GetFirebaseMessaging()
					rd := lib.GetRedisClient()
					token := rd.JSONGet(context.Background(), fmt.Sprintf("%s:fcm", user.UID), "$.token").Val()
					log.Printf("[%d] retrieved token from cache: %s", user.ID, token)
					for _, b := range bookings {
						topic := fmt.Sprintf("EventsToOpen_%d", b.EventID)
						sub, err := fcm.SubscribeToTopic(context.Background(), []string{token}, topic)
						if err != nil {
							log.Printf("Error subscribing to topic %s: %s\n", topic, err.Error())
						} else {
							log.Printf("Added topic %s to subscription: %d\n", topic, sub.SuccessCount)
						}
					}

					err = tx.
						Model(&models.Booking{}).
						Where(&models.Booking{TransactionID: &txnId}).
						Updates(&models.Booking{
							Status: types.BOOKING_COMPLETED,
						}).
						Error
					if err != nil {
						log.Printf("Error updating Booking group [%s]: %s\n", requestId, err.Error())
						return err
					}
					if err := tx.
						Model(&models.Transaction{}).
						Where("reference_id", requestId).
						Where(clause.IN{Column: "status", Values: []any{
							types.TRANSACTION_PENDING,
							types.TRANSACTION_PROCESSING,
						}}).
						Updates(&models.Transaction{
							Amount:     float64(inv.Subtotal),
							AmountPaid: float64(inv.Total),
							Currency:   string(inv.Currency),
							Status:     types.TRANSACTION_COMPLETED,
						}).
						Error; err != nil {
						return err
					}
					log.Printf("[invoice.paid] Transaction with request ID [%s] has completed successfully!\n", requestId)
					return nil
				}); err != nil {
					log.Printf("Transaction failed: %s\n", err.Error())
				}
			}()
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
					if err != nil {
						log.Printf("Error retrieving queue URL: %s\n", err.Error())
						return err
					}
					bUpdates, _ := json.Marshal(&models.Transaction{
						SourceName:  "PaymentIntent",
						SourceValue: pi.ID,
						Status:      types.TRANSACTION_PROCESSING,
						// Amount:      float64(pi.Amount),
						Currency: string(pi.Currency),
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
					apiEnv := os.Getenv("API_ENV")
					txnUpdates := &models.Transaction{
						SourceName:  "PaymentIntent",
						SourceValue: pi.ID,
						Status:      types.TRANSACTION_PROCESSING,
						// Amount:      float64(pi.Amount),
						// AmountPaid:  float64(pi.AmountReceived),
						Currency:        string(pi.Currency),
						PaymentIntentId: &pi.ID,
					}
					txnConds := &models.Transaction{
						ID:     txn.ID,
						Status: types.TRANSACTION_PENDING,
					}
					bUpdates, _ := json.Marshal(txnUpdates)
					updates := string(bUpdates)
					bConds, _ := json.Marshal(txnConds)
					conds := string(bConds)
					txnPayload := &map[string]any{
						"source":  "payment_intent.succeeded",
						"id":      txn.ID.String(),
						"conds":   conds,
						"updates": updates,
					}
					bPayload, _ := json.Marshal(txnPayload)
					sPayload := string(bPayload)
					if apiEnv == string(types.Test) || apiEnv == string(types.Production) {
						cli := lib.AWSGetSQSClient()
						qurl, err := cli.GetQueueUrl(context.Background(), &sqs.GetQueueUrlInput{
							QueueName: aws.String("PaymentTransactionUpdates"),
						})
						if err != nil {
							log.Printf("Error retrieving queue URL: %s\n", err.Error())
							return err
						}
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
						return nil
					}
					if err := lib.KafkaProduceMessage(
						"PaymentTransactionUpdatesProducer",
						"PaymentTransactionUpdates",
						&types.JSONB{
							"source":  "payment_intent.succeeded",
							"id":      txn.ID.String(),
							"updates": txnUpdates,
							"conds":   txnConds,
						},
					); err != nil {
						log.Printf("Error sending message to queue: %s\n", err.Error())
						return err
					}
					return nil
				})
				if err != nil {
					log.Printf("Error processing Transaction: %s\n", err.Error())
					return
				}
			}()
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
					if len(cs.Discounts) > 0 {
						discount := cs.Discounts[0]
						promo := discount.PromotionCode
						if err := tx.
							Model(&models.Transaction{}).
							Where("reference_id = ?", requestId).
							Updates(&models.Transaction{
								PromoId: &promo.ID,
							}).
							Error; err != nil {
							return err
						}
					}
					return nil
				})
				if err != nil {
					log.Printf("Error updating Booking records: %s\n", err.Error())
					return
				}
			}()
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
				if errors.Is(err, gorm.ErrRecordNotFound) {
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
				if errors.Is(err, gorm.ErrRecordNotFound) {
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
				if errors.Is(err, gorm.ErrRecordNotFound) {
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
				if errors.Is(err, gorm.ErrRecordNotFound) {
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
		}).
		POST("/payment_methods", func(ctx *gin.Context) {
			var body struct {
				PaymentMethodID string `json:"pm_id" binding:"required"`
			}
			if err := ctx.ShouldBindJSON(&body); err != nil {
				ctx.Status(http.StatusBadRequest)
				return
			}
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
				if errors.Is(err, gorm.ErrRecordNotFound) {
					ctx.Status(http.StatusNotFound)
					return
				}
				ctx.Status(http.StatusBadRequest)
				return
			}
			ctx.Status(http.StatusNoContent)
		})
	return apiv1
}
