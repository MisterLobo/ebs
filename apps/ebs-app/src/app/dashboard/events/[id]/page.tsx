import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from '@/components/ui/breadcrumb'
import { Separator } from '@/components/ui/separator'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { getActiveOrganization, getOrgEventById, getTickets, organizationOnboarding } from '@/lib/actions'
import { EventPageHeaderActions } from './components/actions'
import { importURLPatternPolyfill } from '@/lib/utils'
import { headers } from 'next/headers'
import { Badge } from '@/components/ui/badge'
import TicketCard from './components/ticket-card'
import { format } from 'date-fns'
import { notFound } from 'next/navigation'
import { OnboardingIncomplete } from '../../components/notice'

export default async function EventPage() {
  await importURLPatternPolyfill()

  const $headers = await headers()
  const url = $headers.get('x-url')
  const urlPattern = new URLPattern({ pathname: '/dashboard/events/:id' })
  const result = urlPattern.exec(url as string)
  const id = result?.pathname.groups.id ?? '0'
  const eventId = parseInt(id)
  
  const org = await getActiveOrganization()
  const eventData = await getOrgEventById(org?.id as number, eventId)
  if (!eventData) {
    throw notFound()
  }
  const { completed } = await organizationOnboarding(org?.id as number)
  const ticketsData = await getTickets(eventId, org?.id)

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
            <BreadcrumbSeparator className="hidden md:block" />
            <BreadcrumbItem>
              <BreadcrumbPage>{ eventData.title }</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>
    </header>
    <div className="flex flex-col w-full mx-4">
      <h1 className="text-3xl font-semibold flex items-center gap-2">{ eventData.title }</h1>
      <h2 className="flex gap-2"><span>ID: { eventId }</span>&middot;<span>{ eventData.name }</span></h2>
      <h2 className="">Status: { eventData.status ? <Badge variant="outline">{eventData.status}</Badge> : '-' }</h2>
      {(eventData.status === 'notify' && eventData.opens_at) && <h2>Event reservation opens on { format(new Date(eventData.opens_at), 'PPPP, p') }</h2>}
      <div className="flex gap-12">
        <h2 className="inline-flex">When: {format(new Date(eventData.date_time as string), 'Pp')}</h2>
        <h2 className="inline-flex">Deadline: {format(new Date(eventData.deadline as string), 'Pp')}</h2>
        <h2 className="inline-flex">Where: {eventData.location}</h2>
      </div>
      <p className="text-sm">Description: { eventData.about || <span>No description</span> }</p>
    </div>
    <EventPageHeaderActions event={eventData} />
    {completed ?
    <>
    {ticketsData.length > 0 ?
    <div className="flex items-center justify-center w-full">
      <div className="min-w-96 w-full flex flex-row flex-wrap items-center grow">
        {ticketsData.map(ticket => (
          <TicketCard ticket={ticket} key={ticket.id} />
        ))}
      </div>
    </div> :
    <p className="text-center italic text-gray-400">This event does not have any tickets. Create one to display here</p>
    }
    </> :
    <OnboardingIncomplete />
    }
    </>
  )
}