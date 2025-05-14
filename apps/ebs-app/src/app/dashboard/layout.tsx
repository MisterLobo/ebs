import { AppSidebar } from '@/components/app-sidebar'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { listOrganizations } from '@/lib/actions'
import { ReactNode } from 'react'

export default async function DashboardLayout({ children }: { children: ReactNode }) {
  const orgs = await listOrganizations()

  return (
    <SidebarProvider>
      <AppSidebar teams={orgs} />
      <SidebarInset>
        { children }
      </SidebarInset>
    </SidebarProvider>
  )
}