"use client"

import * as React from 'react'
import {
  AudioWaveform,
  BookOpen,
  Bot,
  Command,
  Frame,
  GalleryVerticalEnd,
  Map,
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
          url: '/dashboard/tickets_templates',
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
      title: 'Reservations',
      url: '/dashboard/reservations',
      icon: SquareTerminal,
      isActive: true,
      items: [
        {
          title: 'Create Reservations',
          url: '#',
        },
        {
          title: 'Manage Reservations',
          url: '/dashboard/reservations',
        },
      ],
    },
    {
      title: 'Admissions',
      url: '#',
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
      url: '#',
      icon: Bot,
      items: [
        {
          title: 'Revenue Summary',
          url: '#',
        },
      ],
    },
    {
      title: 'Settings',
      url: '#',
      icon: Settings2,
      items: [
        {
          title: 'General',
          url: '#',
        },
        {
          title: 'Team',
          url: '#',
        },
        {
          title: 'Billing',
          url: '#',
        },
        {
          title: 'Limits',
          url: '#',
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
    {
      name: 'Travel',
      url: '#',
      icon: Map,
    },
  ],
}

type Props = {
  teams: Organization[],
}

export function AppSidebar({ teams, ...props }: React.ComponentProps<typeof Sidebar> & Props) {
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
        <NavUser user={data.user} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
