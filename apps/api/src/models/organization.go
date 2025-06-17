package models

import (
	"ebs/src/types"
	"fmt"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

type Organization struct {
	ID      uint   `gorm:"primarykey;uniqueIndex:slugid" json:"id"`
	Name    string `json:"name,omitempty"`
	About   string `json:"about,omitempty"`
	Country string `json:"country,omitempty"`
	// OwnerID is the ID of the User that owns this Organization
	OwnerID uint                   `json:"owner_id,omitempty"`
	Type    types.OrganizationType `gorm:"default:'standard'" json:"type,omitempty"`
	// StripeAccountID is the Stripe Connect Account associated with this Organization
	StripeAccountID      *string         `json:"stripe_account_id,omitempty"`
	Metadata             *types.Metadata `gorm:"type:jsonb" json:"metadata,omitempty"`
	ContactEmail         string          `json:"email,omitempty"`
	ConnectOnboardingURL *string         `json:"connect_onboarding_url,omitempty"`
	Status               string          `gorm:"default:'pending'" json:"status,omitempty"`
	Verified             bool            `gorm:"default:false" json:"verified,omitempty"`
	EmailVerified        bool            `gorm:"default:false" json:"email_verified,omitempty"`
	PaymentVerified      bool            `gorm:"default:false" json:"payment_verified,omitempty"`
	Slug                 string          `gorm:"uniqueIndex:slugid" json:"slug"`
	TenantID             *uuid.UUID      `gorm:"type:uuid" json:"-"`
	Identifier           *string         `gorm:"<-:create" json:"resource_id"`

	Events []Event `gorm:"foreignKey:organizer_id" json:"-"`
	Owner  User    `gorm:"foreignKey:owner_id" json:"-"`

	types.Timestamps
}

func (o *Organization) AfterCreate(tx *gorm.DB) error {
	newSlug := slug.Make(fmt.Sprintf("%s-%d", o.Name, o.ID))
	if err := tx.Model(o).Update("slug", newSlug).Error; err != nil {
		return err
	}
	return nil
}

type Rating struct {
	ID             uint `gorm:"primaryKey" json:"-"`
	OrganizationID uint `json:"-"`
	ByUser         uint `json:"-"`
	Value          uint `json:"rating_value"`

	Organization *Organization `json:"rating_target"`
	User         *User         `gorm:"foreignKey:by_user" json:"-"`
	Identifier   *string       `gorm:"<-:create" json:"resource_id"`

	types.Timestamps
}
