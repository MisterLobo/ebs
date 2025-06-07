package lib

import (
	"context"
	"os"

	"github.com/stripe/stripe-go/v82"
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

func CreatePaymentLink(priceId string) (string, error) {
	sc := GetStripeClient()
	params := stripe.PaymentLinkCreateParams{
		LineItems: []*stripe.PaymentLinkCreateLineItemParams{
			{
				Price:    stripe.String(priceId),
				Quantity: stripe.Int64(1),
			},
		},
	}
	paymentLink, err := sc.V1PaymentLinks.Create(context.Background(), &params)
	return paymentLink.URL, err
}
