'use client'

import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { useCart } from '@/hooks/use-cart'
import { CartItem, Ticket } from '@/lib/types'
import { IconBasket } from '@tabler/icons-react'
import { MinusIcon, PlusIcon, Trash2 } from 'lucide-react'
import { useCallback, useMemo } from 'react'

type Props = {
  data?: Ticket,
}

export default function TicketCard({ data }: Props) {
  const { addToCart, removeItemFromCart, items } = useCart()
  const soldOut = useMemo(() => data?.stats?.reserved === data?.limit, [data?.stats?.reserved])
  const {
    hasItem,
    item,
  } = useMemo(() => {
    const index = items.findIndex(item => item.ticket?.id === data?.id)
    const hasItem = index > -1
    const item = items[index]
    return {
      item,
      hasItem,
    }
  }, [items])
  const addTicket = useCallback((qty = 1) => {
    if (!data) {
      return
    }
    const sub = (data.price as number) * qty
    addToCart({
      ticket: data,
      qty,
      subTotal: sub,
      stats: data.stats,
    } as CartItem)
  }, [])
  const decreaseQty = useCallback(() => {
    if (!data) {
      return
    }
    removeItemFromCart(data.id, 1)
  }, [])

  return (
    <Card>
      <CardHeader>
        <CardTitle className="space-x-2 uppercase">
          <span>{ data?.tier }</span>
          <span>&middot;</span>
          <span className="">{ data?.currency }</span>
          <span>{ data?.price?.toLocaleString('en-US', { minimumFractionDigits: 2 }) }</span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p>{ data?.type } ticket</p>
        <p><span className="uppercase">{ data?.currency }</span> { data?.price?.toLocaleString('en-US', { minimumFractionDigits: 2 }) }</p>
        {/* <p>Ticket limit: { data?.limited ? data.limit?.toLocaleString('en-US') : 'No limit' }</p> */}
        {/* <p>{ data?.stats?.reserved } reserved</p> */}
        <p>
        { soldOut ?
          <span className="text-red-500">Tickets are sold out</span> :
          <span>
          { (data?.stats?.free ?? 0).toLocaleString('en-US') } slots remaining
          </span>
        }
        </p>
      </CardContent>
      {!soldOut &&
      <>
      <Separator />
      <CardAction className="px-6">
        {hasItem ?
        <div className="inline-flex gap-4 items-center">
          <Button size="icon" variant="outline" className="rounded-full cursor-pointer disabled:pointer-events-none" onClick={decreaseQty} disabled={Number(item?.qty) < 1}>
            <MinusIcon />
          </Button>
          {item?.qty}
          <Button size="icon" variant="outline" className="rounded-full cursor-pointer disabled:pointer-events-none" onClick={() => addTicket()} disabled={item?.qty === data?.stats?.free}>
            <PlusIcon />
          </Button>
          <Button size="icon" variant="destructive" className="rounded-full cursor-pointer" onClick={() => removeItemFromCart(data?.id as number, item.qty as number)}>
            <Trash2 />
          </Button>
        </div> :
        <Button className="cursor-pointer" onClick={() => addTicket()}><IconBasket /> Add to cart</Button>
        }
      </CardAction>
      </>}
    </Card>
  )
}