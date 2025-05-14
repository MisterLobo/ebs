'use client'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { AlertTriangle } from 'lucide-react'

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