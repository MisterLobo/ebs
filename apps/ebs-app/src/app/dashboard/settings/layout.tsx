import { AppSidebar } from '@/components/app-sidebar'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { getActiveOrganization, listOrganizations } from '@/lib/actions'
import { notFound } from 'next/navigation'
import { ReactNode } from 'react'

export default async function DashboardLayout({ children }: { children: ReactNode }) {
  const orgs = await listOrganizations()
  const org = await getActiveOrganization()
  if (!org) {
    throw notFound()
  }

  return (
    <SidebarProvider>
      <AppSidebar teams={orgs} />
      <SidebarInset>
        { children }
      </SidebarInset>
    </SidebarProvider>
  )
}