"use client";

import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious,
  type CarouselApi,
} from '@/components/ui/carousel'
import { cn } from '@/lib/utils'
import { useEffect, useState } from 'react';
import { AspectRatio } from '../ui/aspect-ratio';
import Image from 'next/image';

export default function CarouselWithPagination({ className, showControls = true }: React.ComponentProps<'div'> & { showControls?: boolean }) {
  const [api, setApi] = useState<CarouselApi>();
  const [current, setCurrent] = useState(0);
  const [count, setCount] = useState(0);

  useEffect(() => {
    if (!api) {
      return;
    }

    setCount(api.scrollSnapList().length);
    setCurrent(api.selectedScrollSnap() + 1);

    api.on("select", () => {
      setCurrent(api.selectedScrollSnap() + 1);
    });
  }, [api]);

  return (
    <div className={cn("max-w-screen-2xl", className)}>
      <Carousel setApi={setApi} className="w-full max-w-screen-2xl h-72">
        <CarouselContent>
          {Array.from({ length: 5 }).map((_, index) => (
            <CarouselItem key={index}>
              <div
                className="group relative cursor-pointer overflow-hidden rounded-lg h-72"
                onClick={() => {}}
              >
                <AspectRatio ratio={16 / 9}>
                  <Image
                    src="https://images.unsplash.com/photo-1579468118864-1b9ea3c0db4a?w=800&auto=format&fit=crop&q=60"
                    alt="Featured video thumbnail"
                    fill
                    className="transition-transform duration-300 group-hover:scale-110"
                  />
                </AspectRatio>
                <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black p-4">
                  <h3 className="text-xl font-semibold text-white">
                    Latest event
                  </h3>
                  <p className="text-sm text-gray-200">Somewhere &middot; 8PM</p>
                </div>
              </div>
            </CarouselItem>
          ))}
        </CarouselContent>
        {showControls && (
          <>
          <CarouselPrevious />
          <CarouselNext />
          </>
        )}
      </Carousel>
      <div className="mt-4 flex items-center justify-center gap-2">
        {Array.from({ length: count }).map((_, index) => (
          <button
            key={index}
            onClick={() => api?.scrollTo(index)}
            className={cn("h-3.5 w-3.5 rounded-full border-2", {
              "border-primary": current === index + 1,
            })}
          />
        ))}
      </div>
    </div>
  );
}
