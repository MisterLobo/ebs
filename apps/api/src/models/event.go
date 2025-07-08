package models

import (
	"ebs/src/types"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Event struct {
	ID          uint              `gorm:"primarykey" json:"id"`
	Title       string            `json:"title,omitempty"`
	Name        string            `json:"name,omitempty"`
	About       *string           `json:"about,omitempty"`
	Type        string            `json:"type"`
	Location    string            `json:"location,omitempty"`
	DateTime    *time.Time        `json:"date_time,omitempty"`
	Status      types.EventStatus `gorm:"default:'draft'" json:"status,omitempty"`
	OrganizerID uint              `json:"organizer,omitempty"`
	Seats       uint              `json:"seats,omitempty"`
	CreatedBy   uint              `json:"created_by,omitempty"`
	Mode        string            `gorm:"default:'default'" json:"mode,omitempty"`
	OpensAt     *time.Time        `json:"opens_at,omitempty"`
	Deadline    *time.Time        `json:"deadline,omitempty"`
	Metadata    *types.Metadata   `gorm:"type:jsonb" json:"metadata,omitempty"`
	Identifier  *string           `gorm:"<-:create" json:"resource_id"`
	TenantID    *uuid.UUID        `gorm:"type:uuid" json:"-"`
	Category    string            `gorm:"default:'uncategorized'" json:"category"`
	Timezone    string            `gorm:"default:'UTC'" json:"timezone"`
	CalEventID  *string           `json:"-"`

	Creator      User         `gorm:"foreignKey:created_by" json:"-"`
	Organization Organization `gorm:"foreignKey:organizer_id" json:"organization"`
	Tickets      []*Ticket    `json:"tickets,omitempty"`
	Subscribers  []*User      `gorm:"many2many:event_subscriptions;joinForeignKey:SubscriberID;joinReferences:SubscriberID" json:"subscribers,omitempty"`

	types.Timestamps
}

func (e *Event) AfterFind(tx *gorm.DB) error {
	if e.Timezone != "" {
		l, err := time.LoadLocation(e.Timezone)
		if err != nil {
			return err
		}
		if e.DateTime != nil {
			dt := e.DateTime.In(l)
			e.DateTime = &dt
		}

		if e.OpensAt != nil {
			dt := e.OpensAt.In(l)
			e.OpensAt = &dt
		}
		if e.Deadline != nil {
			dt := e.Deadline.In(l)
			e.Deadline = &dt
		}
	}
	return nil
}

type EventSubscription struct {
	ID           uint                          `gorm:"primarykey" json:"id"`
	EventID      uint                          `gorm:"primarykey" json:"event_id,omitempty"`
	SubscriberID uint                          `gorm:"primarykey" json:"subscriber_id,omitempty"`
	Status       types.EventSubscriptionStatus `gorm:"default:'notify'" json:"status,omitempty"`
	TenantID     *uuid.UUID                    `gorm:"type:uuid" json:"-"`
	Identifier   *string                       `gorm:"<-:create" json:"resource_id"`

	Event      *Event `gorm:"foreignKey:event_id" json:"event,omitempty"`
	Subscriber *User  `gorm:"foreignKey:subscriber_id" json:"-"`

	types.Timestamps
}
