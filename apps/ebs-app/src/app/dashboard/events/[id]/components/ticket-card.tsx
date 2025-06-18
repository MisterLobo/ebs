'use client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import LoadingButton from '@/components/ui/loading-button'
import { archiveTicket, closeTicket, publishTicket } from '@/lib/actions'
import { Ticket } from '@/lib/types'
import { Trash2 } from 'lucide-react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useCallback, useState } from 'react'
import { toast } from 'sonner'

type Props = {
  ticket: Ticket,
}
export default function TicketCard({ ticket }: Props) {
  const router = useRouter()
  const [busy, setBusy] = useState(false)
  const onClickPublish = useCallback(async () => {
    setBusy(true)
    const error = await publishTicket(ticket.id)
    if (error) {
      toast('ERROR', {
        description: error,
      })
      return
    }
    setBusy(false)
    router.refresh()
  }, [ticket])
  const onClickDelete = useCallback(async () => {
    setBusy(true)
    const error = await archiveTicket(ticket.id)
    if (error) {
      toast('ERROR', {
        description: error,
      })
      return
    }
  }, [])
  const onClickClose = useCallback(async () => {
    setBusy(true)
    const error = await closeTicket(ticket.id)
    if (error) {
      toast('ERROR', {
        description: error,
      })
      return
    }
    setBusy(false)
    router.refresh()
  }, [])
  return (
    <Card className="m-4 w-96 relative" key={ticket.id}>
      <CardHeader>
        <CardTitle className="text-2xl">
          <Link href={`/dashboard/events/${ticket.event_id}/tickets/${ticket.id}`}>{ ticket.tier }</Link>
        </CardTitle>
        <Button type="button" size="icon" variant="ghost" className="absolute right-0 me-4 rounded-full hover:text-red-500 cursor-pointer" onClick={onClickDelete} disabled={busy}><Trash2 /></Button>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col">
          <Badge className="mb-4">{ ticket.status }</Badge>
          <p>{ ticket.currency === 'usd' ? '$' : ticket.currency?.toUpperCase() }{ ticket.price?.toLocaleString('en-US', { minimumFractionDigits: 0 }) }</p>
          <p>Type: { ticket.type }</p>
          <p>{ ticket.limited ? `Maximum of ${ticket.limit} reservations` : 'Unlimited reservations' }</p>
          <p>{ ticket.limit } Seats available</p>
        </div>
          {busy ?
          <LoadingButton text="Processing" icon={false} className="inline-flex w-full mt-4 disabled:pointer-events-none" disabled /> :
          <>
          {ticket.status === 'open' && <Button type="button" variant="destructive" className="cursor-pointer inline-flex w-full shrink mt-4 disabled:opacity-50 disabled:pointer-events-none" disabled={busy} onClick={onClickClose}>CLOSE TICKET</Button>}
          {ticket.status === 'draft' && <Button type="button" variant="outline" className="cursor-pointer inline-flex w-full mt-4 disabled:opacity-50 disabled:pointer-events-none shrink" onClick={onClickPublish} disabled={busy}>
            PUBLISH TICKET
          </Button>}
          </>
          }
      </CardContent>
    </Card>
  )
}