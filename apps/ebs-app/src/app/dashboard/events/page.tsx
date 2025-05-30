import { getActiveOrganization, getEvents } from '@/lib/actions'
import { DataTable } from './components/data-table'
import { Event } from '@/lib/types'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { Separator } from '@/components/ui/separator'
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList } from '@/components/ui/breadcrumb'
import { redirect } from 'next/navigation'
import { EventsHeaderActions } from './components/actions'
import { format } from 'date-fns'

export default async function EventsPage() {
  const org = await getActiveOrganization()
  if (!org) {
    redirect('/login')
  }
  const eventsData = await getEvents(org.id as number) as Event[]
  const events = eventsData.map(event => ({
    id: event.id,
    name: event.name,
    title: event.title as string,
    location: event.location as string,
    dateTime: format(new Date(event.date_time as string), 'P HH:mm'),
    seats: event.seats as number,
    status: event.status as string,
    about: event.about as string,
    createdBy: event.created_by as number,
    organizer: event.organizer as number,
    type: 'standard',
  }))
  return (
    <>
    <header className="flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-12">
      <div className="flex items-center gap-2 px-4">
        <SidebarTrigger className="-ml-1" />
        <Separator orientation="vertical" className="mr-2 h-4" />
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem className="hidden md:block">
              <BreadcrumbLink href="/dashboard/events">
                Events
              </BreadcrumbLink>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>
      <EventsHeaderActions />
    </header>
    <div className="flex flex-1 flex-col">
      <div className="@container/main flex flex-1 flex-col gap-2">
        <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
          <DataTable data={events} />
        </div>
      </div>
    </div>
    </>
  )
}
