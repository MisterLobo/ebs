'use client'

import useFCM from '@/hooks/use-fcm'
import { app } from '@/lib/firebase'
import { getMessaging, onMessage } from 'firebase/messaging'
import { useRouter } from 'next/navigation'
import { useEffect } from 'react'
import { toast } from 'sonner'

type Props = {
  tokenRetrieved?: (token: string) => void,
}
export default function FCM({ tokenRetrieved }: Props) {
  const token = useFCM()
  const router = useRouter()
  useEffect(() => {
    const messaging = getMessaging(app)
    const unsub = onMessage(messaging, (payload) => {
      console.log('received payload:', payload)
      toast('MESSAGE', {
        description: 'payload received',
      })
      router.refresh()
    })
    if (token.fcmToken) {
      if (tokenRetrieved) {
        tokenRetrieved(token.fcmToken)
      }
    }
    return () => {
      unsub()
    }
  }, [token])
  return <></>
}