'use client'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { AlertTriangle } from 'lucide-react'
import { useRouter } from 'next/navigation'

export function OnboardingNotice({ url }: { url?: string }) {
  const continueOnboarding = () => {
    if (url) {
      location.href = url
    }
  }
  return (
    <Alert variant="default" color="green" className="space-x-2 rounded">
      <AlertTriangle />
      <AlertTitle>NOTICE</AlertTitle>
      <AlertDescription className="flex items-center gap-2">
        <div className="text-sm inline-flex min-w-xl break-words text-wrap">Complete onboarding to remove restrictions</div>
        <div className="inline-flex">
          <Button className="cursor-pointer disabled:pointer-events-none" onClick={continueOnboarding} disabled={!url}>Continue onboarding</Button>
        </div>
      </AlertDescription>
    </Alert>
  )
}

export function OnboardingIncomplete() {
  const router = useRouter()
  const continueOnboarding = () => {
    router.push('/dashboard/setup')
  }
  return (
    <div className="container flex items-center justify-center">
      <div className="flex flex-col mt-20 size-96 items-center justify-center">
        <Card className="w-full h-full">
          <CardHeader>
            <CardTitle>NOTICE</CardTitle>
          </CardHeader>
          <Separator />
          <CardContent className="h-full space-y-2">
            <p>Finish setting up to continue</p>
            <p>Billing information is required to start selling tickets</p>
            <Separator />
          </CardContent>
          <CardAction className="w-full px-4">
            <Button className="w-full cursor-pointer" onClick={continueOnboarding}>Continue</Button>
          </CardAction>
        </Card>
      </div>
    </div>
  )
}