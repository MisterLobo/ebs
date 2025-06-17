'use client'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { getEventSubscription, subscribeToEvent } from '@/lib/actions'
import { Event } from '@/lib/types'
import { format } from 'date-fns'
import { BellIcon, Calendar, Info, MapPinIcon, Ticket } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useCallback, useEffect, useState } from 'react'

type Props = {
  data?: Event,
}
export default function EventCard({ data }: Props) {
  const router = useRouter()
  const [busy, setBusy] = useState(false)
  const [subscription, setSubscription] = useState<number>()
  const buyTickets = useCallback(() => {
    router.push(`/${data?.name}/event/${data?.id}/tickets`)
  }, [router, data?.name, data?.id])
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
    <Card className="w-3xl h-auto">
      {(data?.status === 'notify' && data?.opens_at) && <div className="ml-8 text-sm">{ format(new Date(data.opens_at), 'MMM dd p') }</div>}
      <CardContent className="space-x-2 flex flex-row items-center">
        <div className="inline-flex flex-col items-center justify-center h-16 w-24 font-semibold">
          <span className="flex text-xl uppercase">{ format(new Date(data?.date_time as string), 'MMM')}</span>
          <span className="flex text-4xl">{ format(new Date(data?.date_time as string), 'd')}</span>
        </div>
        <Separator orientation="vertical" />
        <div className="flex flex-col gap-2 justify-center w-full ml-2">
          <h2 className="text-md inline-flex text-ellipsis line-clamp-2">{ data?.title }</h2>
          <div className="flex text-sm leading-none">
            <p className="inline-flex items-center gap-2 min-w-32"><Calendar size={16} />{ format(new Date(data?.date_time as string), 'E p') }</p>
            <p className="inline-flex items-center gap-2 my-1 w-full"><MapPinIcon size={16} />{ data?.location }</p>
          </div>
        </div>
        <div className="inline-flex items-center justify-end max-w-48 ms-4">
          {subscription ?
          <Button variant="secondary" disabled className="pointer-events-none w-32"><BellIcon /> Subscribed</Button> :
          <div className="inline-flex flex-col items-center gap-2">
          {data?.status === 'notify' && <Button className="cursor-pointer w-32" onClick={notifyMe} disabled={busy}><BellIcon /> Notify me</Button>}
          {data?.status === 'registration' && <Button className="cursor-pointer w-32" onClick={buyTickets} disabled={busy}><Ticket />Buy Tickets</Button>}
          <Button className="cursor-pointer disabled:pointer-events-none w-32" variant="secondary" disabled><Info />More details</Button>
          </div>
          }
        </div>
      </CardContent>
    </Card>
  )
}