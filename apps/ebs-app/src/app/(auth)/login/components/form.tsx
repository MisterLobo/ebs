'use client'

import { Button } from '@/components/ui/button'
import { loginUser } from '@/lib/actions'
import { auth, provider } from '@/lib/firebase'
import { signInWithPopup } from 'firebase/auth'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useState } from 'react'

export default function LoginForm() {
  const router = useRouter()
  const [error, setError] = useState<string>()
  const login = async () => {
    setError(undefined)
    try {
      const credential = await signInWithPopup(auth, provider)
      if (credential.user) {
        const { error } = await loginUser(credential.user.email as string)
        if (error) {
          setError(error)
          return
        }
        router.push('/dashboard')
      }
    } catch (error: any) {
      alert(`[error]: ${error.message}`)
    }
  }

  return (
    <div className="flex flex-col items-center h-96 min-w-lg justify-center p-4 relative border rounded-xl">
      {error && <p className="text-red-500">{ error }</p>}
      <h1 className="text-4xl font-semibold leading-none my-4">LOG IN</h1>
      <form className="flex space-y-4 w-full items-center justify-center">
        <Button type="button" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none w-fit" onClick={login}>Log in with Google</Button>
      </form>
      <div className="flex w-full items-center justify-center mt-4">
        <span>Not a member? <Link href="/register" className="text-purple-600 font-semibold hover:underline">Register</Link> here</span>
      </div>
    </div>
  )
}