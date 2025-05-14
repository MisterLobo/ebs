import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export async function importURLPatternPolyfill() {
  // Conditional ESM module loading (Node.js and browser)
  // @ts-expect-error: Property 'UrlPattern' does not exist 
  if (!globalThis.URLPattern) { 
    await import("urlpattern-polyfill");
  }
}
