package models

import (
	"ebs/src/types"
)

type Organization struct {
	ID                   uint            `json:"id"`
	Name                 string          `json:"name,omitempty"`
	About                string          `json:"about,omitempty"`
	Country              string          `json:"country,omitempty"`
	OwnerID              uint            `json:"owner_id,omitempty"`
	Type                 string          `gorm:"default:'standard'" json:"type,omitempty"`
	StripeAccountID      *string         `json:"stripe_account_id,omitempty"`
	Metadata             *types.Metadata `gorm:"type:jsonb" json:"metadata,omitempty"`
	ContactEmail         string          `json:"email,omitempty"`
	ConnectOnboardingURL *string         `json:"connect_onboarding_url,omitempty"`
	Status               string          `gorm:"default:'pending'" json:"status,omitempty"`
	Verified             bool            `gorm:"default:false" json:"verified,omitempty"`

	Events []Event `gorm:"foreignKey:organizer_id" json:"-"`
	Owner  User    `gorm:"foreignKey:owner_id" json:"-"`

	types.Timestamps
}
