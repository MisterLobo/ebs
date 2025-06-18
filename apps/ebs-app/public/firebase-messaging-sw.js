importScripts("https://www.gstatic.com/firebasejs/8.10.0/firebase-app.js")
importScripts("https://www.gstatic.com/firebasejs/8.10.0/firebase-messaging.js")

const firebaseConfig = {
  apiKey: 'API_KEY',
  projectId: 'PROJECT_ID',
  messagingSenderId: 'SENDER_ID',
  appId: 'APP_ID',
};
// eslint-disable-next-line no-undef
firebase.initializeApp(firebaseConfig)
// eslint-disable-next-line no-undef
const messaging = firebase.messaging()

messaging.onBackgroundMessage((payload) => {
  const notificationTitle = payload.title ?? 'Background Notification'
  const notificationOptions = {
    body: payload,
  }
  self.registration.showNotification(notificationTitle, notificationOptions)
})