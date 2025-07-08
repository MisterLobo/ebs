import CarouselWithPagination from '@/components/carousel/carousel-with-pagination'
import Nav from '@/components/nav/nav'
import { CategoryCard, EventCategories } from './components'
import { categories } from '@/lib/constants'
import FCM from '@/components/fcm'
import { tokenRetrieved } from '@/lib/actions'
import { WebWorker } from '@/components/worker'

export default async function Index() {
  return (
    <div className="container mx-auto">
      <Nav />
      <div className="flex flex-col items-center my-10 gap-4">
        <div className="flex w-full gap-4 my-10">
          <CarouselWithPagination className="w-full h-72" />
        </div>
        <EventCategories categories={categories} />
        <div className="flex items-center justify-start w-full">
          <h1 className="text-3xl text-start uppercase">Events near you</h1>
        </div>
        <CategoryCard title="Conferences & Summits" data={new Array(4).fill(0)} />
        <CategoryCard title="Fun Runs / Charity Walks" data={new Array(4).fill(0)} />
        <CategoryCard title="Live Concerts / DJ Sets" data={new Array(4).fill(0)} />
        <CategoryCard title="Barista or Coffee Brewing Workshops" data={new Array(4).fill(0)} />
        <CategoryCard title="Comedy Shows" data={new Array(4).fill(0)} />
      </div>
      <FCM tokenRetrieved={tokenRetrieved} />
      <WebWorker />
    </div>
  )
}
