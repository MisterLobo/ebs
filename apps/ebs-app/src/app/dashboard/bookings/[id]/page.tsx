import { Badge } from '@/components/ui/badge'
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from '@/components/ui/breadcrumb'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getBookingById, getBookingReservations } from '@/lib/actions'
import { importURLPatternPolyfill } from '@/lib/utils'
import { format } from 'date-fns'
import { Loader2 } from 'lucide-react'
import { headers } from 'next/headers'
import { notFound } from 'next/navigation'
import { Suspense } from 'react'
import { BookingReservations } from './components/reservations'
import Link from 'next/link'

export default async function BookingIdPage() {
  await importURLPatternPolyfill()

  const $headers = await headers()
  const url = $headers.get('x-url')
  const urlPattern = new URLPattern({ pathname: '/dashboard/bookings/:id' })
  const result = urlPattern.exec(url as string)
  const id = result?.pathname.groups.id ?? '0'
  const bookingId = parseInt(id)
  const bookingData = await getBookingById(bookingId)
  if (!bookingData) {
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
              <BreadcrumbLink href="/dashboard/bookings">
                Bookings
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator className="hidden md:block" />
            <BreadcrumbItem>
              <BreadcrumbPage>{bookingId}</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>
    </header>
    <div className="flex flex-col w-auto m-4 mx-auto gap-4">
      <Card className="min-w-5xl">
        <CardHeader>
          <div className="flex flex-col gap-2 justify-start">
            <p className="text-sm">ID</p>
            <h1 className="text-2xl font-semibold">{bookingData.id}</h1>
          </div>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-2">
            <div className="col-span-2 flex flex-col">
              <p className="text-sm">Transaction ID</p>
              <h2 className="text-xl uppercase">{bookingData.txn_id}</h2>
            </div>
            <div className="col-span-2 flex flex-col">
              <p className="text-sm">Subtotal</p>
              <h2 className="text-xl uppercase">{bookingData.currency?.toUpperCase()} {bookingData.subtotal?.toLocaleString('en-US')}</h2>
            </div>
            <div className="flex flex-col">
              <p className="text-sm">ID</p>
              <h2 className="text-xl">{bookingData.id}</h2>
            </div>
            <div className="flex flex-col">
              <p className="text-sm">Status</p>
              <h2 className="text-xl uppercase"><Badge>{bookingData.status}</Badge></h2>
            </div>
            <div className="flex flex-col">
              <p className="text-sm">Slots wanted</p>
              <h2 className="text-xl uppercase">{bookingData.metadata?.['slots_wanted']}</h2>
            </div>
            <div className="flex flex-col">
              <p className="text-sm">Slots taken</p>
              <h2 className="text-xl uppercase">{bookingData.metadata?.['slots_wanted']}</h2>
            </div>
            <div className="flex flex-col">
              <p className="text-sm">Guest name</p>
              <h2 className="text-xl">{bookingData.user?.name ?? '-'}</h2>
            </div>
            <div className="flex flex-col">
              <p className="text-sm">Guest email</p>
              <h2 className="text-xl">{bookingData.user?.email ?? '-'}</h2>
            </div>
            <div className="flex flex-col">
              <p className="text-sm">Created at</p>
              <h2 className="text-xl">{format(new Date(bookingData.created_at as string), 'Pp')}</h2>
            </div>
            <div className="flex flex-col">
              <p className="text-sm">Last updated</p>
              <h2 className="text-xl">{format(new Date(bookingData.updated_at as string), 'Pp')}</h2>
            </div>
          </div>
        </CardContent>
      </Card>
      <Tabs defaultValue="reservations">
        <TabsList className="w-full p-0 bg-background justify-start border-b rounded-none">
          <TabsTrigger value="reservations" className="cursor-pointer rounded-none bg-background h-full data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-primary">Reservations</TabsTrigger>
          <TabsTrigger value="event" className="cursor-pointer rounded-none bg-background h-full data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-primary">Event</TabsTrigger>
          <TabsTrigger value="ticket" className="cursor-pointer rounded-none bg-background h-full data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-primary">Ticket</TabsTrigger>
          <TabsTrigger value="metrics" className="cursor-pointer rounded-none bg-background h-full data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-primary">Metrics</TabsTrigger>
        </TabsList>
        <TabsContent value="reservations">
          <Suspense fallback={<Loader2 className="animate-spin size-32" />}>
            <BookingReservations reservations={getBookingReservations(bookingData.id)} />
          </Suspense>
        </TabsContent>
        <TabsContent value="event">
          <Card>
            <CardHeader>
              <CardTitle>
                <span>{bookingData.event?.title}</span>
              </CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-2 gap-4">
              <div className="flex flex-col">
                <p className="text-sm">ID</p>
                <Link href={`/dashboard/events/${bookingData.event_id}`} className="text-xl">
                  {bookingData.event_id}
                </Link>
              </div>
              <div className="flex flex-col">
                <p className="text-sm">Public page</p>
                <Link className="text-xl text-blue-500 underline" href={`/${bookingData.event?.name}/event/${bookingData.event_id}/tickets`}>{bookingData.event?.title}</Link>
              </div>
              <div className="flex flex-col">
                <p className="text-sm">Date and Time</p>
                <h2 className="text-xl">{bookingData.event?.date_time ? format(new Date(bookingData.event?.date_time), 'Pp') : '-'}</h2>
              </div>
              <div className="flex flex-col">
                <p className="text-sm">Venue</p>
                <h2 className="text-xl">{bookingData.event?.location}</h2>
              </div>
              <div className="flex flex-col">
                <p className="text-sm">Type</p>
                <h2 className="text-xl uppercase"><Badge variant="secondary">{bookingData.event?.type}</Badge></h2>
              </div>
              <div className="flex flex-col">
                <p className="text-sm">Status</p>
                <h2 className="text-xl uppercase"><Badge>{bookingData.event?.status}</Badge></h2>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="ticket">
          <Card>
            <CardHeader>
              <CardTitle>
                <span>{bookingData.ticket?.tier}</span>
                <span> &middot; </span>
                <Link href={`/dashboard/tickets/${bookingData.ticket_id}`}>
                  {bookingData.ticket_id}
                </Link>
              </CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-2 gap-4">
              <div className="flex flex-col">
                <p className="text-sm">Created At</p>
                <h2 className="text-xl">{bookingData.ticket?.created_at ? format(new Date(bookingData.ticket?.created_at), 'Pp') : '-'}</h2>
              </div>
              <div className="flex flex-col">
                <p className="text-sm">Updated At</p>
                <h2 className="text-xl">{bookingData.ticket?.updated_at ? format(new Date(bookingData.ticket?.updated_at), 'Pp') : '-'}</h2>
              </div>
              <div className="flex flex-col">
                <p className="text-sm">Price</p>
                <h2 className="text-xl">{bookingData.ticket?.price?.toLocaleString('en-US')}</h2>
              </div>
              <div className="flex flex-col">
                <p className="text-sm">Currency</p>
                <h2 className="text-xl">{bookingData.ticket?.currency?.toUpperCase()}</h2>
              </div>
              <div className="flex flex-col">
                <p className="text-sm">Type</p>
                <h2 className="text-xl uppercase"><Badge variant="secondary">{bookingData.ticket?.type}</Badge></h2>
              </div>
              <div className="flex flex-col">
                <p className="text-sm">Status</p>
                <h2 className="text-xl uppercase"><Badge>{bookingData.ticket?.status}</Badge></h2>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="metrics">
          <Card>
            <CardContent>
              <p>COMING SOON</p>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
    </>
  )
}