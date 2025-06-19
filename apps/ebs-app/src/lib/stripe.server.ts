import { Stripe } from 'stripe'

let stripe: Stripe

export default function getStripeApiClient() {
  if (!stripe) {
    stripe = new Stripe(process.env.STRIPE_SECRET_KEY as string)
  }
  return stripe
}