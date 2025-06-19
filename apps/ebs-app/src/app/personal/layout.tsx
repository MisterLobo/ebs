import FCM from '@/components/fcm'
import { TestEnvironmentAlert } from '@/components/test-env-alert'
import { WebWorker } from '@/components/worker'
import { subscribeToFCMTopics } from '@/lib/actions'
import { isProd } from '@/lib/utils'
import { ReactNode } from 'react'

export default async function PersonalLayout({ children }: { children: ReactNode }) {
  const tokenRetrieved = async (token: string) => {
    'use server'
    await subscribeToFCMTopics(token, 'EventSubscription', 'Events', 'Personal')
  }
  return (
    <>
    {!isProd() && <TestEnvironmentAlert />}
    {children}
    <WebWorker />
    <FCM tokenRetrieved={tokenRetrieved} />
    <WebWorker />
    </>
  )
}