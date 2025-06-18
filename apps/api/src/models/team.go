package models

import (
	"ebs/src/types"

	"github.com/google/uuid"
)

type Team struct {
	ID             uint   `gorm:"primarykey" json:"id"`
	Name           string `json:"name,omitempty"`
	OwnerID        uint   `json:"owner_id"`
	OrganizationID uint   `json:"organization_id,omitempty"`
	Status         string `json:"status,omitempty"`

	Owner        User          `gorm:"foreignKey:owner_id" json:"-"`
	Organization *Organization `gorm:"foreignKey:organization_id" json:"-"`
	Members      []*User       `gorm:"many2many:team_members;References:ID;joinReferences:UserID" json:"members,omitempty"`
	TenantID     *uuid.UUID    `gorm:"type:uuid" json:"-"`
	Identifier   *string       `gorm:"<-:create" json:"resource_id"`

	types.Timestamps
}

type TeamMember struct {
	ID     uint   `gorm:"primarykey" json:"id"`
	TeamID uint   `gorm:"primaryKey" json:"team_id,omitempty"`
	UserID uint   `gorm:"primaryKey" json:"member_id,omitempty"`
	Role   string `json:"role,omitempty"`
	Status string `json:"status,omitempty"`

	types.Timestamps
}
