import { Logo } from './logo'
import { NavMenu } from './nav-menu'
import NavActions from './nav-actions'
import { getActiveOrganization, isAuthenticated } from '@/lib/actions'
import Link from 'next/link'

const Nav = async () => {
  const authenticated = await isAuthenticated()
  const org = await getActiveOrganization()
  const hasOrg = !!org?.id && org.type === 'standard'

  return (
    <nav className="h-16 bg-background border-b">
      <div className="h-full flex items-center justify-between mx-auto sm:px-6">
        <div className="flex items-center gap-8">
          <Link href="/"><Logo /></Link>

          {/* Desktop Menu */}
          <NavMenu className="hidden md:block" isOrgOWner={hasOrg} />
        </div>

        <NavActions authenticated={authenticated} hasOrg={hasOrg} />
      </div>
    </nav>
  )
}

export default Nav;
