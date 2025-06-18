import FCM from '@/components/fcm'
import { WebWorker } from '@/components/worker'
import { subscribeToFCMTopics } from '@/lib/actions'
import { ReactNode } from 'react'

export default async function PersonalLayout({ children }: { children: ReactNode }) {
  const tokenRetrieved = async (token: string) => {
    'use server'
    await subscribeToFCMTopics(token, 'EventSubscription', 'Events', 'Personal')
  }
  return (
    <>
    {children}
    <WebWorker />
    <FCM tokenRetrieved={tokenRetrieved} />
    <WebWorker />
    </>
  )
}