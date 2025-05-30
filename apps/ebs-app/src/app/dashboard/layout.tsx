import { AppSidebar } from '@/components/app-sidebar'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { getActiveOrganization, listOrganizations, organizationOnboarding } from '@/lib/actions'
import { notFound, redirect } from 'next/navigation'
import { ReactNode } from 'react'
import { OnboardingNotice } from './components/notice'

export default async function DashboardLayout({ children }: { children: ReactNode }) {
  const orgs = await listOrganizations()
  const org = await getActiveOrganization()
  if (!org) {
    throw notFound()
  }
  if (org.type === 'personal') {
    redirect('/setup/organizations/create')
  }
  const { completed, url } = await organizationOnboarding(org?.id ?? 0)

  return (
    <SidebarProvider>
      <AppSidebar teams={orgs} />
      <SidebarInset>
        {!completed &&
        <div className="flex w-full items-center justify-center">
          <OnboardingNotice url={url} />
        </div>
        }
        { children }
      </SidebarInset>
    </SidebarProvider>
  )
}