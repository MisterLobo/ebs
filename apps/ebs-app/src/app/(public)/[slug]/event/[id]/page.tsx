import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { getEventById, getTickets } from '@/lib/actions'
import { importURLPatternPolyfill } from '@/lib/utils'
import { format } from 'date-fns'
import { headers } from 'next/headers'
import { notFound, redirect } from 'next/navigation'

export default async function EventPage() {
  await importURLPatternPolyfill()
  const pattern = new URLPattern({ pathname: '/events/:id' })
  const $headers = await headers()
  const url = $headers.get('x-url')
  const result = pattern.exec(url as string)
  const id = result?.pathname.groups.id
  if (!id) {
    redirect('/')
  }
  const eventId = parseInt(id)
  const event = await getEventById(eventId)
  if (!event) {
    throw notFound()
  }
  const tickets = await getTickets(eventId)
  return (
    <div className="flex flex-col w-full p-4">
      <h1 className="text-xl font-semibold">{ event.title }</h1>
      <h2 className="text-xl">{ event.location }</h2>
      <h2 className="text-xl">{ format(new Date(event.date_time as string), 'PPPP p') }</h2>
      <span>{tickets.length} tickets</span>
      {tickets.map(ticket => (
        <Card key={ticket.id}>
          <CardHeader>
            <CardTitle>{ ticket.tier }</CardTitle>
          </CardHeader>
          <CardContent>
            <p>{ ticket.type }</p>
            <p>{ ticket.currency?.toUpperCase() }{ ticket.price }</p>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}