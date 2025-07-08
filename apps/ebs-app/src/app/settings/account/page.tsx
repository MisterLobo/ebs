import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { SettingsAccountGeneral, SettingsAccountSecurity } from './components'
import { getMFADevices, me } from '@/lib/actions'

export default async function AccountSettingsPage() {
  const user = await me()
  const devices = await getMFADevices()
  return (
    <div className="mx-auto w-3xl mt-10 p-2 min-h-96 space-y-4">
      <h1 className="text-3xl">Account</h1>
      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="text-xl">General</h2>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <SettingsAccountGeneral user={user?.me} />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="text-xl">Security</h2>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <SettingsAccountSecurity devices={devices} />
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
            <p>Deactivate account</p>
            <Button variant="destructive" className="w-32 text-red-300" disabled>Deactivate</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}