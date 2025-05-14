package models

import "ebs/src/types"

type Booking struct {
	ID        uint    `gorm:"primarykey" json:"id"`
	TicketID  uint    `json:"ticket_id,omitempty"`
	Status    string  `json:"status,omitempty"`
	Qty       uint8   `json:"qty,omitempty"`
	UnitPrice float32 `json:"unit_price,omitempty"`
	Subtotal  float32 `json:"subtotal,omitempty"`
	Currency  string  `json:"currency,omitempty"`
	UserID    uint    `json:"user_id,omitempty"`
	EventID   uint    `json:"event_id,omitempty"`

	Event        *Event         `gorm:"foreignKey:event_id" json:"event,omitempty"`
	User         *User          `gorm:"foreignKey:user_id" json:"user,omitempty"`
	Tickets      []*Ticket      `gorm:"many2many:reservations;" json:"reserved_tickets,omitempty"`
	Reservations []*Reservation `json:"reservations,omitempty"`

	types.Timestamps
}
