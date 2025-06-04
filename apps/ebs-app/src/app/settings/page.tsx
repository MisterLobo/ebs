import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import SettingsGeneralForm from './components/forms'
import { Loader2 } from 'lucide-react'
import { Suspense } from 'react'
import { getActiveOrganization } from '@/lib/actions'

export default async function SettingsPage() {
  return (
    <div className="mx-auto mt-10 flex flex-col gap-4">
      <Card className="min-w-3xl">
        <CardHeader>
          <CardTitle id="general" className="text-xl">General</CardTitle>
        </CardHeader>
        <CardContent>
          <SettingsGeneralForm resolver={getActiveOrganization()} />
        </CardContent>
      </Card>
      <Card className="min-w-3xl">
        <CardHeader>
          <CardTitle className="text-xl">Team</CardTitle>
        </CardHeader>
        <CardContent>
          <Suspense fallback={<Loader2 className="animate-spin size-12" />}>
            <p>Data unavailable</p>
          </Suspense>
        </CardContent>
      </Card>
      <Card className="min-w-3xl">
        <CardHeader>
          <CardTitle className="text-xl">Billing</CardTitle>
        </CardHeader>
        <CardContent>
          <Suspense fallback={<Loader2 className="animate-spin size-12" />}>
            <p>Data unavailable</p>
          </Suspense>
        </CardContent>
      </Card>
    </div>
  )
}