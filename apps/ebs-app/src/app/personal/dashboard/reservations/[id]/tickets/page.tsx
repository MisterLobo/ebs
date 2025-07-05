import { getBookingTickets } from '@/lib/actions'
import { Booking } from '@/lib/types'
import { importURLPatternPolyfill } from '@/lib/utils'
import { format } from 'date-fns'
import { headers } from 'next/headers'
import ReservedTicket from './components/reserved-ticket'
import { Suspense } from 'react'
import { Loader2 } from 'lucide-react'

export default async function PersonalTickets() {
  await importURLPatternPolyfill()

  const $headers = await headers()
  const url = $headers.get('x-url')
  const urlPattern = new URLPattern({ pathname: '/personal/dashboard/reservations/:id/tickets' })
  const result = urlPattern.exec(url as string)
  const id = result?.pathname.groups.id ?? '0'
  const reservationId = parseInt(id)
  const { data } = await getBookingTickets(reservationId)
  const booking = data as Booking
  
  return (
    <div className="container flex flex-col w-full gap-2">
      <Suspense fallback={<Loader2 className="animate-spin" />}>
        <h1 className="text-4xl font-semibold">{ booking?.event?.title }</h1>
        <h3 className="text-xl">{ booking?.event?.location }</h3>
        {booking?.event?.date_time && <h3 className="text-xl">{ format(new Date(booking?.event?.date_time as string), 'PPPP p') }</h3>}
        <div className="flex flex-col items-center gap-4 my-10">
          <p>{ booking.reservations?.length } tickets</p>
          {booking?.reservations?.map((t, i) => (
            <ReservedTicket booking={booking} reservation={t} data={t.ticket} key={i} />
          ))}
        </div>
      </Suspense>
    </div>
  )
}