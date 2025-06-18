package models

import (
	"ebs/src/types"

	"github.com/google/uuid"
)

type Notification struct {
	ID              uuid.UUID    `gorm:"primarykey;type:uuid;default:gen_random_uuid()" json:"id"`
	ReferenceSource string       `json:"ref_src"`
	ReferenceType   string       `json:"ref_name"`
	ReferenceValue  string       `json:"ref_value"`
	Title           string       `json:"title"`
	Description     *string      `json:"description"`
	ReferenceBody   *types.JSONB `gorm:"type:jsonb" json:"ref_body"`
	ActionType      string       `json:"action_type"`
	ActionData      *types.JSONB `gorm:"type:jsonb" json:"action_data"`
	Type            string       `json:"type"`

	types.Timestamps
}
