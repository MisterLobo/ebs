import { importURLPatternPolyfill } from '@/lib/utils'
import NewTicketForm from './components/forms'
import { headers } from 'next/headers'
import { getEventById } from '@/lib/actions'
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from '@/components/ui/breadcrumb'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { Separator } from '@/components/ui/separator'
import { notFound } from 'next/navigation'

export default async function NewTicketPage() {
  await importURLPatternPolyfill()

  const $headers = await headers()
  const url = $headers.get('x-url')
  const urlPattern = new URLPattern({ pathname: '/dashboard/events/:id/tickets/new' })
  const result = urlPattern.exec(url as string)
  const id = result?.pathname.groups.id ?? '0'
  const eventId = parseInt(id)
  const eventData = await getEventById(eventId)
  if (!eventData) {
    throw notFound()
  }
  
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
            <BreadcrumbItem className="hidden md:block">
              <BreadcrumbLink href={`/dashboard/events/${eventData?.id}`}>
              { eventData?.title }
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator className="hidden md:block" />
            <BreadcrumbItem>
              <BreadcrumbPage>New Ticket</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>
    </header>
    <div className="mx-auto py-10 space-y-2">
      <h1 className="text-3xl font-semibold">New Ticket</h1>
      <NewTicketForm data={eventData} />
    </div>
    </>
  )
}