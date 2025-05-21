'use client'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { isUpcoming } from '@/lib/utils'
import { Separator } from '@radix-ui/react-separator'
import { format } from 'date-fns/format'
import { BellIcon, Trash } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useCallback, useMemo } from 'react'

type Props = {
  data?: any,
}
export default function WaitlistItem({ data }: Props) {
  const router = useRouter()
  const upcoming = useMemo(() => isUpcoming(data.event?.date_time), [data])
  const buyTickets = useCallback(() => {
    router.push(`/${data.event?.name}/event/${data.event?.id}/tickets`)
  }, [router, data.event])

  return (
    <Card className="w-3xl h-auto">
      <CardHeader>
        {(data.event?.status === 'notify' && data.event?.opens_at) && <h2 className="text-md flex items-center gap-2"><BellIcon size={16} /> { format(new Date(data.event?.opens_at), 'P p') }</h2>}
        <CardTitle>
          <h2 className="text-3xl inline-flex break-words text-wrap">{ data.event?.title }</h2>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-x-2">
        <h2 className="text-md"><Badge>{ data.event?.status }</Badge></h2>
        <Separator />
        <CardAction>
          {data.event?.status === 'open' && <Button className="cursor-pointer" disabled={!upcoming} onClick={buyTickets}>Buy Tickets</Button>}
          {data.event?.status === 'notify' && <Button variant="destructive" className="cursor-pointer" disabled><Trash /> Remove</Button>}
        </CardAction>
      </CardContent>
    </Card>
  )
}