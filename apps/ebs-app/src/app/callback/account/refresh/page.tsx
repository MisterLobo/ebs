import { getActiveOrganization } from '@/lib/actions'
import { cookies } from 'next/headers'
import { redirect } from 'next/navigation'

export default async function AccountRefreshPage() {
  const org = await getActiveOrganization()
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${org?.id}/account/refresh`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'POST',
  })
  if (response.status !== 200) {
    return (
      <p>Error: { response.status }</p>
    )
  }
  const { url, error } = await response.json()
  if (url) {
    redirect(url)
  }
  return (
    <p>Error: { error }</p>
  )
}