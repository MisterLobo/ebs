import { getActiveOrganization, getEventById, getTickets } from '@/lib/actions'
import { importURLPatternPolyfill } from '@/lib/utils'
import { format } from 'date-fns'
import { headers } from 'next/headers'
import TicketCard from './components/ticket-card'
import CartItems from './components/cart-sidebar'
import Link from 'next/link'
import { Calendar, Clock, MapPin, PencilIcon, Ticket } from 'lucide-react'
import { notFound } from 'next/navigation'

export default async function BuyTicketsPage() {
  await importURLPatternPolyfill()
  const $headers = await headers()
  const url = $headers.get('x-url')
  const pattern = new URLPattern({ pathname: '/:slug/event/:id/tickets' })
  const result = pattern.exec(url as string)
  const id = result?.pathname.groups.id as string
  const eventId = parseInt(id)

  const event = await getEventById(eventId)
  if (!event) {
    throw notFound()
  }
  if (event.status !== 'open') {
    throw notFound()
  }
  const tickets = await getTickets(eventId)
  const org = await getActiveOrganization()
  const canManage = event.organizer_id === org?.id
  return (
    <div className="">
      <div className="relative">
        <div className="flex items-center gap-2 w-full">
          <h1 className="text-5xl text-wrap break-words w-full">
            { event.title }
          </h1>
          {canManage && <Link href={`/dashboard/events/${eventId}`} className="underline"><PencilIcon /></Link>}
        </div>
        {event.deadline && <h2 className="text-xl flex items-center gap-2"><Clock /> { format(new Date(event.deadline), 'MMM do yyyy HH:mm') }</h2>}
        <div className="flex max-w-2xl items-center justify-between my-4">
          <h2 className="inline-flex items-center gap-2 text-3xl"><Calendar size={32} /> { format(new Date(event.date_time as string), 'MMM do HH:mm') }</h2>
          <h2 className="inline-flex items-center gap-2"><MapPin /> { event.location }</h2>
        </div>
        {tickets.length > 0 ?
        <div className="flex flex-col gap-4">
          {tickets.map(ticket => (
            <TicketCard data={ticket} key={ticket.id} />
          ))}
          <h2 className="text-center text-muted italic">End of list</h2>
        </div> :
        <div className="flex flex-col w-full items-center justify-center h-96">
          <Ticket size={96} />
          <p className="text-center italic my-2">No tickets available</p>
        </div>
        }
      </div>
      {tickets.length > 0 && <CartItems />}
    </div>
  )
}