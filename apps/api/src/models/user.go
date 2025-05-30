package models

import (
	"context"
	"ebs/src/lib"
	"ebs/src/types"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/stripe/stripe-go/v82"
	"gorm.io/gorm"
)

type User struct {
	ID               uint            `gorm:"primarykey" json:"id"`
	Name             string          `json:"name,omitempty"`
	Email            string          `json:"email,omitempty"`
	Role             types.UserRole  `json:"role,omitempty"`
	UID              string          `json:"uid,omitempty"`
	ActiveOrg        uint            `json:"active_org,omitempty"`
	EmailVerified    bool            `json:"email_verified,omitempty"`
	PhoneVerified    bool            `json:"phone_verified,omitempty"`
	VerifiedAt       *time.Time      `json:"verified_at,omitempty"`
	StripeAccountId  *string         `json:"-"`
	StripeCustomerId *string         `json:"-"`
	Metadata         *types.Metadata `gorm:"type:jsonb" json:"metadata,omitempty"`
	LastActive       *time.Time      `json:"last_active,omitempty"`

	Bookings      []Booking            `gorm:"foreignKey:user_id" json:"bookings,omitempty"`
	Organizations []Organization       `gorm:"foreignKey:owner_id" json:"organizations,omitempty"`
	Subscriptions []*EventSubscription `gorm:"many2many:event_subscriptions;" json:"subscriptions,omitempty"`
	Teams         []Team               `gorm:"many2many:team_members;" json:"teams,omitempty"`

	types.Timestamps
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	var user User
	if err := tx.Where("email = ?", u.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
	}
	return fmt.Errorf("User with email %s already exists", u.Email)
}

func (u *User) AfterCreate(tx *gorm.DB) error {
	go func() {
		s := lib.GetStripeClient()
		_, err := s.V1Customers.Create(context.Background(), &stripe.CustomerCreateParams{
			Email: stripe.String(u.Email),
			Name:  stripe.String(u.Name),
			Metadata: map[string]string{
				"id": fmt.Sprint(u.ID),
			},
		})
		if err != nil {
			log.Printf("Error creating Customer account: %s\n", err.Error())
			return
		}
	}()
	return nil
}
