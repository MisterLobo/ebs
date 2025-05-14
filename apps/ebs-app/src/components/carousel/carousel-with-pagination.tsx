"use client";

import * as React from "react";
import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious,
  type CarouselApi,
} from "@/components/ui/carousel";
import { cn } from "@/lib/utils";
import Image from "next/image";
import { Play } from "lucide-react";
import { AspectRatio } from "@/components/ui/aspect-ratio";

export default function CarouselWithPagination({ className, showControls = true }: React.ComponentProps<'div'> & { showControls?: boolean }) {
  const [api, setApi] = React.useState<CarouselApi>();
  const [current, setCurrent] = React.useState(0);
  const [count, setCount] = React.useState(0);

  React.useEffect(() => {
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
      <Carousel setApi={setApi} className="w-full max-w-screen-2xl">
        <CarouselContent>
          {Array.from({ length: 5 }).map((_, index) => (
            <CarouselItem key={index}>
              <div
                className="group relative aspect-video cursor-pointer overflow-hidden rounded-lg"
                onClick={() => {}}
              >
                <AspectRatio ratio={16 / 9}>
                  <Image
                    src="https://images.unsplash.com/photo-1579468118864-1b9ea3c0db4a?w=800&auto=format&fit=crop&q=60"
                    alt="Featured video thumbnail"
                    fill
                    objectFit="cover"
                    className="transition-transform duration-300 group-hover:scale-110"
                  />
                </AspectRatio>
                <div className="absolute inset-0 flex items-center justify-center bg-transparent bg-opacity-40">
                  <Play className="size-16 text-white" aria-hidden="true" />
                </div>
                <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black p-4">
                  <h3 className="text-xl font-semibold text-white">
                    Master Class: Full Stack Development
                  </h3>
                  <p className="text-sm text-gray-200">45:30 • 25K views</p>
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
