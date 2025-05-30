package models

import (
	"ebs/src/types"

	"github.com/google/uuid"
)

type Booking struct {
	ID                uint                `gorm:"primarykey" json:"id"`
	TicketID          uint                `json:"ticket_id,omitempty"`
	Status            types.BookingStatus `json:"status,omitempty"`
	Qty               uint8               `json:"qty,omitempty"`
	UnitPrice         float32             `json:"unit_price,omitempty"`
	Subtotal          float32             `json:"subtotal"`
	Currency          string              `json:"currency,omitempty"`
	UserID            uint                `json:"user_id,omitempty"`
	EventID           uint                `json:"event_id,omitempty"`
	Metadata          *types.JSONB        `gorm:"type:jsonb" json:"metadata,omitempty"`
	CheckoutSessionId *string             `json:"checkout_session_id,omitempty"`
	PaymentIntentId   *string             `json:"payment_intent_id,omitempty"`
	TransactionID     *uuid.UUID          `json:"txn_id,omitempty"`
	SlotsWanted       uint                `json:"slots_wanted"`
	SlotsTaken        uint                `json:"slots_taken"`

	Event        *Event         `gorm:"foreignKey:event_id" json:"event,omitempty"`
	User         *User          `gorm:"foreignKey:user_id" json:"user,omitempty"`
	Ticket       *Ticket        `gorm:"foreignKey:ticket_id" json:"ticket,omitempty"`
	Tickets      []*Ticket      `gorm:"many2many:reservations;" json:"reserved_tickets,omitempty"`
	Reservations []*Reservation `json:"reservations,omitempty"`
	Transaction  *Transaction   `gorm:"foreignKey:transaction_id" json:"txn"`

	types.Timestamps
}
