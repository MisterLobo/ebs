'use client'

import { Button } from '@/components/ui/button'
import { cfSiteverify, loginUser } from '@/lib/actions'
import { auth, provider } from '@/lib/firebase'
import { signInWithPopup } from 'firebase/auth'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useRef, useState } from 'react'
import { Turnstile, type TurnstileInstance } from '@marsidev/react-turnstile'

export default function LoginForm() {
  const router = useRouter()
  const [error, setError] = useState<string>()
	const [token, setToken] = useState<string>()
	const turnstileRef = useRef<TurnstileInstance>(null)
  const login = async () => {
    setError(undefined)
    if (turnstileRef.current?.isExpired()) {
      setError('token has expired')
    }
    const success = await cfSiteverify(token as string)
    if (!success) {
      setError('Failed to verify captcha')
      return
    }
    try {
      const credential = await signInWithPopup(auth, provider)
      if (credential.user) {
        const { error } = await loginUser(credential.user.email as string)
        if (error) {
          setError(error)
          return
        }
        router.push('/personal/dashboard')
      }
    } catch (error: any) {
      alert(`[error]: ${error.message}`)
    }
  }

  return (
    <div className="flex flex-col items-center h-96 min-w-lg justify-center p-4 relative border rounded-xl">
      {error && <p className="text-red-500 text-sm">{ error }</p>}
      <h1 className="text-4xl font-semibold leading-none my-4">LOG IN</h1>
      <form className="flex flex-col space-y-4 w-full items-center justify-center">
        <Turnstile
          ref={turnstileRef}
          siteKey={process.env.NEXT_PUBLIC_CF_TURNSTILE_SITE_KEY as string}
          onError={e => setError(`ERROR: ${e}`)}
          onExpire={() => setError('ERROR: token has expired')}
          onSuccess={token => {
            setToken(token)
            setError(undefined)
          }}
        />
        <Button type="button" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none w-fit" onClick={login}>Log in with Google</Button>
      </form>
      <div className="flex w-full items-center justify-center mt-4">
        <span>Not a member? <Link href="/register" className="text-purple-600 font-semibold hover:underline">Register</Link> here</span>
      </div>
    </div>
  )
}