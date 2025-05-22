import { getActiveOrganization, organizationOnboarding } from '@/lib/actions'
import { CheckIcon, XIcon } from 'lucide-react'
import { SetupActions } from './components/actions'

export default async function OnboardingSetup() {
  const org = await getActiveOrganization()
  const { completed, data } = await organizationOnboarding(org?.id as number)
  console.log('data:', completed, data);
  
  return (
    <div className="container pt-10">
      <h2 className="text-3xl">Continue Setting Up <span className="font-semibold">{ org?.name }</span></h2>
      <h2 className="text-md">Onboarding checklist</h2>
      <div className="flex flex-col mt-10">
      <h2 className="text-2xl">Errors:</h2>
      {data?.errors?.length ? <p>There are errors</p> : <p>No errors</p>}
      {data?.errors &&
      data?.errors?.map((err: any) => (
        <p>{ JSON.stringify(err) }</p>
      ))}
      </div>
      <div className="flex flex-col mt-10">
        <h2 className="text-2xl">Billing</h2>
        <h2 className="text-lg">Charges: { data?.chargesEnabled ? 'you can charge money to your customers' : 'you can\'t charge money to your customers' }</h2>
        <h2 className="text-lg">Payouts: { data?.chargesEnabled ? 'you can receive money to your bank account' : 'you can\'t receive money to you bank account' }</h2>
      </div>
      <div className="flex flex-col mt-10">
        <h2 className="text-2xl">Details Submitted</h2>
        <h2 className="text-lg">{ data?.detailsSubmitted ? 'Account details have been submitted' : 'Account details are incomplete' }</h2>
      </div>
      <div className="flex flex-col mt-10 gap-0">
        <h2 className="text-2xl">Events</h2>
        <h2 className="text-lg flex items-center">Publish events: { completed ? <CheckIcon className="text-green-500" /> : <XIcon className="text-red-500" />}</h2>
        <h2 className="text-lg flex items-center">Schedule events: { completed ? <CheckIcon className="text-green-500" /> : <XIcon className="text-red-500" />}</h2>
      </div>
      <div className="flex mt-10 text-neutral-500">
      {completed ?
        <p>All done</p> :
        <SetupActions onboardingUrl={org?.connect_onboarding_url} />
      }
      </div>
    </div>
  )
}