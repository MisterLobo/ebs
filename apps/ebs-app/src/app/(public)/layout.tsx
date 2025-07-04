import { CartProvider } from '@/components/cart-provider'
import FCM from '@/components/fcm'
import Nav from '@/components/nav/nav'
import { TestEnvironmentAlert } from '@/components/test-env-alert'
import { WebWorker } from '@/components/worker'
import { tokenRetrieved } from '@/lib/actions'
import { isProd } from '@/lib/utils'
import { ReactNode } from 'react'

export default async function PublicLayout({ children }: { children: ReactNode }) {
  return (
    <>
    {!isProd() && <TestEnvironmentAlert />}
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