import { FirebaseOptions, initializeApp } from 'firebase/app'
import { getAuth, GoogleAuthProvider } from 'firebase/auth'

const firebaseConfig: FirebaseOptions = {
  apiKey: process.env.NEXT_PUBLIC_FBASE_API_KEY,
  authDomain: process.env.NEXT_PUBLIC_FBASE_AUTH_DOMAIN,
  projectId: process.env.NEXT_PUBLIC_FBASE_PROJECT_ID,
  storageBucket: process.env.NEXT_PUBLIC_FBASE_STORAGE_BUCKET,
  messagingSenderId: process.env.NEXT_PUBLIC_FBASE_MESSAGING_SENDER_ID,
  appId: process.env.NEXT_PUBLIC_FBASE_APP_ID,
}

const app = initializeApp(firebaseConfig)
const auth = getAuth(app)

const provider = new GoogleAuthProvider()
export { app, auth, provider }