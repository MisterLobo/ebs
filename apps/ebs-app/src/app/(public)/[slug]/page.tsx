import { importURLPatternPolyfill } from "@/lib/utils"
import { headers } from "next/headers"
import { redirect } from "next/navigation"

export default async function EventSlugPage() {
  await importURLPatternPolyfill()
  const pattern = new URLPattern({ pathname: '/events/:slug' })
  const $headers = await headers()
  const url = $headers.get('x-url')
  const result = pattern.exec(url as string)
  const slug = result?.pathname.groups.slug
  if (!slug) {
    redirect('/')
  }
  return (
    <p>Event slug page</p>
  )
}