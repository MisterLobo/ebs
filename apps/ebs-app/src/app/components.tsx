'use client'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { Calendar, MapPin } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useCallback } from 'react'

type Props = {
  id: number,
  name: string,
}
export function EventCategoryCard({ id, name }: Props) {
  const router = useRouter()
  const categoryOnClick = useCallback(() => {
    router.push(`/events?q=${encodeURIComponent(name)}`)
  }, [])
  return (
    <Card className="w-full max-w-3xl h-32 col-span-3 cursor-pointer" onClick={categoryOnClick}>
      <CardContent>
        { name }
      </CardContent>
    </Card>
  )
}

type EventCategoriesProps = {
  categories: string[]
}
export function EventCategories({ categories }: EventCategoriesProps) {
  return (
    <div className="grid grid-cols-12 gap-4 my-5 w-full">
      {categories.slice(0, 8).map((c, i) => (
        <EventCategoryCard key={i} id={i} name={c} />
      ))}
      <div className="col-span-12">
        <Button variant="link" className="text-xs m-0">All categories</Button>
      </div>
    </div>
  )
}

type CategoryEventCardProps = {
  id?: number,
}
export function CategoryEventCard({}: CategoryEventCardProps) {
  return (
    <Card className="w-full max-w-3xl h-64 col-span-3 relative">
      <CardHeader>
        <CardTitle className="text-2xl line-clamp-3 break-all">
          Event Name and Event Name and Event Name and Event Name Very Long Event Name
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col justify-center bottom-0 absolute my-5 w-full gap-1">
        <h4 className="truncate text-ellipsis text-sm">Starts at $1</h4>
        <h4 className="truncate text-ellipsis">Organizer name</h4>
        <Separator />
        <h4 className="flex items-center justify-center gap-2">
          <span className="inline-flex items-center justify-center"><MapPin className="size-6" /></span>
          <span className="truncate text-ellipsis">location, somewhere around the world and here and there</span>
        </h4>
        <h4 className="flex items-center gap-2">
          <div className="inline-flex"><Calendar className="size-6" /></div>
          <span className="truncate text-ellipsis">July 16th, 4PM PST</span>
        </h4>
      </CardContent>
    </Card>
  )
}

type CategoryCardProps = {
  title: string,
  data?: Record<string, any>[],
}
export function CategoryCard({ title, data }: CategoryCardProps) {
  const router = useRouter()
  const seeAll = () => {
    router.push(`/events?q=${encodeURIComponent(title)}`)
  }
  return (
    <>
    <div className="flex w-full items-center justify-between mt-10">
      <h1 className="text-2xl">
        {title}
      </h1>
      <Button variant="link" onClick={seeAll}>See all</Button>
    </div>
    <div className="flex w-full gap-4">
      {data?.map((_, i) => (
        <CategoryEventCard key={i} id={i} />
      ))}
    </div>
    </>
  )
}