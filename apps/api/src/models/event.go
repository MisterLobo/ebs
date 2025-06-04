package models

import (
	"ebs/src/lib"
	"ebs/src/types"
	"log"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID          uint              `gorm:"primarykey" json:"id"`
	Title       string            `json:"title,omitempty"`
	Name        string            `json:"name,omitempty"`
	About       *string           `json:"about,omitempty"`
	Type        string            `gorm:"default:'general'" json:"type"`
	Location    string            `json:"location,omitempty"`
	DateTime    time.Time         `json:"date_time,omitempty"`
	Status      types.EventStatus `gorm:"default:'draft'" json:"status,omitempty"` // `json:"status"`
	OrganizerID uint              `json:"organizer,omitempty"`
	Seats       uint              `json:"seats,omitempty"`
	CreatedBy   uint              `json:"created_by,omitempty"`
	Mode        string            `gorm:"default:'default'" json:"mode,omitempty"`
	OpensAt     *time.Time        `json:"opens_at,omitempty"`
	Deadline    time.Time         `json:"deadline,omitempty"`
	Metadata    *types.Metadata   `gorm:"type:jsonb" json:"metadata,omitempty"`
	Identifier  *string           `json:"resource_id"`
	TenantID    *uuid.UUID        `gorm:"type:uuid" json:"-"`

	Creator      User         `gorm:"foreignKey:created_by" json:"-"`
	Organization Organization `gorm:"foreignKey:organizer_id" json:"organization"`
	Tickets      []Ticket     `json:"tickets,omitempty"`
	Subscribers  []*User      `gorm:"many2many:event_subscriptions;joinForeignKey:SubscriberID;joinReferences:SubscriberID" json:"subscribers,omitempty"`
	// EventSubscriptions []EventSubscription `gorm:"foreignKey:event_id" json:"event_susbcriptions,omitempty"`

	types.Timestamps
}

type EventSubscription struct {
	ID           uint                          `gorm:"primarykey" json:"id"`
	EventID      uint                          `gorm:"primarykey" json:"event_id,omitempty"`
	SubscriberID uint                          `gorm:"primarykey" json:"subscriber_id,omitempty"`
	Status       types.EventSubscriptionStatus `gorm:"default:'notify'" json:"status,omitempty"`
	TenantID     *uuid.UUID                    `gorm:"type:uuid" json:"-"`

	// User  User  `gorm:"foreignKey:subscriber_id;references:id" json:"-"`
	// Event Event `gorm:"foreignKey:event_id" json:"event,omitempty"`

	types.Timestamps
}

func EventOpenProducer(id uint, payload types.JSONB) error {
	err := lib.KafkaProduceMessage("events_open_producer", "events-open", &payload)
	if err != nil {
		log.Printf("Error on producing message: %s\n", err.Error())
		return err
	}
	return nil
}

func EventCloseProducer(id uint, payload types.JSONB) error {
	err := lib.KafkaProduceMessage("events_close_producer", "events-close", &payload)
	if err != nil {
		log.Printf("Error on producting message: %s\n", err.Error())
		return err
	}
	return nil
}
