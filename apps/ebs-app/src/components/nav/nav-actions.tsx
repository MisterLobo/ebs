'use client'

import { useRouter } from 'next/navigation'
import { Button } from '../ui/button'
import { SunIcon } from 'lucide-react'
import { NavigationSheet } from './navigation-sheet'
import { ComponentProps } from 'react'
import { cn } from '@/lib/utils'

type Props = {
  authenticated: boolean,
  hasOrg: boolean,
}
export default function NavActions({ authenticated, hasOrg, className, ...props }: ComponentProps<'div'> & Props) {
  const router = useRouter()
  return (
    <div className={cn('flex items-center gap-3', className)} {...props}>
      {!authenticated &&
      <>
      <Button variant="outline" className="hidden sm:inline-flex cursor-pointer" onClick={() => router.push('/login')}>
        Sign In
      </Button>
      <Button className="cursor-pointer" onClick={() => router.push('/register')}>Sign Up</Button>
      </>
      }
      {hasOrg &&
      <>
      <Button className="cursor-pointer" onClick={() => router.push('/dashboard')}>Dashboard</Button>
      </>
      }
      <Button size="icon" variant="outline">
        <SunIcon />
      </Button>

      {/* Mobile Menu */}
      <div className="md:hidden">
        <NavigationSheet />
      </div>
    </div>
  )
}