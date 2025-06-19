import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export default async function AccountSettingsPage() {
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
          <div className="flex gap-2 items-center w-full justify-between my-5">
            <p>Email: email@domain.com</p>
            <Button className="w-32">Verify email</Button>
          </div>
          <div className="flex gap-2 items-center w-full justify-between my-5">
            <p>Phone: +100****000</p>
            <Button className="w-32">Verify phone</Button>
          </div>
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
            <Button variant="destructive" className="w-32 text-red-300">Deactivate</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}