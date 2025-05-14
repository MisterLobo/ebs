import {
  NavigationMenu,
  NavigationMenuItem,
  NavigationMenuLink,
  NavigationMenuList,
} from "@/components/ui/navigation-menu";
import { NavigationMenuProps } from "@radix-ui/react-navigation-menu";
import Link from "next/link";

type Props = {
  isOrgOWner?: boolean,
}
export const NavMenu = ({ isOrgOWner, ...props }: NavigationMenuProps & Props) => (
  <NavigationMenu {...props}>
    <NavigationMenuList className="gap-6 space-x-0 data-[orientation=vertical]:flex-col data-[orientation=vertical]:items-start">
      <NavigationMenuItem>
        <NavigationMenuLink asChild>
          <Link href="/personal/dashboard">Home</Link>
        </NavigationMenuLink>
      </NavigationMenuItem>
      <NavigationMenuItem>
        <NavigationMenuLink asChild>
          <Link href="/trending">Trending</Link>
        </NavigationMenuLink>
      </NavigationMenuItem>
      <NavigationMenuItem>
        <NavigationMenuLink asChild>
          <Link href="/personal/waitlist">Waitlist</Link>
        </NavigationMenuLink>
      </NavigationMenuItem>
      <NavigationMenuItem>
        <NavigationMenuLink asChild>
          <Link href="/events">Browse Events</Link>
        </NavigationMenuLink>
      </NavigationMenuItem>
      {isOrgOWner && <NavigationMenuItem>
        <NavigationMenuLink asChild>
          <Link href="/dashboard/events/new">Host an Event</Link>
        </NavigationMenuLink>
      </NavigationMenuItem>}
    </NavigationMenuList>
  </NavigationMenu>
);
