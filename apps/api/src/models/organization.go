package models

import (
	"ebs/src/types"
)

type Organization struct {
	ID                   uint                   `gorm:"primarykey;uniqueIndex:slugid" json:"id"`
	Name                 string                 `json:"name,omitempty"`
	About                string                 `json:"about,omitempty"`
	Country              string                 `json:"country,omitempty"`
	OwnerID              uint                   `json:"owner_id,omitempty"`
	Type                 types.OrganizationType `gorm:"default:'standard'" json:"type,omitempty"`
	StripeAccountID      *string                `json:"stripe_account_id,omitempty"`
	Metadata             *types.Metadata        `gorm:"type:jsonb" json:"metadata,omitempty"`
	ContactEmail         string                 `json:"email,omitempty"`
	ConnectOnboardingURL *string                `json:"connect_onboarding_url,omitempty"`
	Status               string                 `gorm:"default:'pending'" json:"status,omitempty"`
	Verified             bool                   `gorm:"default:false" json:"verified,omitempty"`
	PaymentVerified      bool                   `gorm:"default:false" json:"payment_verified,omitempty"`
	Slug                 string                 `gorm:"uniqueIndex:slugid" json:"slug"`

	Events []Event `gorm:"foreignKey:organizer_id" json:"-"`
	Owner  User    `gorm:"foreignKey:owner_id" json:"-"`

	types.Timestamps
}

type Rating struct {
	ID             uint `gorm:"primaryKey" json:"-"`
	OrganizationID uint `json:"-"`
	ByUser         uint `json:"-"`
	Value          uint `json:"rating_value"`

	User *User `gorm:"foreignKey:by" json:"-"`

	types.Timestamps
}
