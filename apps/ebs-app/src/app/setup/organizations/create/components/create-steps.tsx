'use client'

import { Step, StepItem, Stepper, useStepper } from '@/components/ui/stepper'
import { FinalStep } from '../../../../../components/step-buttons'
import { ComponentProps, ReactNode, useCallback, useState } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { Checkbox } from '@/components/ui/checkbox'
import { createOrganization, organizationOnboardingBegin } from '@/lib/actions'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { useRouter } from 'next/navigation'

export type StepItemWithChild = StepItem & { component?: ReactNode }

type Props = {
  steps: StepItemWithChild[],
}
export function CreateOrganizationSteps({ steps, className, ...props }: Props & ComponentProps<'div'>) {
  const [orgId, setOrgId] = useState<number>()
  return (
    <div className={cn(className, 'flex w-full flex-col gap-4')} {...props}>
      <Stepper
        orientation="vertical"
        initialStep={0}
        steps={steps}
        scrollTracking
      >
        <Step title="Basic" description="Provide basic information about the Organization">
          <div className="bg-secondary text-primary my-4 flex min-h-64 items-center justify-center rounded-md border w-full">
            <BasicInformationStep onSuccess={setOrgId} />
          </div>
        </Step>
        <Step title="Basic" description="Provide basic information about the Organization">
          <div className="bg-secondary text-primary my-4 flex min-h-64 items-center justify-center rounded-md border w-full">
            <BillingInformationStep orgId={orgId ?? 0} />
          </div>
        </Step>
        <Step title="Basic" description="Provide basic information about the Organization">
          <div className="bg-secondary text-primary my-4 flex min-h-64 items-center justify-center rounded-md border w-full">
            <TermsAndConditionsStep />
          </div>
        </Step>
        <FinalStep orgId={orgId ?? 0} />
      </Stepper>
    </div>
  )
}

const BasicInforationSchema = z.object({
  name: z.string().min(1, {
    message: 'Name must be at least 1 character.',
  }),
  about: z.string().optional(),
  country: z.string().min(1).optional(),
  type: z.enum(['personal', 'standard']).default('standard').optional(),
  email: z.string().email('Must be a valid email address'),
})
export function BasicInformationStep({ onSuccess, onError }: { onSuccess?: (id: number) => void, onError?: (e: Error) => void }) {
  const [busy, setBusy] = useState(false)
  const { nextStep } = useStepper()
  const [error, setError] = useState<string>()

  const form = useForm<z.infer<typeof BasicInforationSchema>>({
    resolver: zodResolver(BasicInforationSchema),
    defaultValues: {
      name: '',
      type: 'standard',
    },
  })

  async function onSubmit(data: z.infer<typeof BasicInforationSchema>) {
    setBusy(true)
    const { id, error } = await createOrganization(data)
    if (error) {
      setError(error)
      return
    }
    if (!id && !error) {
      const error = 'Could not retrieve information at the moment'
      setError('Could not retrieve information at the moment')
      if (onError) {
        onError(new Error(error))
      }
      return
    }
    setBusy(false)
    if (onSuccess) {
      onSuccess(id)
    }
    nextStep()
  }

  return (
    <div className="h-3xl w-full m-4">
      {error &&
      <Alert variant="destructive">
        <AlertTitle>ERROR</AlertTitle>
        <AlertDescription>{ error }</AlertDescription>
      </Alert>}
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6 h-3xl">
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Name<span className="text-red-500">*</span></FormLabel>
                <FormControl>
                  <Input placeholder="name" {...field} />
                </FormControl>
                <FormDescription>
                  Name of your Organization
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Email<span className="text-red-500">*</span></FormLabel>
                <FormControl>
                  <Input placeholder="email" {...field} />
                </FormControl>
                <FormDescription>
                  Email address for contacting your organization
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <StepperFormActions disabled={busy}/>
        </form>
      </Form>
    </div>
  )
}

