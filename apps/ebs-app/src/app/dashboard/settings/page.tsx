import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import SettingsGeneralForm from './components/forms'
import { Loader2 } from 'lucide-react'
import { Suspense } from 'react'
import { getActiveOrganization } from '@/lib/actions'
import { Button } from '@/components/ui/button'

export default async function SettingsPage() {
  return (
    <div className="mx-auto mt-10">
      <div className="flex flex-col gap-4">
        <Card className="min-w-3xl">
          <CardHeader>
            <CardTitle id="general" className="text-xl">Profile</CardTitle>
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
        <Card>
          <CardHeader>
            <CardTitle>
              <h2 className="text-xl text-red-600">Danger Zone</h2>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex w-full justify-between text-red-600">
              <p>Deactivate Organization</p>
              <Button variant="destructive" className="w-64 text-red-300">Deactivate Organization</Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}