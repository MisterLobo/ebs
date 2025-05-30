package utils

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"ebs/src/config"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/models"
	"ebs/src/types"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/stripe/stripe-go/v82"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateNewEvent(params *types.CreateEventRequestBody, organizationId uint, creatorId uint) (uint, error) {
	dateTime, err := time.Parse(config.TIME_PARSE_FORMAT, params.DateTime)
	if err != nil {
		log.Printf("Error parsing date_time: %s\n", err.Error())
		return 0, err
	}
	dateTime = time.Date(
		dateTime.Year(),
		dateTime.Month(),
		dateTime.Day(),
		dateTime.Hour(),
		dateTime.Minute(),
		0,
		0,
		dateTime.Location(),
	)
	log.Printf("dateTime: Local=%s UTC=%s\n", dateTime.Local().String(), dateTime.UTC().String())
	eventStatus := types.EVENT_DRAFT

	event := models.Event{
		Title:       params.Title,
		Name:        params.Name,
		About:       &params.Description,
		Location:    params.Location,
		DateTime:    dateTime,
		Seats:       params.Seats,
		OrganizerID: organizationId,
		CreatedBy:   creatorId,
		Status:      eventStatus,
		Mode:        params.Mode,
	}

	var eventId uint
	var opens_at *time.Time
	db := db.GetDb()
	err = db.Transaction(func(tx *gorm.DB) error {
		deadline, err := time.Parse(config.TIME_PARSE_FORMAT, params.Deadline)
		if err != nil {
			log.Printf("Error parsing deadline: %s\n", err.Error())
			return err
		}
		deadline = time.Date(
			deadline.Year(),
			deadline.Month(),
			deadline.Day(),
			deadline.Hour(),
			deadline.Minute(),
			0,
			0,
			deadline.Location(),
		)
		log.Printf("dateTime: Local=%s", deadline.String())
		event.Deadline = deadline
		if params.OpensAt != nil {
			opensAt, err := time.Parse(config.TIME_PARSE_FORMAT, *params.OpensAt)
			if err != nil {
				return err
			}
			if params.Mode == "scheduled" {
				event.OpensAt = &opensAt
				event.Status = types.EVENT_TICKETS_NOTIFY
				opens_at = &opensAt
			}
		}

		org := models.Organization{ID: organizationId}
		user := models.User{ID: creatorId}

		err = tx.Find(&user).Error
		if err != nil {
			return err
		}
		err = tx.Find(&org).Error
		if err != nil {
			return err
		}
		if org.Type != types.ORG_STANDARD && org.Type != types.ORG_PERSONAL {
			err := errors.New("Not enough permissions to perform this action")
			return err
		}
		err = tx.Create(&event).Error
		if err != nil {
			return err
		}
		eventId = event.ID

		// Set a schedule for completing the event
		go func() {
			runsAt := event.DateTime
			runDate := time.Date(
				runsAt.UTC().Year(),
				runsAt.UTC().Month(),
				runsAt.UTC().Day(),
				runsAt.UTC().Hour(),
				runsAt.UTC().Minute(),
				0,
				0,
				runsAt.UTC().Location(),
			)
			log.Printf("[DateTime] job scheduled at: %s\n", runDate)
			jobTaskID := uuid.New()
			payloadId := jobTaskID.String()
			jobTask := models.JobTask{
				Name:    fmt.Sprintf("Event_%d_DateTime", eventId),
				JobType: "OneTimeJobStartDateTime",
				RunsAt:  runDate,
				HandlerParams: []any{
					eventId,
				},
				PayloadID: payloadId,
				Payload: map[string]any{
					"payloadId":        payloadId,
					"id":               int64(eventId),
					"producerClientId": "EventsToCompleteProducer",
					"topic":            "EventsToComplete",
					"table":            "events",
				},
				Source:     "Events",
				SourceType: "table",
				Topic:      "EventsToComplete",
			}
			id, err := jobTask.CreateAndEnqueueJobTask(jobTask)
			if err != nil {
				log.Printf("Error creating job for Event: id=%d error=%s\n", eventId, err.Error())
				return
			}
			log.Printf("Created job for Event[%d] with ID %s\n", eventId, id)
		}()

		// Set a schedule for Closing the ticket reservation
		go func() {
			runsAt := event.Deadline
			runDate := time.Date(
				runsAt.UTC().Year(),
				runsAt.UTC().Month(),
				runsAt.UTC().Day(),
				runsAt.UTC().Hour(),
				runsAt.UTC().Minute(),
				0,
				0,
				runsAt.UTC().Location(),
			)
			log.Printf("[Deadline] job scheduled at: %s\n", runDate)
			jobTaskID := uuid.New()
			payloadId := jobTaskID.String()
			jobTask := models.JobTask{
				Name:    fmt.Sprintf("Event_%d_Deadline", eventId),
				JobType: "OneTimeJobStartDateTime",
				RunsAt:  runDate,
				HandlerParams: []any{
					eventId,
				},
				PayloadID: payloadId,
				Payload: map[string]any{
					"payloadId":        payloadId,
					"id":               int64(eventId),
					"producerClientId": "EventsToCloseProducer",
					"topic":            "EventsToClose",
					"table":            "events",
				},
				Source:     "Events",
				SourceType: "table",
				Topic:      "EventsToClose",
			}
			id, err := jobTask.CreateAndEnqueueJobTask(jobTask)
			if err != nil {
				log.Printf("Error creating job for Event: id=%d error=%s\n", eventId, err.Error())
				return
			}
			log.Printf("Created job for Event[%d] with ID %s\n", eventId, id)
		}()

		return nil
	})
	if err != nil {
		return 0, err
	}
	if !params.Publish && params.Mode == "scheduled" && opens_at != nil {
		go func() {
			runsAt := event.OpensAt
			runDate := time.Date(
				runsAt.UTC().Year(),
				runsAt.UTC().Month(),
				runsAt.UTC().Day(),
				runsAt.UTC().Hour(),
				runsAt.UTC().Minute(),
				0,
				0,
				runsAt.UTC().Location(),
			)
			log.Printf("[OpensAt] job scheduled at: %s\n", runDate)
			jobTaskID := uuid.New()
			payloadId := jobTaskID.String()
			jobTask := models.JobTask{
				Name:    fmt.Sprintf("Event_%d_OpensAt", eventId),
				JobType: "OneTimeJobStartDateTime",
				RunsAt:  runDate,
				HandlerParams: []any{
					eventId,
				},
				PayloadID: payloadId,
				Payload: map[string]any{
					"payloadId":        payloadId,
					"id":               int64(eventId),
					"producerClientId": "EventsToOpenProducer",
					"topic":            "EventsToOpen",
					"table":            "events",
				},
				Source:     "Events",
				SourceType: "table",
				Topic:      "EventsToOpen",
			}
			id, err := jobTask.CreateAndEnqueueJobTask(jobTask)
			if err != nil {
				log.Printf("Error creating job for Event: id=%d error=%s\n", eventId, err.Error())
				return
			}
			log.Printf("Created job for Event[%d] with ID %s\n", eventId, id)
		}()
	}
	if params.Publish {
		err := PublishEvent(event.ID)
		if err != nil {
			log.Printf("Failed to publish event: %s\n", err.Error())
			return 0, err
		}
	}
	return event.ID, err
}

