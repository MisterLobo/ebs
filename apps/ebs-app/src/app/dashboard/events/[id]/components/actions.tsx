'use client'

import { Button } from '@/components/ui/button'
import { getActiveOrganization, organizationOnboarding, publishEvent, setEventStatus } from '@/lib/actions'
import { Event } from '@/lib/types'
import { PlusIcon, Send } from 'lucide-react'
import { useParams, useRouter } from 'next/navigation'
import { useCallback, useEffect, useMemo, useState } from 'react'

type Props = {
  event?: Event,
}
export function EventPageHeaderActions({ event }: Props) {
  const router = useRouter()
  const params = useParams()
  const eventId = useMemo(() => params.id, [params])
  const [busy, setBusy] = useState(false)
  const [loading, setLoading] = useState(true)
  const [onboardingComplete, setOnboardingComplete] = useState(false)

  useEffect(() => {
    setLoading(true)
    getActiveOrganization().then(org => organizationOnboarding(org?.id as number)).then(res => {
      setOnboardingComplete(res.completed ?? false)
      setLoading(false)
    })
  }, [])

  const newTicket = () => {
    router.push(`/dashboard/events/${eventId}/tickets/new`)
  }

  const publish = useCallback(async () => {
    if (!onboardingComplete) {
      return
    }
    setBusy(true)
    const error = await publishEvent(Number(eventId))
    setBusy(false)
    if (error) {
      alert(error)
      return
    }
    router.refresh()
  }, [router, onboardingComplete, eventId])

  const openAdm = useCallback(async () => {
    setBusy(true)
    await setEventStatus(event?.id as number, 'admission')
    setBusy(false)
    router.refresh()
  }, [])
  const closeAdm = useCallback(async () => {
    setBusy(true)
    await setEventStatus(event?.id as number, 'closed')
    setBusy(false)
    router.refresh()
  }, [])

  return (
    <div className="flex w-full p-4 relative gap-2">
      <Button type="button" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none" onClick={newTicket} variant="default" disabled={!onboardingComplete || loading}>
        <PlusIcon />
        <span>NEW TICKET</span>
      </Button>
      {event?.status === 'notify' &&
        <Button type="button" className="cursor-pointer" onClick={publish} disabled={!onboardingComplete || loading || busy}><Send />{ busy ? 'REQUESTING' : 'CANCEL' }</Button>
      }
      {['draft', 'notify'].includes(event?.status ?? '') &&
        <Button type="button" className="cursor-pointer" onClick={publish} disabled={!onboardingComplete || loading || busy}><Send />{ busy ? 'PUBLISHING' : 'PUBLISH' }</Button>
      }
      {event?.status === 'registration' &&
        <Button type="button" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none" onClick={openAdm}>OPEN ADMISSION</Button>}{event?.status === 'admission' && <Button type="button" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none" onClick={closeAdm}>CLOSE ADMISSION</Button>
      }
    </div>
  )
}