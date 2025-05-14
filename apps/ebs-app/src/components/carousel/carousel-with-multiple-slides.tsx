import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious,
} from "@/components/ui/carousel";
import { cn } from "@/lib/utils";
import React from "react";
import { AspectRatio } from "../ui/aspect-ratio";
import Image from "next/image";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";
import Link from "next/link";

export default function CarouselWithMultipleSlides({
  className,
  headerText,
  title,
  age,
  channel,
  username,
  avatar,
  avatarAlt = 'ML',
}: React.ComponentProps<'div'> & {
  headerText?: string,
  channel: string,
  username: string,
  title?: string,
  age?: string,
  avatar?: string,
  avatarAlt?: string,
}) {
  return (
    <div className="min-w-5xl">
      {headerText && <h1 className="mt-8 text-4xl font-bold">{ headerText }</h1>}
      <Carousel
        opts={{
          align: "start",
        }}
        className={cn("w-full max-w-screen min-w-96", className)}
      >
        <CarouselContent className="px-4">
          {Array.from({ length: 10 }).map((_, index) => (
            <CarouselItem key={index} className="basis-1/4 p-0 ">
              <div className="p-1 space-y-1">
                <AspectRatio ratio={16 / 9}>
                  <Image
                    src="https://files.vidstack.io/sprite-fight/poster.webp"
                    alt="Featured video thumbnail"
                    fill
                    objectFit="cover"
                    className="transition-transform duration-300 group-hover:scale-110 rounded"
                  />
                </AspectRatio>
                <div className="flex gap-3">
                  <Link href={`/${channel}`} className="pt-1">
                    <Avatar>
                      <AvatarImage src={avatar} alt={channel} />
                      <AvatarFallback>{avatarAlt}</AvatarFallback>
                    </Avatar>
                  </Link>
                  <div className="flex flex-col">
                    <Link href="/watch/learnwithme-3949f88" className="group space-y-2">
                      <span className="font-semibold tracking-tight text-lg text-neutral-100 line-clamp-2 text-ellipsis">
                        {title}
                      </span>
                    </Link>
                    <span className="leading-none text-md text-muted-foreground">
                      <Link className="font-semibold tracking-tight" href={`/${channel}`}>{username}</Link> &middot; {age}
                    </span>
                  </div>
                </div>
              </div>
            </CarouselItem>
          ))}
        </CarouselContent>
        <CarouselPrevious className="cursor-pointer" />
        <CarouselNext className="cursor-pointer bg-background" />
      </Carousel>
    </div>
  );
}
