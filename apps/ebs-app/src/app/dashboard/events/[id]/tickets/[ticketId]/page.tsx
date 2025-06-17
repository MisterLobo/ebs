import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from '@/components/ui/breadcrumb'
import { Separator } from '@/components/ui/separator'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { getActiveOrganization, getOrgEventById, getTicket, getTicketBookings, getTicketReservations } from '@/lib/actions'
import { cn, importURLPatternPolyfill } from '@/lib/utils'
import { headers } from 'next/headers'
import { notFound } from 'next/navigation'
import { Suspense } from 'react'
import { TicketReservations } from './components/reservations'
import { Loader2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatDistance } from 'date-fns'
import { TicketBookings } from './components/bookings'

export default async function TicketPage() {
  await importURLPatternPolyfill()

  const $headers = await headers()
  const url = $headers.get('x-url')
  const urlPattern = new URLPattern({ pathname: '/dashboard/events/:id/tickets/:ticketId' })
  const result = urlPattern.exec(url as string)
  const id = result?.pathname.groups.id ?? '0'
  const eventId = parseInt(id)
  const org = await getActiveOrganization()
  const eventData = await getOrgEventById(org?.id as number, eventId)
  if (!eventData) {
    throw notFound()
  }
  const ticketIdParam = result?.pathname.groups.ticketId ?? '0'
  const ticketId = parseInt(ticketIdParam)
  const ticket = await getTicket(ticketId)

  return (
    <>
    <header className="flex h-16 shrink items-center gap-2 transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-12">
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
              <BreadcrumbPage>{ ticket?.tier }</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>
    </header>
    <div className="flex flex-col w-auto m-4 mx-auto gap-4">
      <Card className="min-w-7xl">
        <CardHeader>
          <CardTitle className="flex gap-2 items-center">
            <h1 className="text-3xl">{ticket?.tier}</h1>
            <Badge>{ticket?.type}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-4">
          <div className="flex flex-col">
            <p className="text-sm">Price</p>
            <p className="text-xl">{ticket?.currency?.toUpperCase()} {ticket?.price?.toLocaleString('en-US')}</p>
          </div>
          <div className="flex flex-col">
            <p className="text-sm">Slots</p>
            <p className="text-xl">{ticket?.limited ? ticket?.limit : 'unlimited'}</p>
          </div>
          <div className="flex flex-col">
            <p className="text-sm">Free slots</p>
            <p className="text-xl">{ticket?.stats?.free}</p>
          </div>
          <div className="flex flex-col">
            <p className="text-sm">Reserved slots</p>
            <p className="text-xl">{ticket?.stats?.reserved ?? 0}</p>
          </div>
        </CardContent>
      </Card>
      <Tabs defaultValue="event">
        <TabsList className="w-full p-0 bg-background justify-start border-b rounded-none">
          <TabsTrigger value="event" className="cursor-pointer rounded-none bg-background h-full data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-primary">Event</TabsTrigger>
          <TabsTrigger value="bookings" className="cursor-pointer rounded-none bg-background h-full data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-primary">Bookings</TabsTrigger>
          <TabsTrigger value="reservations" className="cursor-pointer rounded-none bg-background h-full data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-primary">Reservations</TabsTrigger>
          <TabsTrigger value="metrics" className="cursor-pointer rounded-none bg-background h-full data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-primary">Metrics</TabsTrigger>
        </TabsList>
        <TabsContent value="event">
          <Card className="min-w-5xl">
            <CardHeader>
              <CardTitle>Event Details</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="col-span-2 flex flex-col">
                  <p className="text-sm">Title</p>
                  <p className="text-xl">{ticket?.event?.title}</p>
                </div>
                <div className="inline-flex flex-col">
                  <p className="text-sm">Date and Time</p>
                  <p className="text-xl">{ticket?.event?.date_time ? formatDistance(new Date(ticket?.event?.date_time as string), Date.now(), { addSuffix: true, includeSeconds: true }) : '-'}</p>
                </div>
                <div className="flex flex-col">
                  <p className="text-sm">Location</p>
                  <p className="text-xl">{ticket?.event?.location}</p>
                </div>
                <div className="inline-flex flex-col">
                  <p className="text-sm">Deadline</p>
                  <p className="text-xl">{ticket?.event?.deadline ? formatDistance(new Date(ticket?.event?.deadline as string), Date.now(), { addSuffix: true, includeSeconds: true }) : '-'}</p>
                </div>
                <div className="inline-flex flex-col">
                  <p className="text-sm">Opening date</p>
                  <p className="text-xl">{ticket?.event?.opens_at ? formatDistance(new Date(ticket?.event?.opens_at as string), Date.now(), { addSuffix: true, includeSeconds: true }) : '-'}</p>
                </div>
              </div>
              <div className="flex flex-col">
                <p className="text-sm">Status</p>
                <p className={cn('text-xl', ticket?.event?.status === 'expired' && 'text-red-500')}>{ticket?.event?.status}</p>
              </div>
              <div className="flex flex-col">
                <p className="text-sm">Name</p>
                <p className="text-xl">{ticket?.event?.name}</p>
              </div>
              <div className="grid grid-flow-col">
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="bookings">
          <TicketBookings bookings={getTicketBookings(ticketId)} />
        </TabsContent>
        <TabsContent value="reservations">
          <TicketReservations id={ticketId} reservations={getTicketReservations(ticketId)} />
        </TabsContent>
        <TabsContent value="metrics">
          <Suspense fallback={<Loader2 className="animate-spin" />}>
            <Card className="min-w-5xl">
              <CardHeader>
                <CardTitle>Metrics</CardTitle>
              </CardHeader>
              <CardContent>
                <p>Chart</p>
              </CardContent>
            </Card>
          </Suspense>
        </TabsContent>
      </Tabs>
    </div>
    </>
  )
}