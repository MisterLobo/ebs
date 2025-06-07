package models

import (
	"ebs/src/types"
	"time"

	"github.com/google/uuid"
)

type Reservation struct {
	ID         uint       `gorm:"primarykey" json:"id"`
	TicketID   uint       `json:"ticket_id,omitempty"`
	BookingID  uint       `json:"booking_id,omitempty"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
	ShareURL   string     `json:"share_url,omitempty"`
	Status     string     `gorm:"default:'pending'" json:"status,omitempty"`
	TenantID   *uuid.UUID `gorm:"type:uuid" json:"-"`

	Ticket  Ticket  `json:"ticket,omitempty"`
	Booking Booking `json:"booking,omitempty"`

	types.Timestamps
}
