'use client'

import { CartItem } from '@/lib/types'
import { createContext, ReactNode, useCallback, useMemo, useState } from 'react'

export type CartContextProps = {
  total: number,
  items: CartItem[],
  addToCart: (item: CartItem) => void,
  removeItemFromCart: (id: number, qty: number) => void,
  clearCart: () => void,
}

export const CartContext = createContext<CartContextProps | null>(null)
export const CartProvider = ({ children }: { children: ReactNode }) => {
  const [items, setItems] = useState<CartItem[]>([])
  const total = useMemo(() => {
    let total = 0
    items.forEach(item => {
      total += (Number(item.qty) * Number(item.ticket?.price))
    })
    return total
  }, [items])
  const clearCart = useCallback(() => {
    setItems([])
  }, [])
  const addToCart = useCallback((newItem: CartItem) => {
    setItems(old => {
      const arr = Array.from(old)
      const index = arr.findIndex(value => value.ticket?.id === newItem.ticket?.id)

      if (index === -1) {
        arr.push(newItem)
      } else {
        const item = arr[index]
        const free = item.stats?.free ?? 0
        const qty = item.qty ?? 0
        const newQty = newItem.qty ?? 0
        if (item.ticket?.limited) {
          item.qty = Math.min(qty + newQty, free)
        } else {
          item.qty = qty + newQty
        }
        const ticketPrice = item.ticket?.price ?? 0
        const maxSubtotal = free * ticketPrice
        const newSubtotal = Math.min(newQty, free) * (newItem.ticket?.price ?? 0)
        const itemSubtotal = item.subTotal ?? 0
        const subtotal = itemSubtotal >= maxSubtotal ? 0 : newSubtotal
        item.subTotal = (item.subTotal ?? 0) + subtotal
        arr[index] = item
      }
      return arr
    })
  }, [])
  const removeItemFromCart = useCallback((id: number, newQty: number) => {
    setItems(old => {
      const arr = Array.from(old)
      const index = arr.findIndex(value => value.ticket?.id === id)
      if (index > -1) {
        const item = arr[index]
        if (item.qty === 0 || item.qty === newQty) {
          arr.splice(index, 1)
        } else {
          const qty = item.qty ?? 0
          item.qty = qty - Math.min(qty, newQty)
          
          const newSub = newQty * (item.ticket?.price ?? 0)
          const subtotal = (item.subTotal ?? 0) - newSub
          item.subTotal = Math.max(subtotal, 0)
        }
      }
      return arr
    })
  }, [])

  return (
    <CartContext.Provider value={{
      items,
      total,
      addToCart,
      removeItemFromCart,
      clearCart,
    }}>
      { children }
    </CartContext.Provider>
  )
}