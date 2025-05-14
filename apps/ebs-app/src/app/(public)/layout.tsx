import { CartProvider } from '@/components/cart-provider'
import Nav from '@/components/nav/nav'
import { ReactNode } from 'react'

export default async function PublicLayout({ children }: { children: ReactNode }) {
  return (
    <CartProvider>
      <div className="min-h-screen container">
        <Nav />
        <main className="m-6">
          {children}
        </main>
      </div>
    </CartProvider>
  )
}