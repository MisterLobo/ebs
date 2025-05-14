'use client'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { useCart } from '@/hooks/use-cart'
import { CartItem } from '@/lib/types'
import { MinusIcon, PlusIcon, Trash2 } from 'lucide-react'
import { useCallback, useMemo } from 'react'

type Props = {
  data?: CartItem,
}
export default function CartListItem({ data }: Props) {
  const { addToCart, removeItemFromCart } = useCart()
  const { qty, freeSeats } = useMemo(() => ({
    qty: data?.qty ?? 0,
    freeSeats: data?.stats?.free ?? 0,
  }), [data?.qty, data?.stats?.free])
  const increaseQty = useCallback((v = 1) => addToCart({
    ...data,
    qty: v,
  }), [data])
  const decreaseQty = useCallback((v = 1) => removeItemFromCart(data?.ticket?.id as number, v), [data?.ticket?.id])
  const remove = useCallback(() => removeItemFromCart(data?.ticket?.id as number, data?.qty as number), [data?.ticket?.id, data?.qty])

  return (
    <Card className="flex flex-col w-full my-2 relative">
      <Badge variant="outline" className="absolute top-0 right-0 m-4">standard</Badge>
      <CardHeader className="relative">
        <CardTitle className="flex items-center justify-between">{ data?.ticket?.tier }</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="flex">Price: ${ data?.ticket?.price?.toLocaleString('en-US', { minimumFractionDigits: 2 }) } x { qty } pcs</p>
        <p className="text-xl">Subtotal: <span className="font-semibold">${ data?.subTotal?.toLocaleString('en-US', { minimumFractionDigits: 2 }) }</span></p>
        <Separator className="my-4" />
        <CardAction className="space-x-2">
          <div className="inline-flex gap-4 items-center">
            <Button size="sm" variant="outline" className="rounded-full cursor-pointer disabled:pointer-events-none" onClick={() => decreaseQty(5)} disabled={qty < 1 || qty < 5 || (qty - 5 < 0)}>
              <MinusIcon /> 5
            </Button>
            <Button size="icon" variant="outline" className="rounded-full cursor-pointer disabled:pointer-events-none" onClick={() => decreaseQty()} disabled={qty < 1}>
              <MinusIcon />
            </Button>
            { qty }
            <Button size="icon" variant="outline" className="rounded-full cursor-pointer disabled:pointer-events-none" onClick={() => increaseQty()} disabled={qty >= freeSeats}>
              <PlusIcon />
            </Button>
            <Button size="sm" variant="outline" className="rounded-full cursor-pointer disabled:pointer-events-none" onClick={() => increaseQty(5)} disabled={qty >= freeSeats || qty + 5 > freeSeats}>
              <PlusIcon /> 5
            </Button>
          </div>
          <Button size="icon" variant="destructive" className="rounded-full cursor-pointer" onClick={remove}>
            <Trash2 />
          </Button>
        </CardAction>
      </CardContent>
    </Card>
  )
}