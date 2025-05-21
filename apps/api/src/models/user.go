package models

import (
	"context"
	"ebs/src/lib"
	"ebs/src/types"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v82"
	"gorm.io/gorm"
)

type User struct {
	ID               uint            `gorm:"primarykey" json:"id"`
	Name             string          `json:"name,omitempty"`
	Email            string          `json:"email,omitempty"`
	Role             string          `json:"role,omitempty"`
	UID              string          `json:"uid,omitempty"`
	ActiveOrg        uint            `json:"active_org,omitempty"`
	EmailVerified    bool            `json:"email_verified,omitempty"`
	PhoneVerified    bool            `json:"phone_verified,omitempty"`
	VerifiedAt       time.Time       `json:"verified_at,omitempty"`
	StripeAccountId  *string         `json:"-"`
	StripeCustomerId *string         `json:"-"`
	Metadata         *types.Metadata `gorm:"type:jsonb"`

	Bookings      []Booking            `gorm:"foreignKey:user_id" json:"bookings,omitempty"`
	Organizations []Organization       `gorm:"foreignKey:owner_id" json:"organizations,omitempty"`
	Subscriptions []*EventSubscription `gorm:"many2many:event_subscriptions;" json:"subscriptions,omitempty"`

	types.Timestamps
}

func (u *User) AfterCreate(tx *gorm.DB) error {
	s := lib.GetStripeClient()
	seq := s.V1Customers.Search(context.Background(), &stripe.CustomerSearchParams{
		SearchParams: stripe.SearchParams{
			Context: context.Background(),
			Query:   fmt.Sprintf("email:\"%s\"", u.Email),
		},
	})
	if seq != nil {

	}
	return nil
}
