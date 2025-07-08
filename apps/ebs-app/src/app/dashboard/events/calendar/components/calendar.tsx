'use client'

import { Button } from '@/components/ui/button'
import { connectCalendar } from '@/lib/actions'
import { Organization } from '@/lib/types'
import { use, useCallback } from 'react'

type Props = {
  calendarResolver: Promise<string | null>,
  organizationResolver: Promise<Organization | null>,
}
export default function EventsCalendarComponent({ calendarResolver, organizationResolver }: Props) {
  const calendar = use(calendarResolver)
  const organization = use(organizationResolver)
  const connect = useCallback(async () => {
    const url = await connectCalendar('/dashboard/events/calendar')
    window.open(url ?? '', '_blank')
  }, [])
  return (
    calendar
      ? <div className="h-full">
          <iframe src={`https://calendar.google.com/calendar/embed?wkst=1&ctz=${encodeURIComponent(organization?.timezone as string)}&src=${calendar}&showPrint=0&showCalendars=0`} style={{ border: 'solid 1px #777' }} className="w-full h-full"></iframe>
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