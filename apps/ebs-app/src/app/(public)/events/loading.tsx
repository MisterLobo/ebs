import { Loader2 } from 'lucide-react'
import EventFiltersHeader from './components/filters'
import { Skeleton } from '@/components/ui/skeleton'

export default async function EventsPageLoader() {
  return (
    <div className="flex flex-col w-full">
      <h1 className="text-2xl">Events: <Loader2 className="animate-spin" /></h1>
      <div className="flex w-full">
        <EventFiltersHeader />
      </div>
      <div className="flex flex-col gap-4 justify-center w-full items-center">
        <Skeleton className="animate-shimmer" />
      </div>
    </div>
  )
}
