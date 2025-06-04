import { Loader2 } from 'lucide-react'

export default async function Loading() {
  return (
    <div className="flex items-center justify-center w-full h-screen">
      <Loader2 className="animate-spin size-48" />
    </div>
  )
}