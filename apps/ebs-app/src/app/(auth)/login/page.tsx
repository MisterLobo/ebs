import { redirect } from 'next/navigation'
import LoginForm from './components/form'

export default async function LoginPage() {
  if (process.env.MAINTENANCE_MODE === 'true') {
    redirect('/maintenance')
  }
  return (
    <div className="flex w-full min-h-screen items-center">
      <div className="mx-auto">
        <LoginForm />
      </div>
    </div>
  )
}