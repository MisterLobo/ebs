import Nav from '@/components/nav/nav'
import { ReactNode } from 'react'

export default async function WaitlistLayout({ children }: { children: ReactNode }) {
  return (
    <div className="container">
      <Nav />
      { children }
    </div>
  )
}