import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { getReservations } from '@/lib/actions'
import { format } from 'date-fns'
import ReservationCardActions from './components/card-actions'
import { Booking } from '@/lib/types'

export default async function PersonalDashboard() {
  const { data, error } = await getReservations()
  if (error) {
    console.log('[error]:', error);
  }
  const reservations = data as Booking[]
  
  return (
    <main className="p-6">
      <h1 className="text-3xl font-semibold">My Dashboard</h1>
      {reservations?.length > 0 ?
      <>
      <h2 className="text-xl">Reservations: { reservations?.length }</h2>
      <div className="flex flex-col gap-4 items-center justify-center">
      {reservations.map((res, index: number) => (
        <Card key={index} className="w-3xl h-auto">
          <CardHeader>
            <p className="text-xs">{ format(new Date(res.created_at as string), 'PPP p') }</p>
            <CardTitle>{ res.reserved_tickets?.length } entries</CardTitle>
          </CardHeader>
          <CardContent>
            <p>${ Number(res.subtotal).toLocaleString('en-US', { minimumFractionDigits: 2 }) }</p>
            <CardAction>
              <ReservationCardActions data={res} />
            </CardAction>
          </CardContent>
        </Card>
      ))}
      </div>
      </> :
      <p className="text-center">No reservations</p>
      }
    </main>
  )
}