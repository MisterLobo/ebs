package common

import (
	"ebs/src/db"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
	"encoding/json"
	"log"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"gorm.io/gorm"
)

func EventsOpenConsumer() {
	log.Println("Setting up EventsOpenConsumer")
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BROKER"),
		"group.id":          "events_open_consumer",
		"auto.offset.reset": "smallest",
	})
	if err != nil {
		log.Printf("Error creating consumer: %s", err.Error())
		return
	}
	err = consumer.Subscribe("events-open", nil)
	if err != nil {
		log.Printf("Error on subscription: %s", err.Error())
		return
	}
	run := true
	for run == true {
		ev := consumer.Poll(100)
		switch e := ev.(type) {
		case *kafka.Message:
			var payload map[string]any = make(map[string]any)
			log.Println("Payload received")
			err := json.Unmarshal(e.Value, &payload)
			if err != nil {
				log.Printf("Error parsing payload: %s\n", err.Error())
				return
			}
			log.Println("Payload deserialized")
			idKey := payload["id"].(float64)
			eventId := uint(idKey)
			go utils.UpdateEventStatus(eventId, types.EVENT_OPEN)
			// UPDATE JOB
			go func() {
				db := db.GetDb()
				err := db.Transaction(func(tx *gorm.DB) error {
					payloadId := payload["payloadId"].(string)
					err := tx.Where(&models.JobTask{PayloadID: payloadId}).Updates(&models.JobTask{Status: "done"}).Error
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					log.Printf("Error updating job: %s\n", err.Error())
				}
			}()
			break
		case kafka.Error:
			log.Printf("Consumer for topic '%s' return an error: %s\n", "eventsopen", e.Error())
			run = false
		default:
			break
		}
	}
	log.Println("Received signal for topic. Closing")
	consumer.Close()
}

func EventsCloseConsumer() {
	log.Println("Setting up EventsCloseConsumer")
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BROKER"),
		"group.id":          "events_close_consumer",
		"auto.offset.reset": "smallest",
	})
	if err != nil {
		log.Printf("Error creating consumer: %s", err.Error())
		return
	}
	err = consumer.Subscribe("events-close", nil)
	if err != nil {
		log.Printf("Error on subscription: %s", err.Error())
		return
	}
	run := true
	for run == true {
		ev := consumer.Poll(100)
		switch e := ev.(type) {
		case *kafka.Message:
			var payload map[string]any = make(map[string]any)
			log.Println("Payload received")
			err := json.Unmarshal(e.Value, &payload)
			if err != nil {
				log.Printf("Error parsing payload: %s\n", err.Error())
				return
			}
			log.Println("Payload deserialized")
			idKey := payload["id"].(float64)
			eventId := uint(idKey)
			go utils.UpdateEventStatus(eventId, types.EVENT_CLOSED)
			break
		case kafka.Error:
			log.Printf("Consumer for topic '%s' return an error: %s\n", "eventsclose", e.Error())
			run = false
		default:
			break
		}
	}
	log.Println("Received signal for topic. Closing")
	consumer.Close()
}
