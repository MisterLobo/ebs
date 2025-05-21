package common

import (
	"ebs/src/db"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
	"encoding/json"
	"log"

	awslib "ebs/src/lib/aws"

	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

func EventsToOpenConsumer() {
	qname := "EventsToOpen"
	log.Printf("%s: Listening for messages...", qname)
	c := awslib.NewSQSConsumer(qname, func(body string) {
		if !gjson.Valid(body) {
			log.Printf("[%s]: Received invalid json body. Aborting", qname)
			return
		}
		val := gjson.Get(body, "Message.id")
		log.Printf("[EventsToOpen] val: %f\n", val.Float())
		var payload types.JSONB
		err := json.Unmarshal([]byte(body), &payload)
		if err != nil {
			log.Printf("Error deserializing JSON: %s\n", err.Error())
			return
		}
		message := payload["Message"].(string)
		var msg types.JSONB
		json.Unmarshal([]byte(message), &msg)
		id := msg["id"].(float64)
		eventId := uint(id)
		log.Printf("eventId: %d\n", eventId)
		// Update the event's status
		go utils.UpdateEventStatus(eventId, types.EVENT_TICKETS_OPEN, types.EVENT_TICKETS_NOTIFY)
		go func() {
			db := db.GetDb()
			err := db.Transaction(func(tx *gorm.DB) error {
				err := tx.
					Model(&models.EventSubscription{}).
					Where(&models.EventSubscription{EventID: eventId, Status: types.EVENT_SUBSCRIPTION_NOTIFY}).
					Update("status", types.EVENT_SUBSCRIPTION_ACTIVE).
					Error
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				log.Printf("Error updating event subscription for [%d]: %s\n", eventId, err.Error())
				return
			}
		}()
		// UPDATE JOB
		go func() {
			db := db.GetDb()
			err := db.Transaction(func(tx *gorm.DB) error {
				payloadId := msg["payloadId"].(string)
				err := tx.Where(&models.JobTask{PayloadID: payloadId}).Updates(&models.JobTask{Status: "done"}).Error
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				log.Printf("Error updating event status: %s\n", err.Error())
			}
		}()
	})
	c.Listen()
}

func EventsToCloseConsumer() {
	qname := "EventsToClose"
	c := awslib.NewSQSConsumer(qname, func(body string) {
		if !gjson.Valid(body) {
			log.Printf("[%s]: Received invalid json body. Aborting", qname)
			return
		}
		val := gjson.Get(body, "Message.id")
		log.Printf("[EventsToClose] val: %f\n", val.Float())
		var payload types.JSONB
		err := json.Unmarshal([]byte(body), &payload)
		if err != nil {
			log.Printf("Error deserializing JSON: %s\n", err.Error())
			return
		}
		message := payload["Message"].(string)
		var msg types.JSONB
		json.Unmarshal([]byte(message), &msg)
		id := msg["id"].(float64)
		eventId := uint(id)
		log.Printf("eventId: %d\n", eventId)
		// Update the event's status
		go utils.UpdateEventStatus(eventId, types.EVENT_TICKETS_CLOSED, types.EVENT_TICKETS_OPEN)
		go func() {
			db := db.GetDb()
			err := db.Transaction(func(tx *gorm.DB) error {
				err := tx.
					Model(&models.EventSubscription{}).
					Where(&models.EventSubscription{EventID: eventId}).
					Update("status", types.EVENT_TICKETS_CLOSED).
					Error
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				log.Printf("Error updating event subscription for [%d]: %s\n", eventId, err.Error())
				return
			}
		}()
		// UPDATE JOB
		go func() {
			db := db.GetDb()
			err := db.Transaction(func(tx *gorm.DB) error {
				payloadId := msg["payloadId"].(string)
				err := tx.Where(&models.JobTask{PayloadID: payloadId}).Updates(&models.JobTask{Status: "done"}).Error
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				log.Printf("Error updating event status: %s\n", err.Error())
			}
		}()
	})
	c.Listen()
}

func EventsToCompleteConsumer() {
	qname := "EventsToComplete"
	log.Printf("%s: Listening for messages...", qname)
	c := awslib.NewSQSConsumer(qname, func(body string) {
		if !gjson.Valid(body) {
			log.Printf("[%s]: Received invalid json body. Aborting", qname)
			return
		}
		val := gjson.Get(body, "Message.id")
		log.Printf("[EventsToComplete] val: %f\n", val.Float())
		var payload types.JSONB
		err := json.Unmarshal([]byte(body), &payload)
		if err != nil {
			log.Printf("Error deserializing JSON: %s\n", err.Error())
			return
		}
		message := payload["Message"].(string)
		var msg types.JSONB
		json.Unmarshal([]byte(message), &msg)
		id := msg["id"].(float64)
		eventId := uint(id)
		log.Printf("eventId: %d\n", eventId)
		// Update the event's status
		go utils.UpdateEventStatus(eventId, types.EVENT_COMPLETED, types.EVENT_TICKETS_CLOSED)
		go func() {
			db := db.GetDb()
			err := db.Transaction(func(tx *gorm.DB) error {
				err := tx.
					Model(&models.EventSubscription{}).
					Where(&models.EventSubscription{EventID: eventId}).
					Update("status", types.EVENT_COMPLETED).
					Error
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				log.Printf("Error updating event subscription for [%d]: %s\n", eventId, err.Error())
				return
			}
		}()
		// UPDATE JOB
		go func() {
			db := db.GetDb()
			err := db.Transaction(func(tx *gorm.DB) error {
				payloadId := msg["payloadId"].(string)
				err := tx.Where(&models.JobTask{PayloadID: payloadId}).Updates(&models.JobTask{Status: "done"}).Error
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				log.Printf("Error updating event status: %s\n", err.Error())
			}
		}()
	})
	c.Listen()
}
