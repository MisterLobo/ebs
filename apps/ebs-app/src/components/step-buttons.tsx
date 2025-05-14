'use client'

import { Button } from '@/components/ui/button'
import { useStepper } from '@/components/ui/stepper'
import { useRouter } from 'next/navigation'

export function StepButtons() {
  const { nextStep, prevStep, isLastStep, isOptionalStep, isDisabledStep } = useStepper()

  return (
    <div className="mb-4 flex w-full gap-2">
      <Button
        disabled={isDisabledStep}
        onClick={prevStep}
        size="sm"
        variant="secondary"
      >Prev</Button>
      <Button size="sm" onClick={nextStep} className="cursor-pointer">
        {isLastStep ? 'Finish' : isOptionalStep ? 'Skip' : 'Next' }
      </Button>
    </div>
  )
}

export function FinalStep() {
  const router = useRouter()
  const { hasCompletedAllSteps } = useStepper()

  if (!hasCompletedAllSteps) return null

  const continueToDashboard = async () => {
    router.push('/dashboard')
  }

  return (
    <>
      <div className="bg-secondary text-primary flex h-40 items-center justify-center rounded-md  border">
        <h1 className="text-xl">Woohoo! All steps completed! 🎉</h1>
      </div>
      <div className="flex w-full justify-end gap-2">
        <Button size="sm" onClick={continueToDashboard} className="cursor-pointer">
          Continue to Dashboard
        </Button>
      </div>
    </>
  )
}