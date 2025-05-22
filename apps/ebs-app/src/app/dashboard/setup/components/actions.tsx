'use client'

import { Button } from '@/components/ui/button'

export function SetupActions({ onboardingUrl }: { onboardingUrl?: string }) {
  const onClickContinue = () => {
    if (!onboardingUrl) {
      return
    }
    alert(onboardingUrl)
    window.open(onboardingUrl, '_blank')
  }
  return (
    <Button className="mt-10 w-96 cursor-pointer" onClick={onClickContinue} disabled={!onboardingUrl}>Continue onboarding on Stripe</Button>
  )
}