import { CartProvider } from '@/components/cart-provider'
import FCM from '@/components/fcm'
import Nav from '@/components/nav/nav'
import { WebWorker } from '@/components/worker'
import { subscribeToFCMTopics } from '@/lib/actions'
import { ReactNode } from 'react'

export default async function PublicLayout({ children }: { children: ReactNode }) {
  const tokenRetrieved = async (token: string) => {
    'use server'
    await subscribeToFCMTopics(token, 'EventSubscription', 'Events', 'Personal')
  }
  return (
    <>
    <CartProvider>
      <div className="min-h-screen container">
        <Nav />
        <main className="m-6">
          {children}
        </main>
      </div>
    </CartProvider>
    <FCM tokenRetrieved={tokenRetrieved} />
    <WebWorker />
    </>
  )
}