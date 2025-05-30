import { SidebarTrigger } from '@/components/ui/sidebar'
import NewEventForm from './components/forms'
import { Separator } from '@/components/ui/separator'
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from '@/components/ui/breadcrumb'
import { getActiveOrganization, organizationOnboarding } from '@/lib/actions'

export default async function NewEventPage() {
  const org = await getActiveOrganization()
  const { completed } = await organizationOnboarding(org?.id as number)
  return (
    <>
    <header className="flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-12">
      <div className="flex items-center gap-2 px-4">
        <SidebarTrigger className="-ml-1" />
        <Separator orientation="vertical" className="mr-2 h-4" />
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem className="hidden md:block">
              <BreadcrumbLink href="/dashboard/events">
                Events
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator className="hidden md:block" />
            <BreadcrumbItem>
              <BreadcrumbPage>New</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>
    </header>
    <div className="mx-auto py-10 space-y-2">
      <h1 className="text-3xl font-semibold">New Event</h1>
      <NewEventForm onboardingComplete={completed} />
    </div>
    </>
  )
}