import { clsx, type ClassValue } from "clsx"
import { isFuture, toDate } from "date-fns";
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

export function isUpcoming(initialDate: string) {
  const d = toDate(initialDate)
  const nd = new Date(d.getFullYear(), d.getMonth(), d.getDate(), d.getHours(), d.getMinutes())
  
  const result = isFuture(nd)
  return result
}
