"use client"; // Required for framer-motion

import { Button } from "@/components/ui/button";
import Link from "next/link";
import { motion } from "framer-motion";

export default function NotFound() {
  return (
    <div className="relative flex min-h-screen w-full items-center justify-center overflow-hidden bg-background py-16">
      {/* Animated background shapes */}
      <div className="absolute inset-0 h-full w-full">
        {Array.from({ length: 6 }).map((_, i) => (
          <motion.div
            key={i}
            className="absolute h-64 w-64 rounded-full bg-primary/5"
            initial={{
              scale: 0,
              x: Math.random() * 100 - 50,
              y: Math.random() * 100 - 50,
            }}
            animate={{
              scale: [1, 1.2, 1],
              x: [null, Math.random() * 200 - 100],
              y: [null, Math.random() * 200 - 100],
            }}
            transition={{
              duration: 10,
              repeat: Infinity,
              repeatType: "reverse",
              ease: "easeInOut",
              delay: i * 0.5,
            }}
          />
        ))}
      </div>

      <div className="relative z-10 space-y-8 px-4 text-center">
        <motion.h1
          className="text-9xl font-extrabold text-foreground"
          initial={{ y: -50, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          transition={{ duration: 0.5 }}
        >
          4<span className="text-primary">0</span>4
        </motion.h1>
        <motion.h2
          className="text-2xl font-medium text-muted-foreground"
          initial={{ y: 50, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.2 }}
        >
          Oops! Page Not Found
        </motion.h2>
        <motion.div
          className="flex items-center justify-center gap-4 pt-6"
          initial={{ y: 50, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.4 }}
        >
          <Button variant="outline" size="lg" className="font-medium" asChild>
            <Link href="/contact">Contact Us</Link>
          </Button>
          <Button
            size="lg"
            className="bg-primary font-medium text-primary-foreground hover:bg-primary/90"
            asChild
          >
            <Link href="/">Go Home</Link>
          </Button>
        </motion.div>
      </div>

      {/* Decorative elements */}
      <div className="absolute bottom-10 left-10 size-20 rounded-full border-4 border-primary/20" />
      <div className="absolute right-10 top-10 size-16 rotate-45 border-4 border-primary/20" />
      <div className="absolute bottom-1/4 right-1/4 size-24 rounded-full border-4 border-dashed border-primary/20" />
    </div>
  );
}