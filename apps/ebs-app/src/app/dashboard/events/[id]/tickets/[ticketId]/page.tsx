import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from '@/components/ui/breadcrumb'
import { Separator } from '@/components/ui/separator'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { getEventById } from '@/lib/actions'
import { importURLPatternPolyfill } from '@/lib/utils'
import { headers } from 'next/headers'
import { notFound } from 'next/navigation'

export default async function TicketPage() {
  await importURLPatternPolyfill()

  const $headers = await headers()
  const url = $headers.get('x-url')
  const urlPattern = new URLPattern({ pathname: '/dashboard/events/:id/tickets/:ticketId' })
  const result = urlPattern.exec(url as string)
  const id = result?.pathname.groups.id ?? '0'
  const eventId = parseInt(id)
  const eventData = await getEventById(eventId)
  if (!eventData) {
    throw notFound()
  }
  const ticketIdParam = result?.pathname.groups.ticketId ?? '0'
  const ticketId = parseInt(ticketIdParam)

  return (
    <>
    <header className="flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-12">
      <div className="flex items-center gap-2 px-4">
        <SidebarTrigger className="-ml-1" />
        <Separator orientation="vertical" className="mr-2 h-4" />
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem className="hidden md:block">
              <BreadcrumbLink href="/dashboard/events">Events</BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator className="hidden md:block" />
            <BreadcrumbItem className="hidden md:block">
              <BreadcrumbLink href={`/dashboard/events/${eventId}`}>
              { eventData.title }
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator className="hidden md:block" />
            <BreadcrumbItem>
              <BreadcrumbPage>{ ticketId }</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>
    </header>
    <div className="flex flex-col w-full m-4">
      <p>Ticket with reservations</p>
    </div>
    </>
  )
}