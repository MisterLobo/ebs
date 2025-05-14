"use client";

import * as React from "react";
import Image from "next/image";
import { Play, Pause, X } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

const videoItems = [
  {
    id: 1,
    title: "Deep Sea Wonders",
    description: "Explore the mysteries of the ocean",
    thumbnail:
      "https://images.unsplash.com/reserve/LJIZlzHgQ7WPSh5KVTCB_Typewriter.jpg?q=80&w=1596&auto=format&fit=crop&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%3D%3D",
    videoUrl:
      "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4",
    aspectRatio: "portrait",
  },
  {
    id: 2,
    title: "Urban Landscapes",
    description: "The evolution of modern cities",
    thumbnail:
      "https://images.unsplash.com/photo-1514924013411-cbf25faa35bb?w=800&auto=format&fit=crop&q=60&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxzZWFyY2h8M3x8Y2l0eSUyMGFyY2hpdGVjdHVyZXxlbnwwfHwwfHx8MA%3D%3D",
    videoUrl:
      "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ElephantsDream.mp4",
    aspectRatio: "landscape",
  },
  {
    id: 3,
    title: "Culinary Journey",
    description: "Exploring global cuisines",
    thumbnail:
      "https://images.unsplash.com/photo-1556910103-1c02745aae4d?w=800&auto=format&fit=crop&q=60&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxzZWFyY2h8M3x8Y3VsaW5hcnl8ZW58MHx8MHx8fDA%3D",
    videoUrl:
      "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4",
    aspectRatio: "portrait",
  },
  {
    id: 4,
    title: "Extreme Adventures",
    description: "Pushing the limits of human endurance",
    thumbnail:
      "https://images.unsplash.com/photo-1461280360983-bd93eaa5051b?w=800&auto=format&fit=crop&q=60&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxzZWFyY2h8M3x8ZXh0cmVtZSUyMHNwb3J0c3xlbnwwfHwwfHx8MA%3D%3D",
    videoUrl:
      "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerEscapes.mp4",
    aspectRatio: "landscape",
  },
  {
    id: 5,
    title: "Mountain Expeditions",
    description: "Conquering the world's highest peaks",
    thumbnail:
      "https://images.unsplash.com/photo-1464822759023-fed622ff2c3b?w=800&auto=format&fit=crop&q=60&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxzZWFyY2h8M3x8bW91bnRhaW58ZW58MHx8MHx8fDA%3D",
    videoUrl:
      "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerFun.mp4",
    aspectRatio: "landscape",
  },
];

export default function MixedSizeVideoGallery() {
  const [selectedVideo, setSelectedVideo] = React.useState<number | null>(null);
  const [isPlaying, setIsPlaying] = React.useState(false);
  const videoRef = React.useRef<HTMLVideoElement>(null);

  const handleVideoClick = (index: number) => {
    setSelectedVideo(index);
    setIsPlaying(true);
  };

  const handleClose = () => {
    setSelectedVideo(null);
    setIsPlaying(false);
  };

  const togglePlayPause = async () => {
    if (videoRef.current) {
      if (isPlaying) {
        videoRef.current.pause();
      } else {
        await videoRef.current.play();
      }
      setIsPlaying(!isPlaying);
    }
  };

  return (
    <section className="w-full bg-background py-12 md:py-24 lg:py-32">
      <div className="container mx-auto md:px-6">
        <div className="mb-12 flex flex-col items-center justify-center space-y-4 text-center">
          <h2 className="text-3xl font-bold tracking-tighter sm:text-5xl">
            Mixed Size <span className="text-primary">Video Gallery</span>
          </h2>
          <p className="max-w-[700px] text-gray-500 dark:text-gray-400 md:text-xl/relaxed lg:text-base/relaxed xl:text-xl/relaxed">
            Discover our diverse collection of videos in an engaging masonry
            layout.
          </p>
        </div>
        <div className="grid auto-rows-[200px] grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
          {videoItems.map((item, index) => (
            <div
              key={item.id}
              className={cn(
                "group relative cursor-pointer overflow-hidden rounded-lg",
                item.aspectRatio === "portrait" ? "row-span-2" : "",
                index === 0 ? "md:col-span-2" : "",
              )}
              onClick={() => handleVideoClick(index)}
            >
              <Image
                src={item.thumbnail}
                alt={item.title}
                layout="fill"
                objectFit="cover"
                className="transition-transform duration-300 group-hover:scale-105"
              />
              <div className="absolute inset-0 flex flex-col items-center justify-center bg-black bg-opacity-50 p-4 opacity-100 transition-opacity duration-300">
                <Play className="mb-4 size-12 text-white" />
                <h3 className="mb-2 text-center text-xl font-bold text-white">
                  {item.title}
                </h3>
                <p className="text-center text-sm text-white">
                  {item.description}
                </p>
              </div>
            </div>
          ))}
        </div>
      </div>
      {selectedVideo !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-90">
          <div className="relative aspect-video w-full max-w-4xl">
            <video
              ref={videoRef}
              src={videoItems[selectedVideo]?.videoUrl ?? ""}
              className="h-full w-full"
              autoPlay
              controls
            />
            <Button
              variant="ghost"
              size="icon"
              className="absolute right-4 top-4 text-white"
              onClick={handleClose}
            >
              <X className="size-6" />
              <span className="sr-only">Close</span>
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="absolute bottom-4 left-4 text-white"
              onClick={togglePlayPause}
            >
              {isPlaying ? (
                <Pause className="size-6" />
              ) : (
                <Play className="size-6" />
              )}
              <span className="sr-only">{isPlaying ? "Pause" : "Play"}</span>
            </Button>
          </div>
        </div>
      )}
    </section>
  );
}