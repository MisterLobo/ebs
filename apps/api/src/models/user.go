package models

import (
	"ebs/src/types"
	"time"
)

type User struct {
	ID              uint            `gorm:"primarykey" json:"id"`
	Name            string          `json:"name,omitempty"`
	Email           string          `json:"email,omitempty"`
	Role            string          `json:"role,omitempty"`
	UID             string          `json:"uid,omitempty"`
	ActiveOrg       uint            `json:"active_org,omitempty"`
	EmailVerified   bool            `json:"email_verified,omitempty"`
	PhoneVerified   bool            `json:"phone_verified,omitempty"`
	VerifiedAt      time.Time       `json:"verified_at,omitempty"`
	StripeAccountId string          `json:"-"`
	Metadata        *types.Metadata `gorm:"type:jsonb"`

	Bookings      []Booking            `gorm:"foreignKey:user_id" json:"bookings,omitempty"`
	Organizations []Organization       `gorm:"foreignKey:owner_id" json:"organizations,omitempty"`
	Subscriptions []*EventSubscription `gorm:"many2many:event_subscriptions;" json:"subscriptions,omitempty"`

	types.Timestamps
}
