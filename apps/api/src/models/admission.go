package models

import (
	"ebs/src/types"

	"github.com/google/uuid"
)

type Admission struct {
	ID uint `gorm:"primarykey" json:"id"`

	By            uint       `json:"by,omitempty"`
	ReservationID uint       `json:"reservation_id,omitempty"`
	Type          string     `json:"type,omitempty"`
	Status        string     `json:"status,omitempty"`
	TenantID      *uuid.UUID `gorm:"type:uuid" json:"-"`

	Reservation *Reservation `json:"reservation,omitempty"`
	AdmittedBy  *User        `gorm:"foreignKey:by" json:"-"`

	types.Timestamps
}
