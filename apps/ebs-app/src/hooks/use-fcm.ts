import { useEffect, useState } from 'react'
import useFCMToken from './use-fcm-token'
import { getMessaging, MessagePayload, onMessage } from 'firebase/messaging'
import { toast } from 'sonner'
import { app } from '@/lib/firebase'

const useFCM = () => {
  const fcmToken = useFCMToken();
  const [messages, setMessages] = useState<MessagePayload[]>([]);
  useEffect(() => {
    if (!('serviceWorker' in navigator)) {
      return
    }
    const messaging = getMessaging(app)
    const unsubscribe = onMessage(messaging, (payload) => {
      toast(payload.notification?.title, {
        description: payload.notification?.body,
      })
      setMessages((messages) => [...messages, payload]);
    });
    return () => unsubscribe()
  }, [])
  return { fcmToken, messages }
};

export default useFCM;