import { ChartAreaInteractive } from '@/components/chart-area-interactive'
import { DataTable } from '@/components/data-table'
import { SectionCards } from '@/components/section-cards'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb'
import { Separator } from '@/components/ui/separator'
import {
  SidebarTrigger,
} from '@/components/ui/sidebar'

import data from './data.json'
import { ChartBarInteractive } from '@/components/chart-bar-interactive'
import { getActiveOrganization } from '@/lib/actions'
import { notFound, redirect } from 'next/navigation'
import { Suspense } from 'react'
import { Loader2 } from 'lucide-react'

export default async function Page() {
  const org = await getActiveOrganization()
  if (!org) {
    throw notFound()
  }
  if (org.type === 'personal') {
    redirect('/setup/organizations/create')
  }
  return (
    <Suspense fallback={<Loader2 className="animate-spin" />}>
      <>
      <header className="flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-12">
        <div className="flex items-center gap-2 px-4">
          <SidebarTrigger className="-ml-1" />
          <Separator orientation="vertical" className="mr-2 h-4" />
          <Breadcrumb>
            <BreadcrumbList>
              <BreadcrumbItem className="hidden md:block">
                <BreadcrumbLink href="#">
                  Building Your Application
                </BreadcrumbLink>
              </BreadcrumbItem>
              <BreadcrumbSeparator className="hidden md:block" />
              <BreadcrumbItem>
                <BreadcrumbPage>Data Fetching</BreadcrumbPage>
              </BreadcrumbItem>
            </BreadcrumbList>
          </Breadcrumb>
        </div>
      </header>
      <div className="flex flex-1 flex-col">
        <div className="@container/main flex flex-1 flex-col gap-2">
          <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
            <SectionCards />
            <div className="px-4 lg:px-6 space-y-4">
              <ChartAreaInteractive />
              <ChartBarInteractive />
            </div>
            <DataTable data={data} />
          </div>
        </div>
      </div>
      </>
    </Suspense>
  )
}
