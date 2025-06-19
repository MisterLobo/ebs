import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { getPaymentMethods, getSubscription } from '@/lib/actions'
import { format } from 'date-fns'

export default async function BillingSettingsPage() {
  const {
    sub,
    curr,
    price,
    prod,
  } = await getSubscription()
  const paymentMethods = await getPaymentMethods() ?? []
  return (
    <div className="mx-auto w-3xl mt-10 p-2 min-h-96 space-y-4">
      <h1 className="text-3xl">Billing</h1>
      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="text-xl">Subscription</h2>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="capitalize">Status: <Badge>{ sub?.status }</Badge></p>
          <p>Period: { format((curr?.current_period_start ?? 0)*1e3, 'MMM dd') } to { format((curr?.current_period_end ?? 0)*1e3, 'MMM dd') }</p>
          <p>Current Plan: { prod?.name.toUpperCase() }</p>
          <p>{ price?.currency.toUpperCase() } { price?.unit_amount } per { price?.recurring?.interval }</p>
          <CardAction>
            <Button>Upgrade</Button>
          </CardAction>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="text-xl">Cards</h2>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {paymentMethods?.length > 0 ? paymentMethods?.map(pm => (
            <p key={pm.id}>{ pm.card?.last4 }</p>
          )) : (
            <div className="flex items-center justify-center w-full flex-col gap-2">
              <p>No cards added</p>
              <Button className="w-32">Add a Card</Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}