importScripts("https://www.gstatic.com/firebasejs/8.10.0/firebase-app.js")
importScripts("https://www.gstatic.com/firebasejs/8.10.0/firebase-messaging.js")

const firebaseConfig = {
  apiKey: 'AIzaSyDxiqPn092xxYRF0B-pOjntMD4uzg77AZs',
  // authDomain: 'events-booking-system-8184a.firebaseapp.com',
  projectId: 'events-booking-system-8184a',
  // storageBucket: 'events-booking-system-8184a.firebasestorage.app',
  messagingSenderId: '948261007536',
  appId: '1:948261007536:web:241dc42b2749739f2ecb75',
  // measurementId: 'your_keys',
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