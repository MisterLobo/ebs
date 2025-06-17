'use client'

import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { useCart } from '@/hooks/use-cart'
import { useCallback, useState } from 'react'
import CartListItem from './cart-list-item'
import { TrashIcon } from 'lucide-react'
import { createCheckoutSession } from '@/lib/actions'
import { IconBasket } from '@tabler/icons-react'
import { toast } from 'sonner'

export default function CartItems() {
  const { total, items, clearCart } = useCart()
  const [busy, setBusy] = useState(false)
  const checkout = useCallback(async () => {
    setBusy(true)
    const lineItems = items.map(item => ({
      qty: item.qty as number,
      ticket: item.ticket?.id as number,
    }))
    const { url, error, status } = await createCheckoutSession(lineItems)
    if (error) {
      console.error(`ERROR ${status}:`, error)
      toast(`ERROR ${status}`, {
        description: error,
      })
      setBusy(false)
      return
    }
    if (!url) {
      alert('Could not proceed to checkout. Reason: URL missing')
      return
    }
    location.href = url
  }, [items])

  return (
    <div className="flex flex-col min-w-96 border rounded h-screen fixed right-0 top-0 p-4 z-10 overflow-y-scroll bg-background">
      <h3 className="text-lg font-semibold flex gap-2"><IconBasket /> My Cart</h3>
      <Separator className="my-4" />
      {items.length > 0 && <div className="flex items-center justify-between">
        <span className="text-xl">{items.length} item(s)</span>
        <Button size="icon" className="rounded-full cursor-pointer" variant="destructive" onClick={clearCart}><TrashIcon /></Button>
      </div>}
      <div className="flex flex-col h-full">
      {items.length > 0 ?
      <div className="flex flex-col gap-2">
        {items.map((item, index) => (
          <CartListItem data={item} key={index} />
        ))}
      </div> :
      <p>Cart is empty</p>
      }
      </div>
      <Separator className="my-4" />
      <h4 className="text-3xl">Total: ${ total.toLocaleString('en-US', { minimumFractionDigits: 2 }) }</h4>
      <Separator className="my-4" />
      <Button className="cursor-pointer w-full" onClick={checkout} disabled={total === 0 || items.length === 0 || busy}>{ busy ? 'Processing' : 'Proceed to Checkout' }</Button>
    </div>
  )
}