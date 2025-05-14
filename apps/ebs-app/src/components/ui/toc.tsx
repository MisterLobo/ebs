import * as React from "react"

import { cn } from "@/lib/utils"
import Link from "next/link"

interface TableOfContentsListProps
  extends React.ComponentPropsWithoutRef<"ul"> {
  indent?: boolean
}

const TableOfContentsList = React.forwardRef<
   React.ComponentRef<"ul">,
   TableOfContentsListProps
>(({ className, indent, ...props }, ref) => (
  <ul
    ref={ref}
    className={cn("m-0 list-none", indent && "pl-4", className)}
    {...props}
  />
))
TableOfContentsList.displayName = "TableOfContentsList"

type TableOfContentsTitleProps = React.ComponentPropsWithoutRef<"li">

const TableOfContentsTitle = React.forwardRef<
   React.ComponentRef<"li">,
   TableOfContentsTitleProps
>(({ className, ...props }, ref) => (
  <li
    ref={ref}
    className={cn("mb-2 text-sm font-medium", className)}
    {...props}
  />
))
TableOfContentsTitle.displayName = "TableOfContentsTitle"

interface TableOfContentsItemProps
  extends React.ComponentPropsWithoutRef<"li"> {
  indent?: boolean
}

const TableOfContentsItem = React.forwardRef<
   React.ComponentRef<"li">,
   TableOfContentsItemProps
>(({ className, indent, ...props }, ref) => (
  <li
    ref={ref}
    className={cn("mt-0 pt-2", indent && "ml-4", className)}
    {...props}
  />
))
TableOfContentsItem.displayName = "TableOfContentsItem"

interface TableOfContentsLinkProps extends React.ComponentPropsWithoutRef<"a"> {
  isActive?: boolean
}

const TableOfContentsLink = React.forwardRef<
   React.ComponentRef<"a">,
   TableOfContentsLinkProps
>(({ className, isActive, href = '#', ...props }, ref) => (
  <Link
    ref={ref}
    href={href}
    className={cn(
        "text-foreground hover:text-primary text-md font-medium transition-colors",
        isActive ? "text-foreground font-medium" : "text-muted-foreground",
        className,
    )}
    {...props}
  />
))
TableOfContentsLink.displayName = "TableOfContentsLink"

export {
  TableOfContentsList,
  TableOfContentsTitle,
  TableOfContentsItem,
  TableOfContentsLink,
}
