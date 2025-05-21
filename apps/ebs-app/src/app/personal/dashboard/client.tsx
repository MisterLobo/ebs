'use client'

import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { format } from 'date-fns/format'
import { useCallback, useEffect, useMemo, useState } from 'react'
import ReservationCardActions from './components/card-actions'
import { Booking } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { cancelReservation, getReservations, resumeCheckoutSession } from '@/lib/actions'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { isUpcoming } from '@/lib/utils'
import { LoaderCircle } from 'lucide-react'
import { toast } from 'sonner'
import { useRouter } from 'next/navigation'

function ReservationCard({ data }: { data: Booking }) {
  const router = useRouter()
  const [busy, setBusy] = useState(false)
  const [action, setAction] = useState<'cancel' | 'pay' | null>(null)
  const upcoming = useMemo(() => data && isUpcoming(data.event?.date_time as string), [data])

  const payNow = useCallback(async (res: Booking) => {
    setAction('pay')
    setBusy(true)
    const { url, error } = await resumeCheckoutSession(res.id, res.checkout_session_id as string)
    if (error) {
      toast('ERROR', {
        description: error,
      })
      // setBusy(false)
      return
    }
    if (url) {
      location.href = url
    }
  }, [])

  const cancelReservationClicked = useCallback(async () => {
    setAction('cancel')
    setBusy(true)
    const { ok, error } = await cancelReservation(data.id)
    if (error) {
      toast('ERROR', {
        description: error,
      })
      return
    }
    if (ok) {
      toast('NOTICE', {
        description: 'Cancelation has been requested',
      })
    }
    router.refresh()
  }, [data.id])

  return (
    <Card className="w-3xl h-auto">
      <CardHeader>
      <p>Date: { format(new Date(data?.created_at as string), 'PPP p') }</p>
        <p className="text-xs">{ format(new Date(data.event?.date_time as string), 'PPP p') }</p>
        <CardTitle>{ data?.reserved_tickets?.length } entries</CardTitle>
      </CardHeader>
      <CardContent>
        <p>{ data?.currency?.toUpperCase() } { Number(data?.subtotal).toLocaleString('en-US', { minimumFractionDigits: 2 }) }</p>
        {data.status === 'completed' &&
        <CardAction>
          <ReservationCardActions data={data} />
        </CardAction>}
        {data.status === 'pending' &&
        <CardAction className="flex items-center gap-2">
          {upcoming ?
          <>
          <Button variant="destructive" className="cursor-pointer w-36" onClick={cancelReservationClicked} disabled={!data.checkout_session_id || busy}>
            {busy && action === 'cancel' ?
            <span className="inline-flex items-center gap-2">
              <LoaderCircle className="animate-spin" />
              <span>processing</span>
            </span> : <span>Cancel reservation</span>}
          </Button>
          <Button className="cursor-pointer w-24" onClick={() => payNow(data)} disabled={!data.checkout_session_id || busy}>
            {busy && action === 'pay' ? <LoaderCircle className="animate-spin" /> : <span>Pay now</span>}
          </Button>
          </> :
          <>
          <Button variant="destructive" className="cursor-pointer" disabled>Cancel reservation</Button>
          <Button className="cursor-pointer disabled:pointer-events-none" disabled>Pay now</Button>
          </>
          }
        </CardAction>
        }
      </CardContent>
    </Card>
  )
}

export function PersonalDashboardClient() {
  const [reservations, setRes] = useState<Booking[]>()
  const [error, setError] = useState<string>()

  const { completed, pending, canceled } = useMemo(() => {
    const completed = reservations?.filter(r => r.status === 'completed') ?? []
    const pending = reservations?.filter(r => r.status === 'pending') ?? []
    const canceled = reservations?.filter(r => r.status === 'canceled' || r.status === 'expired') ?? []
    return {
      completed,
      pending,
      canceled,
    }
  }, [reservations])

  const { upcoming, past } = useMemo(() => {
    const upcoming: Booking[] = []
    const past: Booking[] = []
    completed?.forEach(r => {
      if (isUpcoming(r.event?.date_time as string)) {
        upcoming.push(r)
      } else {
        past.push(r)
      }
    })
    return {
      upcoming,
      past,
    }
  }, [completed])

  useEffect(() => {
    getReservations().then(({ data, error }) => {
      if (error) {
        setError(error)
        return
      }
      setRes(data)
    }).catch(console.error)
  }, [])

  return (
    <>
    <h2 className="text-xl">Reservations: { reservations?.length }</h2>
    {error && <h2 className="text-red-500">Error: { error }</h2>}
    <div className="flex flex-col gap-4 items-center justify-center">
      <Tabs defaultValue="completed">
        <TabsList className="w-fit">
          <TabsTrigger value="completed">Completed</TabsTrigger>
          <TabsTrigger value="pending">Pending</TabsTrigger>
          <TabsTrigger value="Canceled">Canceled</TabsTrigger>
        </TabsList>
        <TabsContent value="completed" className="w-3xl h-auto">
          <Accordion type="single" collapsible defaultValue="upcoming">
            <AccordionItem value="upcoming">
              <AccordionTrigger>Upcoming</AccordionTrigger>
              <AccordionContent>
              {upcoming.length > 0 ?
              upcoming.map((res, index) => (
                <ReservationCard key={index} data={res} />
              )) :
              <p className="text-center">No upcoming reservations</p>
              }
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="past">
              <AccordionTrigger>Past</AccordionTrigger>
              <AccordionContent>
              {past.length > 0 ?
              past.map((res, index) => (
                <ReservationCard key={index} data={res} />
              )) :
              <p className="text-center">No past reservations</p>
              }
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </TabsContent>
        <TabsContent value="pending" className="w-3xl h-auto">
        {pending.length > 0 ?
        <div className="space-y-2">
        {pending.map((res, index: number) => (
          <ReservationCard key={index} data={res} />
        ))}
        </div> :
        <p className="text-center">Nothing to show</p>
        }
        </TabsContent>
        <TabsContent value="Canceled" className="w-3xl h-auto">
        {canceled.length > 0 ?
        canceled.map((res, index: number) => (
          <ReservationCard key={index} data={res} />
        )) :
        <p className="text-center">Nothing to show</p>
        }
        </TabsContent>
      </Tabs>
    </div>
    </>
  )
}