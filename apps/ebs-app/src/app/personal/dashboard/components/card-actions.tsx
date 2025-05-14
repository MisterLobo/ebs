'use client'

import { Button } from '@/components/ui/button'
import { Reservation } from '@/lib/types'
import { useRouter } from 'next/navigation'

type Props = {
  data?: Reservation,
}
export default function ReservationCardActions({ data }: Props) {
  const router = useRouter()
  const viewTicketsClicked = () => {
    if (!data) {
      return
    }
    router.push(`/personal/dashboard/reservations/${data.id}/tickets`)
  }

  return (
    <div className="flex">
      <Button type="button" variant="secondary" className="cursor-pointer" onClick={viewTicketsClicked}>
        View Tickets
      </Button>
    </div>
  )
}