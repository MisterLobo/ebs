
import { aboutOrganization, getEvents } from '@/lib/actions'
import { importURLPatternPolyfill } from '@/lib/utils'
import { CheckCircle } from 'lucide-react'
import { headers } from 'next/headers'
import EventCard from './components/event-card'

export default async function SlugAboutPage() {
  await importURLPatternPolyfill()
  const $headers = await headers()
  const url = $headers.get('x-url')
  const pattern = new URLPattern({ pathname: '/:slug/about' })
  const result = pattern.exec(url as string)
  const slug = result?.pathname.groups.slug as string
  const about = await aboutOrganization({ slug })
  const events = await getEvents(about?.id, { public: 'true' })
  return (
    <div className="container">
      <p className="text-3xl">About {about?.name}</p>
      <p className="text-sm">{about?.about ?? 'No description'}</p>
      <p className="text-xs">Contact: {about?.email}</p>
      <p className="text-xs">{about?.country}</p>
      {about?.verified && <p className="flex items-center gap-2"><CheckCircle /><span>Verified Account</span></p>}
      <p className="flex items-center gap-2 text-xs">Country: {about?.country ?? 'N/A'}</p>
      {about?.payment_verified && <p className="flex items-center gap-2"><CheckCircle /><span>Payment Verified</span></p>}
      <p className="uppercase text-3xl my-4">Events ({events.length})</p>
      {events ?
      <div className="flex flex-col gap-2">
      {events.map(e => (
        <EventCard key={e.id} data={e} />
      ))}
      </div> :
      <p className="text-center">No events</p>
      }
    </div>
  )
}