type PropsWithId = {
  orgId: number,
}
const BillingInforationSchema = z.object({
  onboarded: z.coerce.boolean(),
})
export function BillingInformationStep({ orgId }: PropsWithId) {
  const { nextStep } = useStepper()
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)

  const form = useForm<z.infer<typeof BillingInforationSchema>>({
    resolver: zodResolver(BillingInforationSchema),
    defaultValues: {
      onboarded: false,
    },
  })

  const beginOnboarding = useCallback(async () => {
    setBusy(true)
    const { url, account_id, error } = await organizationOnboardingBegin(orgId ?? 0)
    if (error) {
      setError(error)
      return
    }
    if (url && account_id) {
      form.setValue('onboarded', true)
    }
    setBusy(false)
    if (url) {
      window.open(url, '_blank')
    }
  }, [orgId])

  async function onSubmit(data: z.infer<typeof BillingInforationSchema>) {
    /* const { account_id, error } = await organizationOnboarding(orgId ?? 0)
    setError(error) */

    if (!data.onboarded) {
      setError('Must be onboarded on Stripe first')
      form.setError('onboarded', { message: 'Must be onboarded on Stripe first' })
      return
    }

    /* if (account_id && !error) {
      form.setValue('onboarded', true)
    } */

    nextStep()
  }

  return (
    <div className="h-3xl w-full m-4">
      {error &&
      <Alert variant="destructive">
        <AlertTitle>ERROR</AlertTitle>
        <AlertDescription>{ error }</AlertDescription>
      </Alert>
      }
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6 h-3xl">
          {/* <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Name</FormLabel>
                <FormControl>
                  <Input placeholder="name" {...field} />
                </FormControl>
                <FormDescription>
                  Name of your Organization
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          /> */}
          <Button className="cursor-pointer" onClick={beginOnboarding}>Begin onboarding</Button>
          <StepperFormActions disabled={busy} />
        </form>
      </Form>
    </div>
  )
}

const TermsAndAgreementSchema = z.object({
  termsAccepted: z.boolean({ required_error: 'required' }),
})
export function TermsAndConditionsStep() {
  const { nextStep } = useStepper()
  const [accepted, setAccepted] = useState(false)

  const form = useForm<z.infer<typeof TermsAndAgreementSchema>>({
    defaultValues: {
      termsAccepted: false,
    },
  })

  function onSubmit(data: z.infer<typeof TermsAndAgreementSchema>) {
    if (!accepted) {
      form.setError('termsAccepted', { message: 'Terms and Conditions must be accepted' })
      return
    }
    form.setValue('termsAccepted', accepted)
    form.trigger()
    nextStep()
  }

  return (
    <div className="h-3xl w-full m-4">
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6 h-3xl">
          <FormField
            control={form.control}
            name="termsAccepted"
            render={({ field: { onChange, value, ...rest } }) => (
              <FormItem>
                <FormLabel>Accept</FormLabel>
                <FormControl>
                  <div className="flex items-center space-x-2">
                    <Checkbox id="termsAccepted" {...rest} checked={accepted} onCheckedChange={e => {
                      onChange(e)
                      setAccepted(e as boolean)
                    }} className="border border-white" />
                    <label htmlFor="termsAccepted" className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">I understand and accept the Terms and Conditions</label>
                  </div>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Button className="cursor-pointer">Finish</Button>
        </form>
      </Form>
    </div>
  )
}

function StepperFormActions({ disabled }: ComponentProps<'button'>) {
  const router = useRouter()
  const {
    hasCompletedAllSteps,
    isLastStep,
    isOptionalStep,
    isDisabledStep,
    prevStep,
  } = useStepper()
  const continueToDashboard = async () => {
    router.push('/dashboard')
  }
  return (
    <div className="flex w-full justify-end gap-2">
      {hasCompletedAllSteps ? (
        <Button size="sm" onClick={continueToDashboard}>Continue to Dashboard</Button>
      ) : (
        <>
          <Button
            disabled={isDisabledStep}
            onClick={prevStep}
            size="sm"
            variant="secondary"
            className="cursor-pointer"
          >Prev</Button>
          <Button size="sm" className="cursor-pointer" disabled={disabled}>{ isLastStep ? 'Finish' : isOptionalStep ? 'Skip' : 'Next' }</Button>
        </>
      )}
    </div>
  )
}