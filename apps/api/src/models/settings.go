package models

import (
	"ebs/src/types"

	"github.com/google/uuid"
)

type Setting struct {
	ID           uuid.UUID      `gorm:"primarykey;type:uuid;default:gen_random_uuid()" json:"id"`
	SettingKey   string         `gorm:"uniqueIndex:name" json:"setting_key"`
	SettingValue types.JSONBAny `gorm:"type:jsonb" json:"setting_value"`
	Group        string         `gorm:"uniqueIndex:name" json:"group,omitempty"`

	types.Timestamps
}
