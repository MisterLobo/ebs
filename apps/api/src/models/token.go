package models

import (
	"ebs/src/types"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TokenType string

const (
	TokenTypeVerification TokenType = "verification"
	TokenTypeSession      TokenType = "session"
)

type Token struct {
	ID            uuid.UUID       `gorm:"primarykey;type:uuid;default:gen_random_uuid()" json:"-"`
	RequestedBy   uint            `gorm:"->;<-:create" json:"-"`
	RequesterType string          `gorm:"->;<-:create" json:"-"`
	Type          TokenType       `gorm:"->;<-:create;type:text" json:"-"`
	TokenName     string          `gorm:"->;<-:create" json:"-"`
	TokenValue    types.JSONB     `gorm:"->;<-:create;type:jsonb" json:"-"`
	TTL           uint            `gorm:"->;<-:create" json:"-"`
	Metadata      *types.Metadata `gorm:"->;<-:create;type:jsonb" json:"-"`
	ExpiresAt     time.Time       `gorm:"-"`
	Status        string          `gorm:"default:'pending'" json:"-"`

	types.Timestamps
}

func (t *Token) AfterFind(tx *gorm.DB) error {
	ca := *t.CreatedAt
	t.ExpiresAt = ca.Add(time.Duration(t.TTL) * time.Second)
	return nil
}
