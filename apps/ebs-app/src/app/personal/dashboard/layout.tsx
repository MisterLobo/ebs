import Nav from '@/components/nav/nav'
import { ReactNode } from 'react'

export default async function PersonalDashboardLayout({ children }: { children: ReactNode }) {
  return (
    <div className="container">
      <Nav />
    
      { children }
    </div>
  )
}