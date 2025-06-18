import { useEffect, useState } from 'react'
import { getMessaging, getToken, isSupported } from 'firebase/messaging'
import useNotificationPermission from './use-notification'
import { app } from '@/lib/firebase';

const useFCMToken = () => {
  const permission = useNotificationPermission();
  const [fcmToken, setFcmToken] = useState<string | null>(null);

  useEffect(() => {
    if (!navigator) {
      return
    }
    const retrieveToken = async () => {
      try {
        if (typeof window !== 'undefined' && 'serviceWorker' in navigator) {
          if (permission === 'granted') {
            const isFCMSupported = await isSupported()
            if (!isFCMSupported) return
            if (permission === 'granted') {
              const messaging = getMessaging(app)
              const fcmToken = await getToken(messaging, { vapidKey: process.env.NEXT_PUBLIC_FIREBASE_VAPID })
              setFcmToken(fcmToken)
            }
          }
        } else {
          console.error('could not get reference to window and serviceWorker')
        }
      } catch (error) {
        console.error('An error occurred while retrieving token:', error)
      }
    };
    retrieveToken()
  }, [permission])

  return fcmToken
};

export default useFCMToken