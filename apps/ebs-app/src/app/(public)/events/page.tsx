import { getEvents } from '@/lib/actions'
import EventFiltersHeader from './components/filters'
import EventCard from './components/event-card'
import { Fragment } from 'react'

export default async function EventsPage() {
  const events = await getEvents()
  return (
    <div className="flex flex-col w-full">
      {events.length > 0 ?
      <>
      <h1 className="text-2xl">Events: { events.length }</h1>
      <div className="flex w-full">
        <EventFiltersHeader />
      </div>
      <div className="flex flex-col gap-4 justify-center w-full items-center">
        {events.map(event => (
          <Fragment key={event.id}>
            <EventCard data={event} />
          </Fragment>
        ))}
        <h2 className="text-center text-muted italic">End of list</h2>
      </div>
      </> :
      <p className="text-xl italic w-full text-center mt-10">No events yet. </p>
      }
    </div>
  )
}