'use client'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { format } from 'date-fns'
import { Event } from '@/lib/types'
import { useRouter } from 'next/navigation'
import { MapPin } from 'lucide-react'

type Props = {
  data?: Event,
}
export default function EventCard({ data }: Props) {
  const router = useRouter()
  const findTickets = () => {
    router.push(`/${data?.name}/event/${data?.id}/tickets`)
  }

  return (
    <Card className="max-w-5xl">
      <CardHeader>
        <CardTitle className="flex gap-2 items-center"><span>{data?.title}</span><Badge>{data?.status}</Badge></CardTitle>
        <p>{data?.date_time && format(new Date(data?.date_time), 'PPP p')}</p>
        <p className="flex gap-2 items-center"><MapPin /><span className="text-lg">{data?.location}</span></p>
      </CardHeader>
      <CardContent>
        <CardAction>
          <Button className="cursor-pointer" onClick={findTickets}>Find Tickets</Button>
        </CardAction>
      </CardContent>
    </Card>
  )
}