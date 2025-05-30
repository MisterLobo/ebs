'use client'

import { Button } from '@/components/ui/button'
import { useRouter } from 'next/navigation'

export function EventsHeaderActions() {
  const router = useRouter()

  const newTicket = () => {
    router.push(`/dashboard/events/new`)
  }
  return (
    <div className="flex w-full p-4 relative">
      <Button type="button" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none" onClick={newTicket}>NEW EVENT</Button>
    </div>
  )
}