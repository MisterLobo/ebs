'use client'

import { Card, CardContent } from '@/components/ui/card'
import { useRouter } from 'next/navigation'
import { useCallback } from 'react'

type Props = {
  id: number,
  name: string,
}
export function EventCategoryCard({ id, name }: Props) {
  const router = useRouter()
  const categoryOnClick = useCallback(() => {
    router.push(`/events?q=${encodeURIComponent(name)}`)
  }, [])
  return (
    <Card className="w-full max-w-3xl h-32 col-span-3 cursor-pointer" onClick={categoryOnClick}>
      <CardContent>
        { name }
      </CardContent>
    </Card>
  )
}