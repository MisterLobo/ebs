import CarouselWithPagination from '@/components/carousel/carousel-with-pagination'
import Nav from '@/components/nav/nav'

export default function Index() {
  return (
    <div className="min-h-screen container">
      <Nav />
      <main className="px-4">
        <CarouselWithPagination />
      </main>
    </div>
  )
}
