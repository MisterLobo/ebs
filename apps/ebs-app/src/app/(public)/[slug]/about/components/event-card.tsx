'use client'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { format } from 'date-fns'
import { Event } from '@/lib/types'
import { useRouter } from 'next/navigation'
import { BellIcon, MapPin } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { getEventSubscription, subscribeToEvent } from '@/lib/actions'

type Props = {
  data?: Event,
}
export default function EventCard({ data }: Props) {
  const router = useRouter()
  const [busy, setBusy] = useState(false)
  const [subscription, setSubscription] = useState<number>()
  const findTickets = () => {
    router.push(`/${data?.name}/event/${data?.id}/tickets`)
  }
  const notifyMe = useCallback(async () => {
    if (!data?.id) {
      return
    }
    setBusy(true)
    const { id } = await subscribeToEvent(data.id)
    setSubscription(id)
    router.refresh()
  }, [data?.id])
  useEffect(() => {
    if (!data) {
      return
    }
    getEventSubscription(data.id).then(id => {
      setSubscription(id as number)
    })
  }, [data])

  return (
    <Card className="max-w-5xl">
      <CardHeader>
        <CardTitle className="flex gap-2 items-center"><span>{data?.title}</span><Badge variant="secondary">{data?.status}</Badge></CardTitle>
        <p>{data?.date_time && format(new Date(data?.date_time), 'PPP p')}</p>
        <p className="flex gap-2 items-center"><MapPin /><span className="text-lg">{data?.location}</span></p>
      </CardHeader>
      <CardContent>
        <CardAction>
          <div className="inline-flex flex-col items-center gap-2">
          {subscription ?
          <Button variant="secondary" disabled className="pointer-events-none w-32"><BellIcon /> Subscribed</Button> :
          <div className="inline-flex flex-col items-center gap-2">
          {data?.status === 'notify' && <Button className="cursor-pointer w-32" onClick={notifyMe} disabled={busy}><BellIcon /> Notify me</Button>}
          {data?.status === 'registration' && <Button className="cursor-pointer" onClick={findTickets}>Find Tickets</Button>}
          </div>
          }
          </div>
        </CardAction>
      </CardContent>
    </Card>
  )
}