func CreateNewTicket(params *types.CreateTicketRequestBody) (uint, error) {
	ticket := models.Ticket{
		Tier:     params.Tier,
		Type:     params.Type,
		Currency: params.Currency,
		Price:    params.Price,
		Limited:  params.Limited,
		Limit:    params.Limit,
		EventID:  params.EventID,
	}

	db := db.GetDb()
	err := db.Transaction(func(tx *gorm.DB) error {
		var event models.Event
		err := tx.
			Model(&models.Event{}).
			Where(&models.Event{ID: params.EventID}).
			Preload("Organization").
			Find(&event).
			Error
		if err != nil {
			err := fmt.Errorf("Event %d does not exist", params.EventID)
			return err
		}
		err = db.Create(&ticket).Error
		if err != nil {
			return err
		}
		if event.Organization.StripeAccountID == nil {
			err := errors.New("Could not create ticket. Reason: organization not properly setup")
			return err
		}
		const MINIMUM_UNITS float32 = 100
		unitAmount := ticket.Price
		if strings.ToLower(ticket.Currency) == "usd" {
			unitAmount = unitAmount * MINIMUM_UNITS
		}
		createParams := &stripe.ProductCreateParams{
			Name: stripe.String(ticket.Tier),
			DefaultPriceData: &stripe.ProductCreateDefaultPriceDataParams{
				Currency:          stripe.String("usd"),
				UnitAmountDecimal: stripe.Float64(float64(unitAmount)),
			},
			Metadata: map[string]string{
				"ticket_id": fmt.Sprint(ticket.ID),
				"event_id":  fmt.Sprint(event.ID),
				"org_id":    fmt.Sprint(event.Organization.ID),
			},
			Params: stripe.Params{
				StripeAccount: event.Organization.StripeAccountID,
			},
		}
		sc := lib.GetStripeClient()
		product, err := sc.V1Products.Create(context.Background(), createParams)
		err = tx.
			Model(&models.Ticket{}).
			Where(&models.Ticket{ID: ticket.ID}).
			Update("stripe_price_id", product.DefaultPrice.ID).
			Error
		return nil
	})
	if err != nil {
		log.Println("Error: ", err.Error())
		return 0, err
	}
	return ticket.ID, err
}

