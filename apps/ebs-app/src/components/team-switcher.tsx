"use client"

import { ChevronsUpDown, Plus, TicketIcon } from 'lucide-react'

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { useRouter } from 'next/navigation'
import { Button } from './ui/button'
import { Organization } from '@/lib/types'
import { ElementType, useCallback, useEffect, useState } from 'react'
import { getActiveOrganization, switchOrganization } from '@/lib/actions'

export function TeamSwitcher({
  organizations,
  teams,
}: {
  organizations: Organization[],
  teams: {
    name: string
    logo: ElementType
    plan: string
  }[]
}) {
  const router = useRouter()
  const { isMobile } = useSidebar()
  const [activeOrg, setActiveOrg] = useState<Organization>()

  useEffect(() => {
    getActiveOrganization().then(active => setActiveOrg(active))
  }, [])

  const newOrg = () => {
    router.push('/setup/organizations/create')
  }

  const switchToOrg = useCallback(async (org: Organization) => {
    const switched = await switchOrganization(org.id)
    if (switched) {
      location.reload()
    }
  }, [])

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          {activeOrg ?
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                <TicketIcon className="size-4" />
              </div>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">
                  {activeOrg?.name}
                </span>
                <span className="truncate text-xs">{activeOrg?.type}</span>
              </div>
              <ChevronsUpDown className="ml-auto" />
            </SidebarMenuButton>
          </DropdownMenuTrigger> :
          <DropdownMenuTrigger asChild disabled>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
              disabled
            >
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                <TicketIcon className="size-4" />
              </div>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">
                  loading
                </span>
              </div>
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          }
          <DropdownMenuContent
            className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
            align="start"
            side={isMobile ? "bottom" : "right"}
            sideOffset={4}
          >
            <DropdownMenuLabel className="text-xs text-muted-foreground">
              Organizations
            </DropdownMenuLabel>
            {organizations.map((org, index) => (
              <DropdownMenuItem
                key={index}
                onClick={() => switchToOrg(org)}
                className="gap-2 p-2"
              >
                <div className="flex size-6 items-center justify-center rounded-sm border">
                  <TicketIcon className="size-4 shrink-0" />
                </div>
                {org.name}
                <DropdownMenuShortcut>⌘{index + 1}</DropdownMenuShortcut>
              </DropdownMenuItem>
            ))}
            <DropdownMenuSeparator />
            <DropdownMenuItem className="gap-2 p-2">
              <div className="flex size-6 items-center justify-center rounded-md border bg-background">
                <Plus className="size-4" />
              </div>
              <Button variant="ghost" className="font-medium text-muted-foreground cursor-pointer" onClick={newOrg}>Create Organization</Button>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
