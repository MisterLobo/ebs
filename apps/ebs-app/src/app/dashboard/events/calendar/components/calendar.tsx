'use client'

import { Button } from '@/components/ui/button'
import { connectCalendar } from '@/lib/actions'
import { use, useCallback } from 'react'

type Props = {
  calendarResolver: Promise<string | null>,
}
export default function EventsCalendarComponent({ calendarResolver }: Props) {
  const calendar = use(calendarResolver)
  const connect = useCallback(async () => {
    const url = await connectCalendar('/dashboard/events/calendar')
    window.open(url ?? '', '_blank')
  }, [])
  return (
    calendar
      ? <div className="h-full">
          <iframe src={`https://calendar.google.com/calendar/embed?wkst=1&ctz=Asia%2FManila&src=${calendar}&showPrint=0&showCalendars=0`} style={{ border: 'solid 1px #777' }} className="w-full h-full"></iframe>
        </div>
      : (
        <div className="container mx-auto mt-10">
          <div className="flex items-center justify-center h-96">
            <Button onClick={connect}>Connect Calendar</Button>
          </div>
        </div>
      )
  )
}