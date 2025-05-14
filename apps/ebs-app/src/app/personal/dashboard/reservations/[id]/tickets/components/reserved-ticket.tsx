'use client'

import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { downloadTicket } from '@/lib/actions'
import { Reservation, Ticket } from '@/lib/types'
import { IconTransfer } from '@tabler/icons-react'
import { DollarSign, Download, Share2 } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'

type Props = {
  data?: Ticket,
  reservation?: Reservation,
}
export default function ReservedTicket({ data, reservation }: Props) {
  const [busy, setBusy] = useState(false)
  const [url, setUrl] = useState<string>()
  const link = useRef<HTMLAnchorElement>(null)
  const downloadQrCode = useCallback(async () => {
    if (!data) {
      return
    }
    setBusy(true)
    const blob = await downloadTicket(data.id, reservation?.id as number)
    if (blob) {
      const url = URL.createObjectURL(new Blob([blob], { type: 'image/jpeg' }))
      setUrl(url)
    }
    setBusy(false)
  }, [data?.id, reservation?.id])
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
        <p>{ data?.currency?.toUpperCase() } { data?.price?.toLocaleString('en-US', { minimumFractionDigits: 2 }) }</p>
        <CardAction className="space-x-2">
          <Button type="button" className="cursor-pointer disabled:pointer-events-none"><Share2 /> SHARE</Button>
          <Button type="button" className="cursor-pointer disabled:pointer-events-none" onClick={downloadQrCode} disabled={busy}><Download /> DOWNLOAD QR CODE</Button>
          <Button type="button" variant="secondary" className="cursor-pointer disabled:pointer-events-none" disabled><IconTransfer /> TRANSFER</Button>
          <Button type="button" variant="destructive" className="cursor-pointer disabled:pointer-events-none" disabled><DollarSign /> SELL</Button>
        </CardAction>
      </CardContent>
      <a className="hidden" ref={link} download href={url}>Download</a>
    </Card>
  )
}