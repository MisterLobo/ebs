import { AppSidebar } from '@/components/app-sidebar'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { getActiveOrganization, listOrganizations, organizationOnboarding, subscribeToFCMTopics } from '@/lib/actions'
import { notFound } from 'next/navigation'
import { ReactNode } from 'react'
import { OnboardingNotice } from './components/notice'
import { WebWorker } from '@/components/worker'
import FCM from '@/components/fcm'
import { TestEnvironmentAlert } from '@/components/test-env-alert'
import { isProd } from '@/lib/utils'

export default async function DashboardLayout({ children }: { children: ReactNode }) {
  const orgs = await listOrganizations()
  const org = await getActiveOrganization()
  if (!org) {
    throw notFound()
  }
  const { completed, account_id, url } = await organizationOnboarding(org?.id ?? 0)

  const tokenRetrieved = async (token: string) => {
    'use server'
    await subscribeToFCMTopics(token, 'EventSubscription', 'Events', 'Personal')
  }
  
  return (
    <>
    <SidebarProvider>
      <AppSidebar teams={orgs} />
      <SidebarInset>
        {!isProd() && <TestEnvironmentAlert />}
        {(!completed && !account_id) &&
        <div className="flex w-full items-center justify-center">
          <OnboardingNotice url={url} />
        </div>
        }
        { children }
      </SidebarInset>
    </SidebarProvider>
    <WebWorker />
    <FCM tokenRetrieved={tokenRetrieved} />
    </>
  )
}