package models

import (
	"ebs/src/types"

	"github.com/google/uuid"
)

type Transaction struct {
	ID uuid.UUID `gorm:"primarykey;type:uuid;default:gen_random_uuid()" json:"id"`

	BookingID   uint
	Currency    string
	Amount      float64
	SourceName  string
	SourceValue string
	ReferenceID string
	Status      types.TransactionStatus `gorm:"pending"`
	Metadata    types.JSONB

	types.Timestamps

	Booking Booking `gorm:"foreignKey:booking_id" json:"-"`
}
