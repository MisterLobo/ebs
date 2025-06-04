package models

import "github.com/google/uuid"

type TrailLog struct {
	ID        uuid.UUID `gorm:"primarykey;type:uuid;default:gen_random_uuid()" json:"id"`
	Type      string
	Initiator string
	Group     string
}
