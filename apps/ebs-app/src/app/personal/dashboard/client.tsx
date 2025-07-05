'use client'

import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { format } from 'date-fns/format'
import { useCallback, useEffect, useMemo, useState } from 'react'
import ReservationCardActions from './components/card-actions'
import { Booking, Transaction } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { cancelTransaction, getReservations, resumeCheckoutSession } from '@/lib/actions'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { isUpcoming } from '@/lib/utils'
import { LoaderCircle } from 'lucide-react'
import { toast } from 'sonner'
import { formatDistance } from 'date-fns'

function ReservationCard({ data }: { data: Booking }) {
  return (
    <Card className="w-full max-w-3xl h-auto">
      <CardHeader>
        <h3 className="text-xl">{ data.event?.title }</h3>
        <p>Created { formatDistance(new Date(data?.created_at as string), Date.now(), { addSuffix: true, includeSeconds: true }) }</p>
        <p className="text-xs">{ data.event && formatDistance(new Date(data.event?.date_time as string), Date.now(), { addSuffix: true }) }</p>
        <CardTitle>{ data?.slots_taken } slots taken</CardTitle>
      </CardHeader>
      <CardContent>
        <p>{ data?.currency?.toUpperCase() } { Number(data?.subtotal).toLocaleString('en-US', { minimumFractionDigits: 2 }) }</p>
        {data.status === 'completed' &&
        <CardAction>
          <ReservationCardActions data={data} />
        </CardAction>}
      </CardContent>
    </Card>
  )
}

export function PersonalDashboardClient() {
  const [reservations, setRes] = useState<Booking[]>()
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const [action, setAction] = useState<'cancel' | 'pay' | null>(null)

  const payNow = useCallback(async (txn: Transaction) => {
    setAction('pay')
    setBusy(true)
    const { url, error } = await resumeCheckoutSession(txn.id as string, txn.checkout_session_id as string)
    if (error) {
      toast('ERROR', {
        description: error,
        duration: 30000,
        dismissible: true,
      })
      // setBusy(false)
      return
    }
    if (url) {
      location.href = url
    }
  }, [])

  const cancelTransactionClicked = useCallback(async (txn: Transaction) => {
    setAction('cancel')
    setBusy(true)
    const { ok, error, status } = await cancelTransaction({ id: txn.id as string})
    if (error) {
      toast(`ERROR ${status}`, {
        description: error,
      })
      return
    }
    if (ok) {
      toast('NOTICE', {
        description: 'Cancelation has been requested',
      })
      setBusy(false)
      txn.status = 'canceled'
    }
  }, [])

  const { completed, canceled, transactions } = useMemo(() => {
    if (!reservations) return {
      completed: [],
      canceled: [],
      transactions: null,
    }
    
    const completed = reservations?.filter(r => r.status === 'completed') ?? []
    const pending = reservations?.filter(r => r.status === 'pending') ?? []
    const canceled = reservations?.filter(r => r.status === 'canceled') ?? []
    const pendingTransactions = new Map<string, { t?: Transaction, a: number, i?: Booking[] }>()
    pending.forEach(v => {
      let amt = 0
      if (v.txn) {
        amt = v.txn.currency?.toLowerCase() === 'usd' ? (v.txn.amount ?? 0) / 100.00 : v.txn.amount ?? 0
      }
      if (pendingTransactions.has(v.txn_id as string)) {
        const txn = pendingTransactions.get(v.txn_id as string)
        txn?.i?.push(v)
        return
      }
      pendingTransactions.set(v.txn_id as string, { t: v.txn, a: amt, i: [v]})
    })
    
    return {
      completed,
      canceled,
      transactions: pendingTransactions,
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
      <Tabs defaultValue="completed" className="max-w-3xl w-full">
        <TabsList className="w-fit">
          <TabsTrigger value="completed">Completed</TabsTrigger>
          <TabsTrigger value="pending">Pending</TabsTrigger>
          <TabsTrigger value="Canceled">Canceled</TabsTrigger>
        </TabsList>
        <TabsContent value="completed" className="w-full h-auto">
          <Accordion type="single" collapsible defaultValue="upcoming">
            <AccordionItem value="upcoming">
              <AccordionTrigger>Upcoming</AccordionTrigger>
              <AccordionContent className="space-y-2">
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
              <AccordionContent className="space-y-2">
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
        {transactions && Array.from(transactions?.entries() as any).length > 0 ?
        <div className="space-y-2">
          <Accordion type="single" collapsible>
          {Array.from(transactions?.entries()).map(([k,v]) => (
            <AccordionItem value={k} key={k}>
              <AccordionTrigger>{ format(new Date(v.t?.created_at as string), 'MMM dd') }</AccordionTrigger>
              <AccordionContent className="space-y-2">
                <p>Total amount: { v.t?.currency?.toUpperCase() } { v.a.toLocaleString('en-US') }</p>
                <div className="flex w-full items-center gap-2 my-2">
                  <Button className="cursor-pointer" variant="destructive" disabled={v.t?.status !== 'pending' || busy} onClick={() => cancelTransactionClicked(v.t as Transaction)}>
                    {busy && action === 'cancel' ? <LoaderCircle className="animate-spin" /> : <span>Cancel reservation</span>}
                  </Button>
                  <Button className="cursor-pointer w-24" onClick={() => payNow(v.t as Transaction)} disabled={v.t?.status !== 'pending' || !v.t?.checkout_session_id || busy}>
                    {busy && action === 'pay' ? <LoaderCircle className="animate-spin" /> : <span>Pay now</span>}
                  </Button>
                </div>
                {v?.i?.map((res, index: number) => (
                  <ReservationCard key={index} data={res} />
                ))}
              </AccordionContent>
            </AccordionItem>
          )
          )}
          </Accordion>
        </div> :
        <p className="text-center">No records to show</p>
        }
        </TabsContent>
        <TabsContent value="Canceled" className="w-3xl h-auto space-y-2">
        {canceled.length > 0 ?
        canceled.map((res, index: number) => (
          <ReservationCard key={index} data={res} />
        )) :
        <p className="text-center">No records to show</p>
        }
        </TabsContent>
      </Tabs>
    </div>
    </>
  )
}