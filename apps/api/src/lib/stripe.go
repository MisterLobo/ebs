package lib

import (
	"os"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentlink"
)

var stripeClient *stripe.Client

func GetStripeClient() *stripe.Client {
	if stripeClient != nil {
		return stripeClient
	}
	apiKey := os.Getenv("STRIPE_SECRET_KEY")
	sc := stripe.NewClient(apiKey)
	stripeClient = sc

	return sc
}

func NewStripeClient(c *stripe.Client) {
	stripeClient = c
}

func StripeInitialize() {
	apiKey := os.Getenv("STRIPE_SECRET_KEY")
	stripe.Key = apiKey
}

func StripeCreatePaymentLink(priceId string) (string, error) {
	params := stripe.PaymentLinkParams{
		LineItems: []*stripe.PaymentLinkLineItemParams{
			{
				Price:    stripe.String(priceId),
				Quantity: stripe.Int64(1),
			},
		},
	}
	paymentLink, err := paymentlink.New(&params)
	return paymentLink.URL, err
}
