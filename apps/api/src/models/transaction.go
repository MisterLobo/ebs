package models

import (
	"ebs/src/types"

	"github.com/google/uuid"
)

type Transaction struct {
	ID uuid.UUID `gorm:"primarykey;type:uuid;default:gen_random_uuid()" json:"id"`

	Currency          string                  `json:"currency,omitempty"`
	Amount            float64                 `json:"amount,omitempty"`
	SourceName        string                  `json:"source_name,omitempty"`
	SourceValue       string                  `json:"source_value,omitempty"`
	ReferenceID       string                  `json:"reference_id,omitempty"`
	Status            types.TransactionStatus `gorm:"pending" json:"status,omitempty"`
	Metadata          *types.Metadata         `gorm:"type:jsonb" json:"metadata,omitempty"`
	CheckoutSessionId *string                 `json:"checkout_session_id,omitempty"`
	PaymentIntentId   *string                 `json:"payment_intent_id,omitempty"`
	CouponCode        *string                 `json:"coupon_code,omitempty"`
	TenantID          *uuid.UUID              `gorm:"type:uuid" json:"-"`

	types.Timestamps
}
