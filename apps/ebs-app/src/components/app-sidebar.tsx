"use client"

import * as React from 'react'
import {
  AudioWaveform,
  BookOpen,
  Bot,
  Command,
  Frame,
  GalleryVerticalEnd,
  PieChart,
  Settings2,
  SquareTerminal,
} from 'lucide-react'

import { NavMain } from '@/components/nav-main'
import { NavProjects } from '@/components/nav-projects'
import { NavUser } from '@/components/nav-user'
import { TeamSwitcher } from '@/components/team-switcher'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from '@/components/ui/sidebar'
import { NavPersonal } from './nav-personal'
import { Organization } from '@/lib/types'
import { me } from '@/lib/actions'

// This is sample data.
const data = {
  user: {
    name: 'shadcn',
    email: 'm@example.com',
    avatar: '/avatars/shadcn.jpg',
  },
  teams: [
    {
      name: 'Acme Inc',
      logo: GalleryVerticalEnd,
      plan: 'Enterprise',
    },
    {
      name: 'Acme Corp.',
      logo: AudioWaveform,
      plan: 'Startup',
    },
    {
      name: 'Evil Corp.',
      logo: Command,
      plan: 'Free',
    },
  ],
  navMain: [
    {
      title: 'Dashboard',
      url: '/dashboard',
      icon: SquareTerminal,
      isActive: true,
      items: [
        {
          title: 'Onboarding',
          url: '/dashboard/setup',
        },
        {
          title: 'Overview',
          url: '/dashboard',
        },
      ],
    },
    {
      title: 'Events',
      url: '/dashboard/events',
      icon: SquareTerminal,
      isActive: true,
      items: [
        {
          title: 'New Event',
          url: '/dashboard/events/new',
        },
        {
          title: 'Manage Events',
          url: '/dashboard/events',
        },
      ],
    },
    {
      title: 'Tickets',
      url: '/dashboard/tickets',
      icon: BookOpen,
      isActive: true,
      items: [
        {
          title: 'New Ticket Price',
          url: '/dashboard/tickets/new',
        },
        {
          title: 'Ticket Prices',
          url: '/dashboard/tickets',
        },
        {
          title: 'Templates',
          url: '/dashboard/tickets/templates',
        },
      ],
    },
    {
      title: 'Bookings',
      url: '/dashboard/bookings',
      icon: SquareTerminal,
      isActive: true,
      items: [
        {
          title: 'Create Booking',
          url: '/dashboard/bookings/new',
        },
        {
          title: 'Manage Bookings',
          url: '/dashboard/bookings',
        },
      ],
    },
    {
      title: 'Admissions',
      url: '/dashboard/admissions',
      icon: SquareTerminal,
      isActive: true,
      items: [
        {
          title: 'Create Admission',
          url: '/dashboard/admissions/new',
        },
        {
          title: 'Manage Admissions',
          url: '/dashboard/admissions',
        },
      ],
    },
    {
      title: 'Sales',
      icon: Bot,
      items: [
        {
          title: 'Revenue Summary',
          url: '/dashboard//sales/revenue',
        },
      ],
    },
    {
      title: 'Settings',
      icon: Settings2,
      items: [
        {
          title: 'General',
          url: '/settings#general',
        },
        {
          title: 'Team',
          url: '/settings#team',
        },
        {
          title: 'Billing',
          url: '/settings#billing',
        },
        {
          title: 'Limits',
          url: '/settings#limits',
        },
      ],
    },
  ],
  personal: [
    {
      name: 'Tickets',
      url: '/personal/dashboard',
      icon: Frame,
    },
  ],
  projects: [
    {
      name: 'Home',
      url: '/',
      icon: Frame,
    },
    {
      name: 'Browse',
      url: '/events',
      icon: PieChart,
    },
  ],
}

type Props = {
  teams: Organization[],
}

export function AppSidebar({ teams, ...props }: React.ComponentProps<typeof Sidebar> & Props) {
  const [userData, setUserData] = React.useState<any>()
  React.useEffect(() => {
    me().then(d => {
      setUserData(d?.me)
    })
  }, [])
  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <TeamSwitcher organizations={teams} teams={data.teams} />
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={data.navMain} />
        <NavPersonal projects={data.personal} />
        <NavProjects projects={data.projects} />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={userData} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
