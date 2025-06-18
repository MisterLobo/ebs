import { useEffect, useState } from "react"

const useNotificationPermission = () => {
  const [permission, setPermission] = useState<NotificationPermission>('default')

  useEffect(() => {
    const handler = () => setPermission(Notification.permission)
    handler()
    Notification.requestPermission().then(handler)

    navigator.permissions.query({ name: 'notifications' })
      .then(perm => {
        perm.onchange = handler
      })
  }, [])

  return permission
}

export default useNotificationPermission