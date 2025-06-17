package models

import (
	"context"
	"ebs/src/lib"
	"ebs/src/types"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	"gorm.io/gorm"
)

type User struct {
	ID                   uint            `gorm:"primarykey" json:"id"`
	Name                 string          `json:"name,omitempty"`
	Email                string          `gorm:"uniqueIndex" json:"email,omitempty"`
	Role                 types.UserRole  `json:"role,omitempty"`
	UID                  string          `json:"uid,omitempty"`
	ActiveOrg            uint            `json:"active_org,omitempty"`
	EmailVerified        bool            `json:"email_verified,omitempty"`
	PhoneVerified        bool            `json:"phone_verified,omitempty"`
	VerifiedAt           *time.Time      `json:"verified_at,omitempty"`
	StripeAccountId      *string         `json:"-"`
	StripeCustomerId     *string         `json:"-"`
	StripeSubscriptionId *string         `json:"-"`
	Metadata             *types.Metadata `gorm:"type:jsonb" json:"metadata,omitempty"`
	LastActive           *time.Time      `json:"last_active,omitempty"`
	TenantID             *uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();uniqueIndex" json:"-"`
	Identifier           *string         `gorm:"<-:create" json:"resource_id"`

	Bookings      []*Booking      `gorm:"foreignKey:user_id" json:"bookings,omitempty"`
	Organizations []*Organization `gorm:"foreignKey:owner_id" json:"organizations,omitempty"`
	Subscriptions []*Event        `gorm:"many2many:event_subscriptions;" json:"subscriptions,omitempty"`
	Teams         []*Team         `gorm:"many2many:team_members;" json:"teams,omitempty"`

	types.Timestamps
}

func (u *User) AfterCreate(tx *gorm.DB) error {
	go func() {
		s := lib.GetStripeClient()
		result := s.V1Customers.Search(context.Background(), &stripe.CustomerSearchParams{
			SearchParams: stripe.SearchParams{
				Query: fmt.Sprintf("email:'%s' AND metadata['id']:'%d'", u.Email, u.ID),
			},
		})
		for r, err := range result {
			if err != nil {
				log.Printf("Stripe customer search return an error: %s\n", err.Error())
				break
			}
			if r.Email != u.Email {
				continue
			}
			if err := tx.Transaction(func(tx *gorm.DB) error {
				if err := tx.
					Model(&User{}).
					Where("email = ?", u.Email).
					Updates(&User{}).
					Error; err != nil {
					return err
				}
				return nil
			}); err != nil {
				log.Printf("Error updating user [%s]: %s\n", u.Email, err.Error())
			}
			return
		}
		c, err := s.V1Customers.Create(context.Background(), &stripe.CustomerCreateParams{
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
		priceEnv := os.Getenv("STRIPE_FREE_PRICE")
		scp := &stripe.SubscriptionCreateParams{
			Customer: &c.ID,
			Metadata: map[string]string{
				"id": fmt.Sprint(u.ID),
			},
			Items: []*stripe.SubscriptionCreateItemParams{
				{
					Price: stripe.String(priceEnv),
				},
			},
		}
		if _, err = s.V1Subscriptions.Create(context.Background(), scp); err != nil {
			log.Printf("Error creating subscription: %s\n", err.Error())
			return
		}
	}()
	return nil
}
