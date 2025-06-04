import stripe from '@stripe/stripe-js'

export const stripeClient = stripe.loadStripe(process.env.NEXT_PUBLIC_STRIPE_PUBLIC_KEY as string)