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

import { getActiveOrganization, getDailyTransactions, getMonthlyCustomers, getOrgDashboard, getSoldTickets } from '@/lib/actions'
import { notFound } from 'next/navigation'
import { Suspense } from 'react'
import { Loader2 } from 'lucide-react'
import { DailyTicketReservations } from './components/daily-ticket-reservations'
import { TopEventSales } from './components/event-sales'

export default async function Page() {
  const org = await getActiveOrganization()
  if (!org) {
    throw notFound()
  }
  const ticketsSold = await getSoldTickets(org.id)
  const newCustomers = await getMonthlyCustomers(org.id)
  
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
            <SectionCards
              tickets_sold={ticketsSold?.sales?.total_sold ?? 0}
              total_revenue={ticketsSold?.sales?.total_revenue}
              new_customers={newCustomers ?? 0}
            />
            <div className="px-4 lg:px-6 space-y-4">
              {/* <ChartAreaInteractive /> */}
              {/* <ChartBarInteractive /> */}
              <DailyTicketReservations fetcher={getDailyTransactions(org.id)} />
            </div>
            <TopEventSales fetcher={getOrgDashboard(org.id)} />
          </div>
        </div>
      </div>
      </>
    </Suspense>
  )
}