func CreateNewOrganization(params *types.CreateOrganizationRequestBody) (uint, error) {
	organization := models.Organization{
		Name:         params.Name,
		About:        params.About,
		Country:      params.Country,
		OwnerID:      params.OwnerID,
		ContactEmail: params.ContactEmail,
		Type:         params.Type,
		Slug:         slug.Make(params.Name),
	}

	db := db.GetDb()
	err := db.Transaction(func(tx *gorm.DB) error {
		err := tx.Create(&organization).Error
		if err != nil {
			return err
		}
		sc := lib.GetStripeClient()
		acc, err := sc.V1Accounts.Create(context.Background(), &stripe.AccountCreateParams{
			BusinessProfile: &stripe.AccountCreateBusinessProfileParams{
				Name:         stripe.String(organization.Name),
				SupportEmail: stripe.String(organization.ContactEmail),
			},
			BusinessType: stripe.String("non_profit"),
			Company: &stripe.AccountCreateCompanyParams{
				Name: stripe.String(organization.Name),
			},
			Type:     stripe.String("express"),
			Email:    stripe.String(organization.ContactEmail),
			Metadata: map[string]string{"organizationId": fmt.Sprintf("%d", organization.ID)},
			Capabilities: &stripe.AccountCreateCapabilitiesParams{
				CardPayments: &stripe.AccountCreateCapabilitiesCardPaymentsParams{Requested: stripe.Bool(true)},
				Transfers: &stripe.AccountCreateCapabilitiesTransfersParams{
					Requested: stripe.Bool(true),
				},
			},
		})
		if err != nil {
			log.Printf("Error creating account for organization: %s\n", err.Error())
			return errors.New("Error creating account for organization")
		}
		err = tx.
			Model(&models.Organization{}).
			Where(&models.Organization{ID: organization.ID}).
			Update("stripe_account_id", acc.ID).
			Error
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return organization.ID, err
}

func GetTicketsForEvent(id uint, isOwner bool) ([]*models.Ticket, error) {
	var tickets []*models.Ticket
	cond := models.Ticket{EventID: id}
	if !isOwner {
		cond.Status = "open"
	}
	db := db.GetDb()
	tx := db.Session(&gorm.Session{PrepareStmt: true})
	err := tx.
		Where(&cond).
		Order("created_at desc").
		Find(&tickets).Error

	if err != nil {
		return nil, err
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		for _, v := range tickets {
			var stats *models.TicketStats
			tx.
				Model(&models.Booking{}).
				Where(&models.Booking{TicketID: v.ID}).
				Select("SUM(qty) as reserved").
				Scan(&stats)
			stats.Free = v.Limit - stats.Reserved
			v.Stats = stats
		}
		return nil
	})
	return tickets, nil
}

func PublishEvent(id uint) error {
	db := db.GetDb()
	err := db.Transaction(func(tx *gorm.DB) error {
		err := tx.
			Model(&models.Event{}).
			Where(&models.Event{ID: id, Status: types.EVENT_DRAFT}).
			Update("status", types.EVENT_REGISTRATION).Error
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func GetTicket(id uint) (*models.Ticket, error) {
	var ticket models.Ticket
	db := db.GetDb()
	db.Model(&models.Ticket{}).Where(&models.Ticket{ID: id}).Preload("Event").First(&ticket)
	if ticket.ID < 1 {
		err := errors.New("Ticket not found")
		return nil, err
	}
	log.Printf("event: %v", ticket.Event.ID)
	return &ticket, nil
}

func GetTicketSeats(id uint) (free uint, reserved uint, err error) {
	db := db.GetDb()
	var ticket *models.Ticket
	tx := db.Session(&gorm.Session{PrepareStmt: true})
	tx.Where(&models.Ticket{ID: id}).First(&ticket)
	if ticket.ID < 1 {
		err := errors.New("Ticket not found")
		return 0, 0, err
	}
	var stats models.TicketStats
	tx.
		Model(&models.Booking{}).
		Where(&models.Booking{TicketID: id}).
		Select("SUM(qty) as reserved").
		Scan(&stats)

	freeSeats := ticket.Limit - stats.Reserved
	reservedSeats := stats.Reserved
	return freeSeats, reservedSeats, nil
}

func PublishTicket(id uint) error {
	db := db.GetDb()
	err := db.Model(&models.Ticket{}).Where(&models.Ticket{ID: id, Status: "draft"}).Update("status", "open").Error
	return err
}

func CloseTicket(id uint) error {
	db := db.GetDb()
	err := db.Model(&models.Ticket{}).Where(&models.Ticket{ID: id, Status: "open"}).Update("status", "closed").Error
	return err
}

func DeleteTicket(id uint) error {
	db := db.GetDb()
	err := db.Where(&models.Ticket{ID: id}).Update("status", "archived").Error
	return err
}

func CreateReservation(params *types.CreateBookingRequestBody, userId uint, csURL string, txId *string, csID *string, requestId *uuid.UUID) ([]uint, []string, error) {
	metadata := types.JSONB{
		"requestId": requestId.String(),
	}
	log.Printf("[metadata]: %v; %d items\n", metadata, len(params.Items))
	db := db.GetDb()
	reservationIDs := []uint{}
	errors := make([]string, 0)
	now := time.Now()
	expirationTime := now.Add(1 * time.Hour)
	// rd := lib.GetRedisClient()
	err := db.Transaction(func(tx *gorm.DB) error {
		/* txn := models.Transaction{
			Status:      types.TRANSACTION_PENDING,
			ReferenceID: requestId.String(),
		}
		err := tx.Create(&txn).Error
		if err != nil {
			return err
		} */
		/* val, err := rd.Get(context.Background(), requestId.String()).Result()
		if err != nil {
			log.Printf("Error retrieving cache value: %s\n", err.Error())
			return err
		} */
		txnId, err := uuid.Parse(*txId)
		if err != nil {
			log.Printf("Error parsing value: %s\n", err.Error())
			return err
		}
		log.Printf("[txnId]: %v", txnId)
		var txn models.Transaction
		if err = tx.
			Model(&models.Transaction{}).
			Where(&models.Transaction{ID: txnId}).
			First(&txn).
			Error; err != nil {
			log.Printf("Transaction not found [%s]: %s\n", txnId.String(), err.Error())
			return err
		}
		for _, v := range params.Items {
			var ticket models.Ticket
			err := tx.Where(&models.Ticket{ID: v.TicketID}).First(&ticket).Error
			if err != nil {
				return err
			}
			var count int64
			err = tx.
				Model(&models.Reservation{}).
				Select("COUNT(id)").
				Where(clause.IN{Column: "status", Values: []any{types.RESERVATION_PENDING, types.RESERVATION_COMPLETED}}).
				Where("valid_until > ?", time.Now()).
				Where(&models.Reservation{TicketID: v.TicketID}).
				Count(&count).
				Error
			if err != nil {
				return err
			}
			log.Println("CHECK 1")
			slotsLeft := ticket.Limit - uint(count)
			slots := slotsLeft - uint(v.Qty)
			slotsToTake := 0
			if slots > 0 && v.Qty > 0 {
				slotsToTake = int(math.Min(float64(slots), float64(v.Qty)))
			}

			if slotsToTake == 0 {
				err := fmt.Errorf("Ticket [%s] has no more slots available", ticket.Tier)
				log.Println(err)
				errors = append(errors, err.Error())
				continue
			}
			log.Println("CHECK 2")

			metadata["slots_wanted"] = v.Qty
			metadata["slots_taken"] = slotsToTake
			subtotal := ticket.Price * float32(v.Qty)
			r := models.Booking{
				TicketID:          v.TicketID,
				Qty:               v.Qty,
				Subtotal:          subtotal,
				Status:            types.BOOKING_PENDING,
				Currency:          "usd",
				UserID:            userId,
				EventID:           ticket.EventID,
				Metadata:          &metadata,
				CheckoutSessionId: csID,
				TransactionID:     &txnId,
				SlotsWanted:       uint(v.Qty),
				SlotsTaken:        uint(slotsToTake),
			}
			log.Println("CHECK 3")
			log.Printf("New Booking: %v\n", r)
			err = tx.Create(&r).Error
			if err != nil {
				err = fmt.Errorf("error in Booking transaction: %s\n", err.Error())
				log.Println(err.Error())
				return err
			}
			bookingId := r.ID
			log.Printf("[%s] New Booking: %d\n", txnId, bookingId)

			reservationIDs = append(reservationIDs, r.ID)
			runsAt := expirationTime
			go func() {
				runDate := time.Date(
					runsAt.UTC().Year(),
					runsAt.UTC().Month(),
					runsAt.UTC().Day(),
					runsAt.UTC().Hour(),
					runsAt.UTC().Minute(),
					0,
					0,
					runsAt.UTC().Location(),
				)
				log.Printf("[ValidUntil] job scheduled at: %s\n", runDate)
				jobTaskID := uuid.New()
				payloadId := jobTaskID.String()
				jobTask := models.JobTask{
					Name:    fmt.Sprintf("Event_%d_ValidUntil", bookingId),
					JobType: "OneTimeJobStartDateTime",
					RunsAt:  runDate,
					HandlerParams: []any{
						bookingId,
					},
					PayloadID: payloadId,
					Payload: map[string]any{
						"payloadId":        payloadId,
						"id":               int64(bookingId),
						"producerClientId": "PendingTransactionsProducer",
						"topic":            "PendingTransactions",
						"table":            "reservations",
					},
					Source:     "Reservations",
					SourceType: "table",
					Topic:      "PendingTransactions",
				}
				id, err := jobTask.CreateAndEnqueueJobTask(jobTask)
				if err != nil {
					log.Printf("Error creating job for Booking: id=%d error=%s\n", bookingId, err.Error())
					return
				}
				log.Printf("Created job for Booking[%d] with ID %s\n", bookingId, id)
			}()
			for range slotsToTake {
				reservation := models.Reservation{
					TicketID:   v.TicketID,
					BookingID:  r.ID,
					ValidUntil: expirationTime,
				}
				err = tx.Create(&reservation).Error
			}
			if err != nil {
				log.Printf("error in Reservation transaction: %s\n", err.Error())
				return err
			}
		}
		if len(errors) > 0 {
			err := fmt.Errorf("There were [%d] errors while adding Booking records", len(errors))
			return err
		}

		return nil
	})
	if err != nil {
		log.Printf("CreateReservation failed: %s\n", err.Error())
		return []uint{}, errors, err
	}

	return reservationIDs, nil, nil
}

func GetOrgReservations(id uint) ([]models.Booking, error) {
	var bookings []models.Booking
	db := db.GetDb()
	err := db.Where(&models.Booking{Event: &models.Event{OrganizerID: id}}).Preload("Event.Organization").Find(&bookings).Error
	return bookings, err
}
func GetOwnReservations(id uint) ([]models.Booking, error) {
	db := db.GetDb()
	var bookings []models.Booking
	err := db.
		Model(&models.Booking{}).
		Where(&models.Booking{UserID: id}).
		Not(&models.Booking{TransactionID: &uuid.Nil}).
		Preload("Event").
		Preload("Tickets").
		Preload("Transaction").
		Order("created_at DESC").
		Find(&bookings).
		Error
	return bookings, err
}

func CreateStripeCheckout(params *types.CreateBookingRequestBody, metadata map[string]string) (*string, *string, *string, error) {
	sc := lib.GetStripeClient()
	successUrl := fmt.Sprintf("%s/checkout/callback/success", os.Getenv("APP_HOST"))
	piParams := &stripe.CheckoutSessionCreatePaymentIntentDataParams{}
	meta := types.Metadata{}
	for k, v := range metadata {
		piParams.AddMetadata(k, v)
		meta[k] = v
	}
	log.Printf("[meta]: %v\n", meta)
	createParams := stripe.CheckoutSessionCreateParams{
		SuccessURL:        stripe.String(successUrl),
		UIMode:            stripe.String("hosted"),
		Mode:              stripe.String("payment"),
		PaymentIntentData: piParams,
		AfterExpiration: &stripe.CheckoutSessionCreateAfterExpirationParams{
			Recovery: &stripe.CheckoutSessionCreateAfterExpirationRecoveryParams{
				Enabled: stripe.Bool(true),
			},
		},
		InvoiceCreation: &stripe.CheckoutSessionCreateInvoiceCreationParams{
			Enabled: stripe.Bool(true),
		},
		Metadata: metadata,
	}

	db := db.GetDb()
	lineItems := []*stripe.CheckoutSessionCreateLineItemParams{}
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, v := range params.Items {
			var ticket models.Ticket
			err := tx.
				Where(&models.Ticket{ID: v.TicketID}).
				Preload("Event.Organization").
				First(&ticket).Error
			if err != nil {
				return err
			}
			stripeAccountId := ticket.Event.Organization.StripeAccountID
			createParams.Params = stripe.Params{
				StripeAccount: stripeAccountId,
			}
			priceId := ticket.StripePriceId
			price, err := sc.V1Prices.Retrieve(context.Background(), *priceId, &stripe.PriceRetrieveParams{
				Params: stripe.Params{
					StripeAccount: stripeAccountId,
				},
			})
			if err != nil {
				return err
			}

			log.Printf("price: %s %v\n", *priceId, price.UnitAmountDecimal)
			lineItems = append(lineItems, &stripe.CheckoutSessionCreateLineItemParams{
				Price:    priceId,
				Quantity: stripe.Int64(int64(v.Qty)),
			})
		}
		return nil
	})
	if err != nil {
		log.Printf("CreateStripeCheckout failed: %s\n", err.Error())
		return nil, nil, nil, err
	}
	createParams.LineItems = lineItems
	checkoutSession, err := sc.V1CheckoutSessions.Create(context.Background(), &createParams)
	if err != nil {
		log.Printf("CreateStripeCheckout failed: %s\n", err.Error())
		return nil, nil, nil, err
	}
	log.Printf("CheckoutSessionID: %s\n", checkoutSession.ID)
	requestId := metadata["requestId"]
	var txnId string
	err = db.Transaction(func(tx *gorm.DB) error {
		txn := &models.Transaction{
			Amount:            float64(checkoutSession.AmountTotal),
			Currency:          string(checkoutSession.Currency),
			CheckoutSessionId: &checkoutSession.ID,
			Status:            types.TRANSACTION_PENDING,
			ReferenceID:       requestId,
			SourceName:        "table",
			SourceValue:       "Booking",
			// Metadata:          &meta,
		}
		err := tx.Create(txn).Error
		if err != nil {
			return err
		}
		txnId = txn.ID.String()
		return nil
	})
	if err != nil {
		log.Printf("Error while creating Transaction: %s\n", err.Error())
		return nil, nil, nil, err
	}
	rd := lib.GetRedisClient()
	_, err = rd.SetEx(context.Background(), requestId, txnId, 10*time.Minute).Result()
	if err != nil {
		log.Printf("Error caching value [%s]: %s\n", txnId, err.Error())
	}

	return &checkoutSession.URL, &checkoutSession.ID, &txnId, nil
}

func UpdateEventStatus(id uint, newStatus types.EventStatus, oldStatus types.EventStatus) error {
	db := db.GetDb()
	log.Println("OpenEventStatus: Begin Transaction")
	err := db.Transaction(func(tx *gorm.DB) error {
		var event models.Event
		conds := &models.Event{ID: id, Status: oldStatus}
		err := tx.Where(conds).First(&event).Error
		if err != nil {
			log.Printf("Failed to update event status: %s\n", err.Error())
			return err
		}
		err = tx.
			Model(&models.Event{}).
			Where(conds).
			Updates(&models.Event{
				Status: newStatus,
				Mode:   "default",
			}).Error
		if err != nil {
			log.Printf("Event status update did not complete successfully: %s\n", err.Error())
			return err
		}
		err = tx.
			Model(&models.EventSubscription{}).
			Where(&models.EventSubscription{EventID: id, Status: "pending"}).
			Update("status", "done").
			Error
		if err != nil {
			log.Printf("EventSubscription update failed: %s\n", err.Error())
			return err
		}
		return nil
	})
	if err != nil {
		log.Printf("Error on transaction: %s\n", err.Error())
		return err
	}
	log.Println("OpenEventStatus: End Transaction")
	return nil
}

func EnqueueJobs() {
	scheduler, err := lib.GetScheduler()
	if err != nil {
		log.Printf("Error retrieving Scheduler instance: %s\n", err.Error())
		return
	}
	db := db.GetDb()
	err = db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		in1h := now.Add(1 * time.Hour)
		var events []models.Event
		err := tx.
			Where(&models.Event{Status: "notify", Mode: "scheduled"}).
			Where("opens_at BETWEEN ? AND ?", now, in1h).
			Find(&events).
			Error
		if err != nil {
			return err
		}
		jid, err := scheduler.NewJob(
			gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(time.Now().Add(30*time.Minute))),
			gocron.NewTask(func(n int) {
				log.Println("Some value:", n)
			}, 1),
		)
		if err != nil {
			return err
		}
		log.Printf("New job in ueue: %s\n", jid.ID().String())
		return nil
	})
	if err != nil {
		log.Printf("Error in boot Task: %s\n", err.Error())
		return
	}
}

func EncryptMessage(key []byte, message string) (string, error) {
	plaintext := []byte(message)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nonce, nonce, plaintext, nil)
	encodedString := hex.EncodeToString(cipherText)

	return encodedString, nil
}

func DecryptMessage(key []byte, message string) (*string, error) {
	cipherText, err := hex.DecodeString(message)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	decryptedData, err := gcm.Open(nil, cipherText[:gcm.NonceSize()], cipherText[gcm.NonceSize():], nil)
	if err != nil {
		return nil, err
	}
	decodedString := string(decryptedData)

	return &decodedString, nil
}
