'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Booking } from '@/lib/types'
import { formatDistance } from 'date-fns'
import { Loader2 } from 'lucide-react'
import { Suspense, use } from 'react'

type Props = {
  bookings: Promise<{ data?: Booking[], error?: string } | null>
}
export function TicketBookings({ bookings }: Props) {
  const response = use(bookings)
  return (
    <Card className="min-w-5xl">
      <CardHeader>
        <CardTitle>Bookings{response?.data?.length ? ` (${response?.data?.length})` : ''}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="grid grid-cols-2 gap-4">
          <Suspense fallback={<Loader2 className="animate-spin" />}>
            {response?.data?.length ? response?.data?.map(d => (
              <Card key={d.id} className="w-full">
                <CardContent>
                  <div className="grid grid-cols-2 gap-2">
                    <div className="col-span-2 flex flex-col">
                      <p className="text-sm">Transaction ID</p>
                      <h2 className="text-xl uppercase">{d.txn_id}</h2>
                    </div>
                    <div className="col-span-2 flex flex-col">
                      <p className="text-sm">Subtotal</p>
                      <h2 className="text-xl uppercase">{d.currency?.toUpperCase()} {d.subtotal?.toLocaleString('en-US')}</h2>
                    </div>
                    <div className="flex flex-col">
                      <p className="text-sm">ID</p>
                      <h2 className="text-xl">{d.id}</h2>
                    </div>
                    <div className="flex flex-col">
                      <p className="text-sm">Status</p>
                      <h2 className="text-xl uppercase">{d.status}</h2>
                    </div>
                    <div className="flex flex-col">
                      <p className="text-sm">Slots wanted</p>
                      <h2 className="text-xl uppercase">{d.metadata?.['slots_wanted']}</h2>
                    </div>
                    <div className="flex flex-col">
                      <p className="text-sm">Slots taken</p>
                      <h2 className="text-xl uppercase">{d.metadata?.['slots_taken']}</h2>
                    </div>
                    <div className="flex flex-col">
                      <p className="text-sm">Guest name</p>
                      <h2 className="text-xl">{d.user?.name ?? '-'}</h2>
                    </div>
                    <div className="flex flex-col">
                      <p className="text-sm">Guest email</p>
                      <h2 className="text-xl">{d.user?.email ?? '-'}</h2>
                    </div>
                    <div className="flex flex-col">
                      <p className="text-sm">Created at</p>
                      <h2 className="text-xl">{d.created_at ? formatDistance(new Date(d.created_at as string), Date.now(), { addSuffix: true }) : '-'}</h2>
                    </div>
                    <div className="flex flex-col">
                      <p className="text-sm">Last updated</p>
                      <h2 className="text-xl">{d.updated_at ? formatDistance(new Date(d.updated_at as string), Date.now(), { addSuffix: true }) : '-'}</h2>
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