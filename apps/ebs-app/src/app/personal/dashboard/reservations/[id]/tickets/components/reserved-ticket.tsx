'use client'

import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { downloadTicket, getTicketShareLink } from '@/lib/actions'
import { Booking, Reservation, Ticket } from '@/lib/types'
import { isUpcoming } from '@/lib/utils'
import { IconTransfer } from '@tabler/icons-react'
import { format } from 'date-fns'
import { DollarSign, Download, Share2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { toast } from 'sonner'

type Props = {
  data?: Ticket,
  reservation?: Reservation,
  booking?: Booking,
}
export default function ReservedTicket({ data, booking, reservation }: Props) {
  const [busy, setBusy] = useState(false)
  const [url, setUrl] = useState<string>()
  const link = useRef<HTMLAnchorElement>(null)
  const admitted = useMemo(() => isUpcoming(booking?.event?.date_time as string) && booking?.status === 'completed' && reservation?.status === 'completed', [reservation])
  const canUse = useMemo(() => isUpcoming(booking?.event?.date_time as string) && booking?.status === 'completed' && reservation?.status === 'paid' && !admitted, [reservation, admitted])
  const downloadQrCode = useCallback(async () => {
    if (!data || !reservation) {
      return
    }
    setBusy(true)
    const { blob, error, status } = await downloadTicket(data.id, reservation.id as number)
    if (error) {
      toast(`ERROR ${status}`, {
        description: error,
      })
      return
    }
    if (blob) {
      const url = URL.createObjectURL(new Blob([blob], { type: 'image/jpeg' }))
      setUrl(url)
    }
    setBusy(false)
  }, [data?.id, reservation?.id])
  const shareTicket = useCallback(async () => {
    if (!data || !reservation) {
      return
    }
    setBusy(true)
    const url = await getTicketShareLink(data.id as number, reservation.id as number)
    if (url) {
      await navigator.clipboard.writeText(url)
      toast('NOTICE', {
        description: 'Share URL has been copied to clipboard',
      })
    }
    setBusy(false)
  }, [])
  useEffect(() => {
    if (url && link.current) {
      link.current?.click()
    }
  }, [url, link.current])
  return (
    <Card className="w-3xl h-auto">
      <CardHeader>
        <CardTitle>#{ reservation?.id } { data?.tier } { data?.type }</CardTitle>
      </CardHeader>
      <CardContent>
        {canUse ? <p>Valid until: { format(new Date(booking?.event?.date_time as string), 'PPP p') }</p> : (admitted ? <p className="text-orange-500 uppercase">admitted</p> : <p className="text-orange-500 uppercase">expired</p>)
        }
        <p>{ data?.currency?.toUpperCase() } { data?.price?.toLocaleString('en-US', { minimumFractionDigits: 2 }) }</p>
        <CardAction className="space-x-2">
          <Button type="button" className="cursor-pointer disabled:pointer-events-none disabled:cursor-not-allowed" onClick={shareTicket} disabled={admitted || !canUse || busy}><Share2 /> SHARE</Button>
          <Button type="button" className="cursor-pointer disabled:pointer-events-none disabled:cursor-not-allowed" onClick={downloadQrCode} disabled={admitted || !canUse || busy}><Download /> DOWNLOAD QR CODE</Button>
          <Button type="button" variant="secondary" className="cursor-pointer disabled:pointer-events-none disabled:cursor-not-allowed" disabled><IconTransfer /> TRANSFER</Button>
          <Button type="button" variant="destructive" className="cursor-pointer disabled:pointer-events-none disabled:cursor-not-allowed" disabled><DollarSign /> SELL</Button>
        </CardAction>
      </CardContent>
      <a className="hidden" ref={link} download href={url}>Download</a>
    </Card>
  )
}