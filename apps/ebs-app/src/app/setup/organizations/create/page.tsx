import { CreateOrganizationSteps, StepItemWithChild } from './components/create-steps'

const steps = [
  {
    label: 'Basic',
    description: 'Provide organization name, country and details',
  },
  { label: 'Billing',
    description: 'Provide billing inforation',
  },
  {
    label: 'Terms and Conditions',
    description: 'Accept the terms and conditions',
  }
] satisfies StepItemWithChild[]

export default async function NewOrganization() {
  return (
    <div className="container min-h-screen pt-10 space-y-4">
      <h1 className="text-3xl text-center">Create Organization</h1>
      <CreateOrganizationSteps steps={steps} className="mx-auto max-w-xl" />
    </div>
  )
}