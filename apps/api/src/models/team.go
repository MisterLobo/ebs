package models

import "ebs/src/types"

type Team struct {
	ID             uint   `gorm:"primarykey" json:"id"`
	Name           string `json:"name,omitempty"`
	OwnerID        uint   `json:"owner_id"`
	OrganizationID uint   `json:"organization_id,omitempty"`
	Status         string `json:"status,omitempty"`

	Owner        User         `gorm:"foreignKey:owner_id" json:"-"`
	Organization Organization `gorm:"foreignKey:organization_id" json:"-"`
	Members      []User       `gorm:"many2many:team_members;" json:"members,omitempty"`

	types.Timestamps
}

type TeamMember struct {
	ID       uint   `gorm:"primarykey" json:"id"`
	TeamID   uint   `json:"team_id,omitempty"`
	MemberID uint   `json:"member_id,omitempty"`
	Role     string `json:"role,omitempty"`
	Status   string `json:"status,omitempty"`

	InnerRole Role `gorm:"foreignKey:role" json:"-"`
	Team      Team `gorm:"foreignKey:team_id" json:"-"`
	Member    User `gorm:"foreignKey:member_id" json:"-"`

	types.Timestamps
}
