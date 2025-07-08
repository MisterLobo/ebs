package common

import (
	"ebs/src/db"
	awslib "ebs/src/lib/aws"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

func KafkaPendingTransactionsConsumer(body string) {
	qname := "PendingTransactions"
	if !gjson.Valid(body) {
		log.Printf("[%s]: Received invalid json body. Aborting", qname)
		return
	}
	val := gjson.Get(body, "Message.id")
	log.Printf("[PendingTransactions] val: %f\n", val.Float())
	var payload types.JSONB
	err := json.Unmarshal([]byte(body), &payload)
	if err != nil {
		log.Printf("Error deserializing JSON: %s\n", err.Error())
		return
	}
	message := gjson.Get(body, "Message").String()
	var msg types.JSONB
	json.Unmarshal([]byte(message), &msg)
	id := gjson.Get(body, "Message.id").Float()
	bookingID := uint(id)
	log.Printf("[PendingTransactions]: %d", bookingID)
	// Update the reservations's status
	go func() {
		db := db.GetDb()
		err := db.Transaction(func(tx *gorm.DB) error {
			var booking models.Booking
			err := tx.
				Where(&models.Booking{ID: bookingID}).
				Preload("User").
				First(&booking).
				Error
			if err != nil {
				return err
			}
			if booking.Status == types.BOOKING_COMPLETED && booking.PaymentIntentId != nil {
				return nil
			}
			err = tx.
				Model(&models.Booking{}).
				Where(&models.Booking{ID: bookingID}).
				Update("status", types.BOOKING_EXPIRED).
				Error
			if err != nil {
				return err
			}
			err = tx.
				Model(&models.Reservation{}).
				Where(&models.Reservation{BookingID: bookingID}).
				Update("status", types.RESERVATION_CANCELED).
				Error
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			log.Printf("Error updating reservation status: %s\n", err.Error())
		}
	}()

	// UPDATE JOB
	payloadId := gjson.Get(body, "payloadId").String()
	go func() {
		db := db.GetDb()
		err := db.Transaction(func(tx *gorm.DB) error {
			err := tx.
				Where(&models.JobTask{PayloadID: payloadId}).
				Updates(&models.JobTask{Status: "done"}).
				Error
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			log.Printf("Error updating event status: %s\n", err.Error())
		}
	}()
}
func PendingTransactionsConsumer() {
	qname := "PendingTransactions"
	log.Printf("%s: Listening for messages...", qname)
	c := awslib.NewSQSConsumer(qname, func(body string) {
		KafkaPendingTransactionsConsumer(body)
	})
	c.Listen()
}

func PaymentTransactionUpdatesConsumer() {
	qname := utils.WithSuffix("PaymentTransactionUpdates")
	log.Printf("%s: Listening for messages...", qname)
	c := awslib.NewSQSConsumer(qname, func(body string) {
		if !gjson.Valid(body) {
			log.Printf("[%s]: Received invalid json body. Aborting", qname)
			return
		}
		val := gjson.Get(body, "id")
		log.Printf("[PaymentTransactionUpdates] val: %s\n", val.String())
		var payload types.JSONB
		err := json.Unmarshal([]byte(body), &payload)
		if err != nil {
			log.Printf("Error deserializing JSON: %s\n", err.Error())
			return
		}
		sId := payload["id"].(string)
		id, _ := uuid.Parse(sId)
		log.Printf("[TXN:%s] Beginning update...\n", sId)
		sConds := payload["conds"].(string)
		bConds := []byte(sConds)
		var conds models.Transaction
		json.Unmarshal(bConds, &conds)
		log.Printf("[TXN:%s] Conds: %v\n", id, conds)
		sUpdates := payload["updates"].(string)
		bUpdates := []byte(sUpdates)
		var updates models.Transaction
		json.Unmarshal(bUpdates, &updates)
		db := db.GetDb()
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.
				Model(&models.Transaction{}).
				Where(&conds).
				Updates(&updates).
				Error; err != nil {
				return err
			}
			log.Printf("[TXN:%s] Transaction committed successfully\n", id)
			return nil
		}); err != nil {
			log.Printf("Error updating Transaction [%s]: %s\n", id, err.Error())
			return
		}
		log.Printf("[TXN:%s] Finished update...\n", id)
	})
	c.Listen()
}

func KafkaPaymentTransactionUpdatesConsumer(spayload string) {
	qname := "PaymentTransactionUpdates"
	if !gjson.Valid(spayload) {
		log.Printf("[%s]: Received invalid json body. Aborting", qname)
		return
	}
	var payload types.JSONB
	err := json.Unmarshal([]byte(spayload), &payload)
	if err != nil {
		log.Printf("Error deserializing JSON: %s\n", err.Error())
		return
	}
	sId := gjson.Get(spayload, "id").String()
	log.Printf("[TXN:%s] Beginning update...\n", sId)
	sConds := gjson.Get(spayload, "conds").String()
	bConds := []byte(sConds)
	var conds models.Transaction
	if err := json.Unmarshal(bConds, &conds); err != nil {
		log.Printf("Error deserializing Transaction from buffer: %s\n", err.Error())
		return
	}
	sUpdates := gjson.Get(spayload, "updates").String()
	bUpdates := []byte(sUpdates)
	var updates models.Transaction
	if err := json.Unmarshal(bUpdates, &updates); err != nil {
		log.Printf("Error deserializing Transaction from buffer: %s\n", err.Error())
		return
	}
	db := db.GetDb()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Model(&models.Transaction{}).
			Where(&conds).
			Updates(&updates).
			Error; err != nil {
			return err
		}
		log.Printf("[TXN:%s] Transaction committed successfully\n", sId)
		return nil
	}); err != nil {
		log.Printf("Error updating Transaction [%s]: %s\n", sId, err.Error())
		return
	}
	log.Printf("[TXN:%s] Finished update...\n", sId)
}
