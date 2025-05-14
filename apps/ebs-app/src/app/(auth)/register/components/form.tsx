'use client'

import { Button } from '@/components/ui/button'
import { registerUser } from '@/lib/actions'
import { auth, provider } from '@/lib/firebase'
import { signInWithPopup } from 'firebase/auth'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useState } from 'react'

export default function RegisterForm() {
  const router = useRouter()
  const [error, setError] = useState<string>()
  const register = async () => {
    setError(undefined)
    try {
      const credential = await signInWithPopup(auth, provider)
      if (credential.user) {
        const { error } = await registerUser(credential.user.email as string)
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
    <div className="flex flex-col items-center h-96 min-w-lg justify-center p-4 relative border rounded-xl">
      {error && <p className="text-red-500">{ error }</p>}
      <h1 className="text-4xl font-semibold leading-none my-4">SIGN UP</h1>
      <form className="flex space-y-4 w-full items-center justify-center">
        <Button type="button" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none w-fit" onClick={register}>Sign up with Google</Button>
      </form>
      <div className="flex w-full items-center justify-center mt-4">
        <span>Already a member? <Link href="/login" className="text-purple-600 font-semibold hover:underline">Log in</Link> here</span>
      </div>
    </div>
  )
}