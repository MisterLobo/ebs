package models

import (
	"database/sql/driver"
	"ebs/src/types"

	"github.com/google/uuid"
)

type TicketStatus types.Status

func (self *TicketStatus) Scan(value interface{}) error {
	*self = TicketStatus(value.([]byte))
	return nil
}

func (self TicketStatus) Value() (driver.Value, error) {
	return string(self), nil
}

type Ticket struct {
	ID            uint            `gorm:"primarykey" json:"id"`
	Type          string          `json:"type,omitempty"`
	Tier          string          `json:"tier,omitempty"`
	Status        string          `gorm:"default:'draft'" json:"status,omitempty"`
	Price         float32         `json:"price"`
	Currency      string          `json:"currency,omitempty"`
	Limited       bool            `json:"limited"`
	Limit         uint            `json:"limit"`
	EventID       uint            `json:"event_id,omitempty"`
	StripePriceId *string         `json:"-"`
	Metadata      *types.Metadata `gorm:"type:jsonb" json:"metadata"`
	Identifier    *string         `json:"resource_id"`
	TenantID      *uuid.UUID      `gorm:"type:uuid" json:"-"`

	Event    Event     `json:"event,omitempty"`
	Bookings []Booking `gorm:"many2many:reservations;" json:"bookings,omitempty"`

	Stats *TicketStats `gorm:"-" json:"stats,omitempty"`

	types.Timestamps
}

type TicketStats struct {
	TicketID uint `json:"ticket_id,omitempty"`
	Free     uint `json:"free,omitempty"`
	Reserved uint `json:"reserved,omitempty"`
}

type TicketTransfer struct {
	ID            string `json:"-"`
	ReservationID uint   `json:"reservation_id,omitempty"`
	OldOwnerID    uint   `json:"-"`
	NewOwnerID    uint   `json:"owner_id"`
	Status        string `json:"status,omitempty"`

	types.Timestamps
}
