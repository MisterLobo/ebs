'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Booking } from '@/lib/types'
import { format } from 'date-fns'
import { Loader2 } from 'lucide-react'
import { Suspense, use } from 'react'

type Props = {
  id?: number,
  reservations: Promise<{ data?: Booking, error?: string } | null>
}
export function BookingReservations({ reservations }: Props) {
  const response = use(reservations)
  return (
    <Card className="min-w-5xl">
      <CardHeader>
        <CardTitle>Reservations{response?.data?.reservations?.length ? ` (${response?.data?.reservations?.length})` : ''}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="grid grid-cols-2 gap-4">
          <Suspense fallback={<Loader2 className="animate-spin" />}>
            {response?.data?.reservations?.length ? response?.data?.reservations?.map(d => (
              <Card key={d.id} className="w-full">
                <CardContent>
                  <div className="grid grid-cols-2 gap-2">
                    <div className="flex flex-col">
                      <p className="text-sm">ID</p>
                      <h2 className="text-xl">{d.id}</h2>
                    </div>
                    <div className="flex flex-col">
                      <p className="text-sm">Status</p>
                      <h2 className="text-xl uppercase">{d.status}</h2>
                    </div>
                    <div className="flex flex-col">
                      <p className="text-sm">Booking ID</p>
                      <h2 className="text-xl">{d.booking_id}</h2>
                    </div>
                    <div className="flex flex-col">
                      <p className="text-sm">Created At</p>
                      <h2 className="text-xl">{format(new Date(d.created_at as string), 'Pp')}</h2>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )) : <p>No data available</p>}
          </Suspense>
        </div>
      </CardContent>
    </Card>
  )
}