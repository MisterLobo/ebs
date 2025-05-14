import LoginForm from './components/form'

export default async function LoginPage() {
  return (
    <div className="flex w-full min-h-screen items-center">
      <div className="mx-auto">
        <LoginForm />
      </div>
    </div>
  )
}