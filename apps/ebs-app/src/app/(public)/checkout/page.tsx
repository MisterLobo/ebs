'use client'

import { Button } from '@/components/ui/button'
import { useCart } from '@/hooks/use-cart'
import { createCheckoutSession } from '@/lib/actions'
import { useCallback } from 'react'

export default function CheckoutPage() {
  const { items } = useCart()
  const placeOrder = useCallback(async () => {
    const lineItems = items.map(item => ({
      qty: item.qty as number,
      ticket: item.ticket?.id as number,
    }))
    const { url, error } = await createCheckoutSession(lineItems)
    if (error) {
      alert(error)
      return
    }
    if (url) {
      location.href = url
    }
  }, [items])
  return (
    <div className="container mx-auto">
      <p>Checkout { items.length } items</p>
      <Button className="cursor-pointer disabled:pointer-events-none" onClick={placeOrder} disabled={items.length < 1}>Place order</Button>
    </div>
  )
}