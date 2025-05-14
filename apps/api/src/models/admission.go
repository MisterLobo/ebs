package models

import "ebs/src/types"

type Admission struct {
	ID uint `json:"id"`

	ReservationID uint   `json:"reservation_id,omitempty"`
	Type          string `json:"type,omitempty"`
	Status        string `json:"status,omitempty"`

	Reservation Reservation `json:"reservation,omitempty"`

	types.Timestamps
}
