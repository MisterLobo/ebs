"use client";

import * as React from "react";
import { Play } from "lucide-react";
import Image from "next/image";

import { AspectRatio } from "@/components/ui/aspect-ratio";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

const videos: any[] = [
  {
    id: "1",
    title: "Introduction to React",
    duration: "12:34",
    thumbnail: "https://files.vidstack.io/sprite-fight/poster.webp",
    videoUrl: "https://files.vidstack.io/sprite-fight/720p.mp4",
  },
  {
    id: "2",
    title: "Advanced CSS Techniques",
    duration: "18:22",
    thumbnail: "https://images.unsplash.com/photo-1507721999472-8ed4421c4af2?w=600&h=400&fit=crop&q=80",
    videoUrl: "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ElephantsDream.mp4",
  },
  {
    id: "3",
    title: "JavaScript ES6 Features",
    duration: "15:45",
    thumbnail: "https://images.unsplash.com/photo-1579468118864-1b9ea3c0db4a?w=600&h=400&fit=crop&q=80",
    videoUrl: "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4",
  },
  {
    id: "4",
    title: "Building RESTful APIs",
    duration: "20:10",
    thumbnail: "https://images.unsplash.com/photo-1516259762381-22954d7d3ad2?w=600&h=400&fit=crop&q=80",
    videoUrl: "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerEscapes.mp4",
  },
  {
    id: "5",
    title: "Responsive Web Design",
    duration: "14:55",
    thumbnail: "https://images.unsplash.com/photo-1517292987719-0369a794ec0f?w=600&h=400&fit=crop&q=80",
    videoUrl: "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerFun.mp4",
  },
  {
    id: "6",
    title: "Vue.js for Beginners",
    duration: "16:40",
    thumbnail: "https://images.unsplash.com/photo-1614741118887-7a4ee193a5fa?w=600&h=400&fit=crop&q=80",
    videoUrl: "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerJoyrides.mp4",
  },
  /* {
    id: "7",
    title: "Building RESTful APIs",
    duration: "20:10",
    thumbnail:
      "https://images.unsplash.com/photo-1516259762381-22954d7d3ad2?w=600&h=400&fit=crop&q=80",
    videoUrl:
      "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerEscapes.mp4",
  },
  {
    id: "8",
    title: "Responsive Web Design",
    duration: "14:55",
    thumbnail:
      "https://images.unsplash.com/photo-1517292987719-0369a794ec0f?w=600&h=400&fit=crop&q=80",
    videoUrl:
      "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerFun.mp4",
  },
  {
    id: "9",
    title: "Vue.js for Beginners",
    duration: "16:40",
    thumbnail:
      "https://images.unsplash.com/photo-1614741118887-7a4ee193a5fa?w=600&h=400&fit=crop&q=80",
    videoUrl:
      "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerJoyrides.mp4",
  }, */
];

export default function VideoGalleryWithPlayback({ headerText }: { headerText: string }) {

  const closeVideo = () => {
  };

  return (
    <div className="container mx-auto p-4 py-32">
      <h1 className="mb-8 text-4xl font-bold">{ headerText }</h1>
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
        {videos.map((video) => (
          <div
            key={video.id}
            className="group relative overflow-hidden rounded-lg cursor-pointer"
            onClick={() => {}}
          >
            <AspectRatio ratio={16 / 9}>
              <Image
                src={video.thumbnail}
                alt={video.title}
                fill
                objectFit="cover"
                className="transition-transform duration-300 group-hover:scale-110"
              />
            </AspectRatio>
            <div className="absolute inset-0 bg-gradient-to-t from-black/60 to-transparent transition-opacity duration-300 group-hover:opacity-100">
              <div className="absolute bottom-0 left-0 right-0 p-4">
                <h2 className="mb-1 line-clamp-2 text-xl font-semibold text-white">
                  {video.title}
                </h2>
                <p className="text-sm text-gray-300">{video.duration}</p>
              </div>
            </div>
            <div className="absolute inset-0 flex items-center justify-center overflow-hidden rounded-lg bg-transparent bg-opacity-0 transition-opacity duration-200 group-hover:bg-opacity-40">
              <Play
                className="size-12 text-white opacity-0 transition-opacity duration-200 group-hover:opacity-100"
                aria-hidden="true"
              />
            </div>
            <div className="absolute right-2 top-2 rounded bg-black/60 px-2 py-1 text-xs font-medium text-white">
              {video.duration}
            </div>
          </div>
        ))}
      </div>
      <div className="flex w-full justify-center py-5">
        <Button className="bg-background hover:bg-background text-foreground hover:text-foreground border-foreground hover:border-1 rounded cursor-pointer">
          Show all
        </Button>
      </div>

      <Dialog open={false} onOpenChange={closeVideo}>
        <DialogContent className="sm:max-w-[800px]">
          <DialogHeader>
            <DialogTitle>Title</DialogTitle>
          </DialogHeader>
          
        </DialogContent>
      </Dialog>
    </div>
  );
}