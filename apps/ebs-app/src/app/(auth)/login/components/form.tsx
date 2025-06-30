'use client'

import { Button } from '@/components/ui/button'
import { cfSiteverify, loginUser, loginPasskeyMFA } from '@/lib/actions'
import { auth, provider } from '@/lib/firebase'
import { signInWithPopup } from 'firebase/auth'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useCallback, useRef, useState } from 'react'
import { Turnstile, type TurnstileInstance } from '@marsidev/react-turnstile'
import { toast } from 'sonner'

export default function LoginForm() {
  const router = useRouter()
  const [error, setError] = useState<string>()
	const [token, setToken] = useState<string>()
	const turnstileRef = useRef<TurnstileInstance>(null)
  const loginWithGoogle = useCallback(async () => {
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
        const idToken = await credential.user.getIdToken()
        const { error, publicKey } = await loginUser(credential.user.email as string, idToken)
        if (error) {
          setError(error)
          return
        }
        if (publicKey) {
          const creds = await navigator.credentials.get({ publicKey })
          const cjson: Record<string, any> = JSON.parse(JSON.stringify(creds))
          const { ok } = await loginPasskeyMFA(credential.user.email as string, cjson, 'finish')
          if (!ok) {
            toast('ERROR', {
              description: 'Log in failed',
            })
            return
          }
          toast('You are now logged in!')
        }
        router.push('/')
      }
    } catch (error: any) {
      alert(error.message)
    }
  }, [])

  return (
    <div className="flex flex-col items-center h-96 min-w-lg justify-center p-4 relative border rounded-xl">
      {error && <p className="text-red-500 text-sm text-center">ERROR: { error }</p>}
      <h1 className="text-4xl font-semibold leading-none my-4">LOG IN</h1>
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
        <Button type="button" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none w-fit" onClick={loginWithGoogle}>Log in with Google</Button>
      </form>
      <div className="flex w-full items-center justify-center mt-4">
        <span>No account yet? <Link href="/register" className="text-purple-600 font-semibold hover:underline">Register</Link> here</span>
      </div>
    </div>
  )
}