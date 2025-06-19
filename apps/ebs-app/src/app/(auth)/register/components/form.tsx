'use client'

import { Button } from '@/components/ui/button'
import { cfSiteverify, registerUser } from '@/lib/actions'
import { auth, provider } from '@/lib/firebase'
import { Turnstile, TurnstileInstance } from '@marsidev/react-turnstile'
import { signInWithPopup } from 'firebase/auth'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useRef, useState } from 'react'

export default function RegisterForm() {
  const router = useRouter()
  const [error, setError] = useState<string>()
	const [token, setToken] = useState<string>()
  const turnstileRef = useRef<TurnstileInstance>(null)
  const register = async () => {
    setError(undefined)
    const success = await cfSiteverify(token as string)
    if (!success) {
      setError('Failed to verify captcha')
      return
    }
    try {
      const credential = await signInWithPopup(auth, provider)
      if (credential.user) {
        const idToken = await credential.user.getIdToken()
        const { error } = await registerUser(credential.user.email as string, idToken)
        if (error) {
          setError(error)
          return
        }
        router.push('/login')
      }
    } catch (error: any) {
      alert(`[error]: ${error.message}`);
    }
  }

  return (
    <div className="flex flex-col items-center h-96 w-lg justify-center p-4 relative border rounded-xl">
      {error && <p className="text-red-500 text-sm text-center">ERROR: { error }</p>}
      <h1 className="text-4xl font-semibold leading-none my-4">SIGN UP</h1>
      <form className="flex flex-col space-y-4 w-full items-center justify-center">
        <Turnstile
          ref={turnstileRef}
          siteKey={process.env.NEXT_PUBLIC_CF_TURNSTILE_SITE_KEY as string}
          onError={e => setError(e)}
          onExpire={() => setError('token has expired')}
          onSuccess={token => {
            setToken(token)
            setError(undefined)
          }}
        />
        <Button type="button" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none w-fit" onClick={register}>Sign up with Google</Button>
      </form>
      <div className="flex w-full items-center justify-center mt-4">
        <span>Already a member? <Link href="/login" className="text-purple-600 font-semibold hover:underline">Log in</Link> here</span>
      </div>
    </div>
  )
}