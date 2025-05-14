"use client";

import { useState } from "react";
import Image from "next/image";
import { Play } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { AspectRatio } from "../ui/aspect-ratio";

export default function FeaturedVideoGallery() {
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  return (
    <section className="container mx-auto py-8">
      <h2 className="mb-6 text-2xl font-bold">
        Featured <span className="text-primary">Videos</span>
      </h2>
      <div className="space-y-8">
        <div
          className="group relative aspect-video cursor-pointer overflow-hidden rounded-lg"
          onClick={() => {}}
        >
          <AspectRatio ratio={16 / 9}>
            <Image
              src="https://files.vidstack.io/sprite-fight/poster.webp"
              alt="Featured video thumbnail"
              fill
              sizes="100vw"
              objectFit="cover"
              className="transition-transform duration-300 group-hover:scale-110"
            />
          </AspectRatio>
          {/* <Player src="https://files.vidstack.io/sprite-fight/720p.mp4" muted autoPlay /> */}
          <div className="absolute inset-0 flex items-center justify-center overflow-hidden rounded-lg bg-opacity-0 bg-transparent transition-opacity duration-200 group-hover:bg-opacity-40">
            <Play 
              className="size-12 text-white opacity-0 transition-opacity duration-200 group-hover:opacity-100"
              aria-hidden="true"
            />
          </div>
          <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black p-4">
            <h3 className="text-xl font-semibold text-white">
              Master Class: Full Stack Development
            </h3>
            <p className="text-sm text-gray-200">45:30 • 25K views</p>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          
        </div>
      </div>

      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-[800px]">
          <DialogHeader>
            <DialogTitle>Title</DialogTitle>
          </DialogHeader>
          <div className="aspect-video">
            
          </div>
        </DialogContent>
      </Dialog>
    </section>
  );
}