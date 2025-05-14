import { ComponentProps } from 'react'
import CarouselSimple from './carousel-simple'
import CarouselVertical from './carousel-vertical'
import CarouselWithFooter from './carousel-with-footer'
import CarouselWithMultipleSlides from './carousel-with-multiple-slides'
import CarouselWithPagination from './carousel-with-pagination'
import CarouselWithProgress from './carousel-with-progress'
import CarouselWithSlideOpacity from './carousel-with-slide-opacity'
import CarouselWithSlideStatus from './carousel-with-slide-status'
import CarouselWithSlideScale from './carousel-with-scale'

export type carouselVariants = 'default' | 'vertical' | 'withFooter' | 'multiSlide' | 'withPagination' | 'withProgress' | 'withSlideOpacity' | 'withSlideStatus' | 'withSlideScale'

function Carousel({
  className,
  variant,
}: ComponentProps<'div'> & { variant: carouselVariants } & {
  asChild: boolean,
}) {

  switch (variant) {
    case 'default': {
      return <CarouselSimple className={className} />
    }
    case 'vertical': {
      return <CarouselVertical className={className} />
    }
    case 'withFooter': {
      return <CarouselWithFooter className={className} />
    }
    case 'multiSlide': {
      return <CarouselWithMultipleSlides username="" channel="" className={className} />
    }
    case 'withPagination': {
      return <CarouselWithPagination className={className} />
    }
    case 'withProgress': {
      return <CarouselWithProgress className={className} />
    }
    case 'withSlideOpacity': {
      return <CarouselWithSlideOpacity className={className} />
    }
    case 'withSlideStatus': {
      return <CarouselWithSlideStatus className={className} />
    }
    case 'withSlideScale': {
      return <CarouselWithSlideScale className={className} />
    }
  }
}

export { Carousel }