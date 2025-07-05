package models

import (
	"ebs/src/types"

	"github.com/google/uuid"
)

type Account struct {
	ID          uuid.UUID    `gorm:"primarykey;type:uuid;default:gen_random_uuid()" json:"-"`
	Name        string       `json:"-"`
	OwnerID     uint         `json:"-"`
	OwnerType   string       `json:"-"`
	AccountType string       `json:"-"`
	Session     *types.JSONB `gorm:"type:jsonb" json:"-"`
	Metadata    *types.JSONB `gorm:"type:jsonb" json:"-"`
	Status      string       `json:"-"`

	types.Timestamps
}
