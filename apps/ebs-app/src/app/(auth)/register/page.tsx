'use client'

import { redirect } from 'next/navigation'
import RegisterForm from './components/form'

export default function RegisterPage() {
  if (process.env.MAINTENANCE_MODE === 'true') {
    redirect('/maintenance')
  }
  return (
    <div className="flex w-full min-h-screen items-center">
      <div className="mx-auto">
        <RegisterForm />
      </div>
    </div>
  )
}