package common

import (
	"context"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
	"encoding/json"
	"fmt"
	"log"
	"os"

	awslib "ebs/src/lib/aws"
	"ebs/src/lib/mailer"

	"firebase.google.com/go/v4/messaging"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

type Plucked struct {
	Email string
	UID   string
}

func subscribeAndSendToTopic(event *models.Event, topic string, unsubAfter bool, plucked ...*Plucked) {
	ctx := context.Background()
	fcmTokens := make([]string, 0)
	rd := lib.GetRedisClient()
	for _, item := range plucked {
		key := fmt.Sprintf("%s:fcm", item.UID)
		value := rd.JSONGet(ctx, key, "$.token").Val()
		fcmTokens = append(fcmTokens, value)
	}
	fcm, _ := lib.GetFirebaseMessaging()
	res, err := fcm.Send(ctx, &messaging.Message{
		Topic: topic,
		Data: map[string]string{
			"title": "Event Registration",
			"body":  fmt.Sprintf("Registration for %s is now closed", event.Title),
		},
	})
	if err != nil {
		log.Printf("[FCM] error sending notification message: %s", err.Error())
	} else {
		log.Printf("[FCM] notification sent to topic %s: %s", topic, res)
	}
	if unsubAfter {
		unsub, err := fcm.UnsubscribeFromTopic(ctx, fcmTokens, topic)
		if err != nil {
			log.Printf("[FCM] could not unsubscribe to topic %s: %s", topic, err.Error())
			return
		}
		log.Printf("[FCM] unsubscribed to topic %s: %v", topic, unsub)
	}
}

func sendOpenEventNotifications(eventId uint) {
	var event models.Event
	var plucked []*Plucked
	db := db.GetDb()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Model(&models.Event{}).
			Where(&models.Event{ID: eventId}).
			Preload("Creator").
			First(&event).
			Error; err != nil {
			return err
		}
		var subscriberIDs []uint
		if err := tx.
			Model(&models.EventSubscription{}).
			Where(&models.EventSubscription{EventID: eventId}).
			Select("subscriber_id").
			Pluck("subscriber_id", &subscriberIDs).
			Error; err != nil {
			return err
		}
		if err := tx.
			Model(&models.User{}).
			Distinct("email").
			Where("id IN (?)", subscriberIDs).
			Select("email", "uid").
			Find(&plucked).
			Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Printf("[EventsToOpenConsumer] Error on running database transaction: %s\n", err.Error())
		return
	}

	go subscribeAndSendToTopic(&event, utils.WithSuffix(fmt.Sprintf("EventsToOpen_%d", eventId)), true, plucked...)

	var emails []string
	for _, pluck := range plucked {
		emails = append(emails, pluck.Email)
	}
	senderFrom := os.Getenv("SMTP_FROM")
	input := &lib.SendMailInput{
		Subject:  fmt.Sprintf("Silver Elven Event Notification: %s", event.Title),
		From:     senderFrom,
		FromName: "noreply",
		To: []string{
			event.Creator.Email,
		},
		ReplyTo: event.Organization.ContactEmail,
		Bcc:     emails,
		Body: fmt.Sprintf(`
			<p>Registration for Event <b>%s</b> is now open</p>
			<p>What: %s</p>
			<p>Where: %s</p>
			<p>When: %s</p>
			<p>Book now via this link <a href="%s/%s/event/%d/tickets">here</a></p>
			<p>This is a system-generated message. Do not reply to this email.</p>
			`,
			event.Title,
			event.Title,
			event.Location,
			event.DateTime,
			os.Getenv("APP_HOST"),
			event.Name,
			event.ID,
		),
		Html: true,
	}
	if err := mailer.NewMailerMessage(input); err != nil {
		log.Printf("[mailer] Error sending message: %s\n", err.Error())
		return
	}
}
func KafkaEventsToOpenConsumer(spayload string) {
	val := gjson.Get(spayload, "id")
	topic := gjson.Get(spayload, "topic").String()
	if !gjson.Valid(spayload) {
		log.Printf("[%s]: Received invalid json body. Aborting", topic)
		return
	}
	log.Printf("[%s] val: %f\n", topic, val.Float())
	payloadId := gjson.Get(spayload, "payloadId").String()
	var payload types.JSONB
	if err := json.Unmarshal([]byte(spayload), &payload); err != nil {
		log.Printf("[%s] Error deserializing JSON: %s\n", topic, err.Error())
		return
	}
	eventId := uint(val.Int())
	log.Printf("eventId: %d\n", eventId)
	go utils.UpdateEventStatus(eventId, types.EVENT_REGISTRATION, types.EVENT_TICKETS_NOTIFY)
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
	go sendOpenEventNotifications(eventId)
	// UPDATE JOB
	go func() {
		db := db.GetDb()
		err := db.Transaction(func(tx *gorm.DB) error {
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
}

func sendClosedEventNotifications(eventId uint) {
	var event models.Event
	var plucked []*Plucked
	db := db.GetDb()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where(&models.Event{ID: eventId}).
			Preload("Creator").
			Preload("Organization").
			First(&event).
			Error; err != nil {
			return err
		}
		var guests []uint
		if err := tx.
			Model(&models.Booking{}).
			Where("event_id = ?", eventId).
			Select("user_id").
			Pluck("user_id", &guests).
			Error; err != nil {
			return err
		}
		if err := tx.
			Model(&models.User{}).
			Distinct("email").
			Where("id IN (?)", guests).
			Select("email", "uid").
			Find(&plucked).
			Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Printf("[EventsToCloseConsumer] Error on running database transaction: %s\n", err.Error())
		return
	}

	go subscribeAndSendToTopic(&event, utils.WithSuffix(fmt.Sprintf("EventsToClose_%d", eventId)), true, plucked...)

	var emails []string
	for _, pluck := range plucked {
		emails = append(emails, pluck.Email)
	}
	senderFrom := os.Getenv("SMTP_FROM")
	input := &lib.SendMailInput{
		Subject:  fmt.Sprintf("Silver Elven Event Notification: %s", event.Title),
		From:     senderFrom,
		FromName: event.Organization.Name,
		Bcc:      emails,
		To: []string{
			event.Creator.Email,
		},
		ReplyTo: event.Organization.ContactEmail,
		Body: fmt.Sprintf(`
			<p>Registration for Event <b>%s</b> is now closed. Ticket admissions are now open</p>
			<p>Event Details</p>
			<p>What: %s</p>
			<p>Where: %s</p>
			<p>When: %s</p>
			<p>Go to the location of event before tickets admission closes at %s</p>
			<p>This is a system-generated message. Do not reply to this email.</p>
			`,
			event.Title,
			event.Title,
			event.Location,
			event.DateTime,
			event.Deadline,
		),
		Html: true,
	}
	if err := mailer.NewMailerMessage(input); err != nil {
		log.Printf("[mailer] Error sending message: %s\n", err.Error())
		return
	}
}
func KafkaEventsToCloseConsumer(spayload string) {
	val := gjson.Get(spayload, "id")
	topic := gjson.Get(spayload, "topic").String()
	if !gjson.Valid(spayload) {
		log.Printf("[%s]: Received invalid json body. Aborting", topic)
		return
	}
	log.Printf("[%s] val: %f\n", topic, val.Float())
	payloadId := gjson.Get(spayload, "payloadId").String()
	var payload types.JSONB
	if err := json.Unmarshal([]byte(spayload), &payload); err != nil {
		log.Printf("[%s] Error deserializing JSON: %s\n", topic, err.Error())
		return
	}
	eventId := uint(val.Int())
	log.Printf("eventId: %d\n", eventId)
	go utils.UpdateEventStatus(eventId, types.EVENT_ADMISSION, types.EVENT_REGISTRATION)
	go sendClosedEventNotifications(eventId)
	// UPDATE JOB
	go func() {
		db := db.GetDb()
		err := db.Transaction(func(tx *gorm.DB) error {
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
}

func sendCompletedEventNotifications(eventId uint) {
	var event models.Event
	var plucked []*Plucked
	db := db.GetDb()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where(&models.Event{ID: eventId}).
			Preload("Creator").
			Preload("Organization").
			First(&event).
			Error; err != nil {
			return err
		}
		var guests []uint
		if err := tx.
			Model(&models.Booking{}).
			Where("event_id = ?", eventId).
			Select("user_id").
			Pluck("user_id", &guests).
			Error; err != nil {
			return err
		}
		if err := tx.
			Model(&models.User{}).
			Distinct("email").
			Where("id IN (?)", guests).
			Select("email", "uid").
			Find(&plucked).
			Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Printf("[EventsToCompleteConsumer] Error on running database transaction: %s\n", err.Error())
		return
	}

	go subscribeAndSendToTopic(&event, utils.WithSuffix(fmt.Sprintf("EventsToComplete_%d", eventId)), true, plucked...)

	var emails []string
	for _, pluck := range plucked {
		emails = append(emails, pluck.Email)
	}
	senderFrom := os.Getenv("SMTP_FROM")
	input := &lib.SendMailInput{
		Subject:  fmt.Sprintf("Silver Elven Event Notification: %s", event.Title),
		From:     senderFrom,
		FromName: event.Organization.Name,
		Bcc:      emails,
		To: []string{
			event.Creator.Email,
		},
		ReplyTo: event.Organization.ContactEmail,
		Body: fmt.Sprintf(`
			<p>Ticket admission for Event <b>%s</b> is now closed.</p>
			<p>Event Details</p>
			<p>What: %s</p>
			<p>Where: %s</p>
			<p>When: %s</p>
			<p>You can view the event page <a href="%s/%s/event/%d/tickets">here</a></p>
			<p>This is a system-generated message. Do not reply to this email.</p>
			`,
			event.Title,
			event.Title,
			event.Location,
			event.DateTime,
			os.Getenv("APP_HOST"),
			event.Name,
			event.ID,
		),
		Html: true,
	}
	if err := mailer.NewMailerMessage(input); err != nil {
		log.Printf("[mailer] Error sending message: %s\n", err.Error())
		return
	}
}
func KafkaEventsToCompleteConsumer(spayload string) {
	val := gjson.Get(spayload, "id")
	topic := gjson.Get(spayload, "topic").String()
	if !gjson.Valid(spayload) {
		log.Printf("[%s]: Received invalid json body. Aborting", topic)
		return
	}
	log.Printf("[%s] val: %f\n", topic, val.Float())
	payloadId := gjson.Get(spayload, "payloadId").String()
	var payload types.JSONB
	if err := json.Unmarshal([]byte(spayload), &payload); err != nil {
		log.Printf("[%s] Error deserializing JSON: %s\n", topic, err.Error())
		return
	}
	eventId := uint(val.Int())
	log.Printf("eventId: %d\n", eventId)
	go utils.UpdateEventStatus(eventId, types.EVENT_COMPLETED, types.EVENT_ADMISSION)
	go sendCompletedEventNotifications(eventId)
	// UPDATE JOB
	go func() {
		db := db.GetDb()
		err := db.Transaction(func(tx *gorm.DB) error {
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
}

func EventsToOpenConsumer() {
	qname := utils.WithSuffix("EventsToOpen")
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
		go utils.UpdateEventStatus(eventId, types.EVENT_REGISTRATION, types.EVENT_TICKETS_NOTIFY)
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
		go sendOpenEventNotifications(eventId)
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
	qname := utils.WithSuffix("EventsToClose")
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
		go utils.UpdateEventStatus(eventId, types.EVENT_ADMISSION, types.EVENT_REGISTRATION)
		go func() {
			db := db.GetDb()
			err := db.Transaction(func(tx *gorm.DB) error {
				err := tx.
					Model(&models.EventSubscription{}).
					Where(&models.EventSubscription{EventID: eventId}).
					Update("status", types.EVENT_ADMISSION).
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
		go sendClosedEventNotifications(eventId)
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
	qname := utils.WithSuffix("EventsToComplete")
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
		go utils.UpdateEventStatus(eventId, types.EVENT_COMPLETED, types.EVENT_ADMISSION)
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
		go sendCompletedEventNotifications(eventId)
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

func KafkaEmailsToSendConsumer(spayload string) {
	if !gjson.Valid(spayload) {
		log.Println("Received invalid json body. Aborting")
		return
	}
	from := gjson.Get(spayload, "from").String()
	fromName := gjson.Get(spayload, "from-name").String()
	subject := gjson.Get(spayload, "subject").String()
	log.Printf("from [%s] with subject: %s\n", from, subject)

	toArr := gjson.Get(spayload, "to").Array()
	to := make([]string, 0)
	for _, item := range toArr {
		to = append(to, item.String())
	}
	ccArr := gjson.Get(spayload, "cc").Array()
	cc := make([]string, 0)
	for _, item := range ccArr {
		cc = append(cc, item.String())
	}
	bccArr := gjson.Get(spayload, "bcc").Array()
	bcc := make([]string, 0)
	for _, item := range bccArr {
		bcc = append(bcc, item.String())
	}
	replyTo := gjson.Get(spayload, "reply-to").String()

	var body types.JSONB
	if err := json.Unmarshal([]byte(spayload), &body); err != nil {
		log.Printf("error deserializing json: %s\n", err.Error())
		return
	}
	go func() {
		input := &lib.SendMailInput{
			From:     from,
			FromName: fromName,
			To:       to,
			Cc:       cc,
			Bcc:      bcc,
			ReplyTo:  replyTo,
			Subject:  body["subject"].(string),
			Body:     body["body"].(string),
			Html:     body["html"].(bool),
		}
		if err := lib.SendMail(input); err != nil {
			log.Printf("[MAILER] error sending email: %s\n", err.Error())
			return
		}
		log.Printf("[MAILER]: an email has been sent to %s\n", to)
	}()
}

func EmailsToSendConsumer() {
	qname := utils.WithSuffix("EmailsToSend")
	log.Printf("%s: Listening for messages...", qname)
	c := awslib.NewSQSConsumer(qname, func(spayload string) {
		if !gjson.Valid(spayload) {
			log.Printf("[%s]: Received invalid json body. Aborting", qname)
			return
		}
		from := gjson.Get(spayload, "from").String()
		fromName := gjson.Get(spayload, "from-name").String()
		subject := gjson.Get(spayload, "subject").String()
		log.Printf("from [%s] with subject: %s\n", from, subject)

		toArr := gjson.Get(spayload, "to").Array()
		to := make([]string, 0)
		for _, item := range toArr {
			to = append(to, item.String())
		}
		ccArr := gjson.Get(spayload, "cc").Array()
		cc := make([]string, 0)
		for _, item := range ccArr {
			cc = append(cc, item.String())
		}
		bccArr := gjson.Get(spayload, "bcc").Array()
		bcc := make([]string, 0)
		for _, item := range bccArr {
			bcc = append(bcc, item.String())
		}
		replyTo := gjson.Get(spayload, "reply-to").String()
		var body types.JSONB
		if err := json.Unmarshal([]byte(spayload), &body); err != nil {
			log.Printf("error deserializing json: %s\n", err.Error())
			return
		}
		go func() {
			input := &lib.SendMailInput{
				From:     from,
				FromName: fromName,
				To:       to,
				Cc:       cc,
				Bcc:      bcc,
				ReplyTo:  replyTo,
				Subject:  body["subject"].(string),
				Body:     body["body"].(string),
				Html:     body["html"].(bool),
			}
			if err := lib.SendMail(input); err != nil {
				log.Printf("[MAILER] error sending email: %s\n", err.Error())
				return
			}
			log.Printf("[MAILER]: an email has been sent to %s\n", to)
		}()
	})
	c.Listen()
}
