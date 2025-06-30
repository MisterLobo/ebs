import CarouselWithPagination from '@/components/carousel/carousel-with-pagination'
import Nav from '@/components/nav/nav'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { Calendar, MapPin } from 'lucide-react'
import { EventCategoryCard } from './components'
import { Button } from '@/components/ui/button'
import { categories } from '@/lib/constants'

export default async function Index() {
  return (
    <div className="container mx-auto">
      <Nav />
      <div className="flex flex-col items-center my-10 gap-4">
        <div className="flex w-full gap-4 my-10">
          <CarouselWithPagination className="w-full h-72" />
        </div>
        <div className="grid grid-cols-12 gap-4 my-5 w-full">
          {categories.slice(0, 8).map((c, i) => (
            <EventCategoryCard key={i} id={i} name={c} />
          ))}
          <div className="col-span-12">
            <Button variant="link" className="text-xs m-0">All categories</Button>
          </div>
        </div>
        <div className="flex items-center justify-start w-full">
          <h1 className="text-3xl text-start uppercase">Events near you</h1>
        </div>
        <div className="flex w-full items-center justify-between mt-10">
          <h2 className="text-2xl">
            Conferences &amp; Summits
          </h2>
          <Button variant="link">See all</Button>
        </div>
        <div className="grid grid-cols-12 gap-4 w-full">
          {new Array(4).fill(0).map((_, i) => (
            <Card key={i} className="w-full max-w-3xl h-64 col-span-3 relative">
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
          ))}
        </div>
        <div className="flex w-full items-center justify-between mt-10">
          <h1 className="text-2xl">
            Fun Runs / Charity Walks
          </h1>
          <Button variant="link">See all</Button>
        </div>
        <div className="flex w-full gap-4">
          {new Array(4).fill(0).map((_, i) => (
            <Card key={i} className="w-full max-w-3xl h-64 col-span-3 relative">
              <CardHeader>
                <CardTitle className="text-2xl line-clamp-3 break-all">
                  Event Name and Event Name and Event Name
                </CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col justify-center bottom-0 absolute my-5 w-full gap-1">
                <h4 className="truncate text-ellipsis text-sm">Organizer name</h4>
                <h4 className="truncate text-ellipsis">Organizer name</h4>
                <Separator />
                <h4 className="flex items-center justify-center gap-2">
                  <span className="inline-flex items-center justify-center"><MapPin className="size-6" /></span>
                  <span className="truncate text-ellipsis">location, somewhere around the world and here and there</span>
                </h4>
                <h4 className="flex items-center gap-2">
                  <div className="inline-flex"><Calendar className="size-6" /></div>
                  <span className="truncate text-ellipsis">July 16th, 4PM</span>
                </h4>
              </CardContent>
            </Card>
          ))}
        </div>
        <div className="flex w-full items-center justify-between mt-10">
          <h1 className="text-2xl">
            Live Concerts / DJ Sets
          </h1>
          <Button variant="link">See all</Button>
        </div>
        <div className="flex w-full gap-4">
          {new Array(4).fill(0).map((_, i) => (
            <Card key={i} className="w-full max-w-3xl h-64 col-span-3 relative">
              <CardHeader>
                <CardTitle className="text-2xl line-clamp-3 break-all">
                  Event Name and Event Name and Event Name
                </CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col justify-center bottom-0 absolute my-5 w-full gap-1">
                <h4 className="truncate text-ellipsis">Organizer name</h4>
                <Separator />
                <h4 className="flex items-center justify-center gap-2">
                  <span className="inline-flex items-center justify-center"><MapPin className="size-6" /></span>
                  <span className="truncate text-ellipsis">location, somewhere around the world and here and there</span>
                </h4>
                <h4 className="flex items-center gap-2">
                  <div className="inline-flex"><Calendar className="size-6" /></div>
                  <span className="truncate text-ellipsis">July 16th, 4PM</span>
                </h4>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </div>
  )
}
