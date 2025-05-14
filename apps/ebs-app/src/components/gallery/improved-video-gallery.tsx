"use client";

import React, { useState } from "react";
import { Grid, List } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function ImprovedVideoGallery({ heading }: { heading?: string }) {
  const [layout, setLayout] = useState<"grid" | "list">("list");

  return (
    <div className="container mx-auto p-4 py-16">
      <div className="mb-6 flex items-center justify-between">
        {heading && <h2 className="text-3xl font-bold">{heading}</h2>}
        <div className="flex gap-2">
          <Button
            variant={layout === "grid" ? "default" : "ghost"}
            size="icon"
            onClick={() => setLayout("grid")}
          >
            <Grid className="size-4" />
          </Button>
          <Button
            variant={layout === "list" ? "default" : "ghost"}
            size="icon"
            onClick={() => setLayout("list")}
          >
            <List className="size-4" />
          </Button>
        </div>
      </div>

      {layout === "grid" ? (
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          
        </div>
      ) : (
        <div className="space-y-2">
          
        </div>
      )}

      <Dialog
        onOpenChange={() => {}}
      >
        <DialogContent className="w-full max-w-7xl">
          <DialogHeader>
            <DialogTitle>Title</DialogTitle>
          </DialogHeader>
          
        </DialogContent>
      </Dialog>
    </div>
  );
}