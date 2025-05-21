package models

import (
	"ebs/src/lib"
	"ebs/src/types"
	"log"
	"time"
)

type Event struct {
	ID          uint              `json:"id"`
	Title       string            `json:"title,omitempty"`
	Name        string            `json:"name,omitempty"`
	About       *string           `json:"about,omitempty"`
	Type        string            `json:"type,default'general'"`
	Location    string            `json:"location,omitempty"`
	DateTime    time.Time         `json:"date_time,omitempty"`
	Status      types.EventStatus `gorm:"default:'draft'" json:"status,omitempty"` // `json:"status"`
	OrganizerID uint              `json:"organizer,omitempty"`
	Seats       uint              `json:"seats,omitempty"`
	CreatedBy   uint              `json:"created_by,omitempty"`
	Mode        string            `gorm:"default:'default'" json:"mode,omitempty"`
	OpensAt     *time.Time        `json:"opens_at,omitempty"`
	Deadline    time.Time         `json:"deadline,omitempty"`

	Creator            User                `gorm:"foreignKey:created_by" json:"-"`
	Organization       Organization        `gorm:"foreignKey:organizer_id" json:"-"`
	Tickets            []Ticket            `json:"tickets,omitempty"`
	Subscribers        []*User             `gorm:"many2many:event_subscriptions;" json:"subscribers,omitempty"`
	EventSubscriptions []EventSubscription `gorm:"foreignKey:event_id" json:"event_susbcriptions,omitempty"`

	types.Timestamps
}

type EventSubscription struct {
	ID           uint                          `gorm:"primarykey" json:"id"`
	EventID      uint                          `json:"event_id,omitempty"`
	SubscriberID uint                          `json:"subscriber_id,omitempty"`
	Status       types.EventSubscriptionStatus `gorm:"default:'notify'" json:"status,omitempty"`

	User  User  `gorm:"foreignKey:subscriber_id" json:"-"`
	Event Event `gorm:"foreignKey:event_id" json:"event,omitempty"`

	types.Timestamps
}

func EventOpenProducer(id uint, payload map[string]any) error {
	err := lib.KafkaProduceMessage("events_open_producer", "events-open", payload)
	if err != nil {
		log.Printf("Error on producing message: %s\n", err.Error())
		return err
	}
	return nil
}

func EventCloseProducer(id uint, payload map[string]any) error {
	err := lib.KafkaProduceMessage("events_close_producer", "events-close", payload)
	if err != nil {
		log.Printf("Error on producting message: %s\n", err.Error())
		return err
	}
	return nil
}
