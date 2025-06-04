import { DataTable } from './components/data-table'
import { getActiveOrganization, getOrganizationTickets } from '@/lib/actions'

export default async function EventPage() {
  const org = await getActiveOrganization()
  const tickets = await getOrganizationTickets(org?.id as number)
  return (
    <div className="flex flex-1 flex-col">
      <div className="@container/main flex flex-1 flex-col gap-2">
        <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
          <DataTable data={tickets} />
        </div>
      </div>
    </div>
  )
}
