import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { LoaderCircle } from 'lucide-react'
import { Suspense } from 'react'
import { PersonalDashboardClient } from './client'

export default async function PersonalDashboard() {
  return (
    <main className="p-6 container">
      <h1 className="text-3xl font-semibold">My Dashboard</h1>
      <Suspense fallback={
        <>
        <h2 className="text-xl">Reservations: <LoaderCircle className="animate-spin" /></h2>
        <div className="flex flex-col gap-4 items-center justify-center">
          <Tabs defaultValue="completed">
            <TabsList className="w-fit">
              <TabsTrigger value="completed">Completed</TabsTrigger>
              <TabsTrigger value="pending">Pending</TabsTrigger>
              <TabsTrigger value="Canceled">Canceled</TabsTrigger>
            </TabsList>
            <TabsContent value="completed" className="w-3xl h-auto">
              <LoaderCircle className="animate-spin" />
            </TabsContent>
            <TabsContent value="pending" className="w-3xl h-auto">
              <LoaderCircle className="animate-spin" />
            </TabsContent>
            <TabsContent value="Canceled" className="w-3xl h-auto">
              <LoaderCircle className="animate-spin" />
            </TabsContent>
          </Tabs>
        </div>
        </>
      }>
        <PersonalDashboardClient />
      </Suspense>
    </main>
  )
}