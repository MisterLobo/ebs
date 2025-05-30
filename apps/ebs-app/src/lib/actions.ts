'use server'

import { cookies } from 'next/headers'
import { Admission, Booking, Event, EventQueryFilters, NewEventRequestPayload, NewOrganizationRequestPayload, NewTicketRequestPayload, Organization, Ticket, Transaction } from './types'
import { notFound, redirect } from 'next/navigation'
import { TurnstileServerValidationResponse } from '@marsidev/react-turnstile'

export async function getActiveOrganization(): Promise<Organization | undefined> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/active`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status === 401) {
    $cookies.delete('token')
    return
  }
  if (response.status !== 200) {
    return
  }
  const { active } = await response.json()
  return active
}

export async function getOrganization(id: number) {
  return { id }
}

export async function createOrganization(data: NewOrganizationRequestPayload) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'POST',
    body: JSON.stringify(data),
  })
  const { id, error } = await response.json()
  console.log('[status]:', response.status)
  
  return { id, error }
}

export async function organizationOnboarding(id: number): Promise<{ completed?: boolean, account_id?: number, url?: string, error?: string, data?: Record<string, any> }> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${id}/onboarding`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  const { completed, account_id, url, error, data } = await response.json()
  console.log('[status]:', completed, url)
  
  return { completed, account_id, url, error, data }
}

export async function organizationOnboardingBegin(id: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${id}/onboarding`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'POST',
  })
  const { url, account_id, error } = await response.json()
  console.log('[status]:', response.status)
  
  return { url, account_id, error }
}

export async function listOrganizations() {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations?type=standard&owned=true`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status !== 200) {
    return
  }
  const { data } = await response.json()
  return data
}

export async function switchOrganization(id: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${id}/switch`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'POST',
  })
  if (response.status !== 200) {
    return false
  }
  const { access_token } = await response.json()
  if (access_token) {
    $cookies.set('token', access_token)
    return true
  }
  return false
}

export async function isSharedOrganization(): Promise<boolean | undefined> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/check`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status === 401 && $cookies.has('token')) {
    $cookies.delete('token')
    return
  }
  if (response.status !== 200) {
    return
  }
  const { shared } = await response.json()
  return shared
}

export async function isAuthenticated() {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  return !!token
}

export async function createEvent(data: NewEventRequestPayload) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/events`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'POST',
    body: JSON.stringify(data),
  })
  const { id, error } = await response.json()
  console.log('[status]:', response.status)
  
  return { id, error }
}

export async function publishEvent(id: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/events/${id}/publish`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'PATCH',
  })
  const { error } = await response.json()
  console.log('[status]:', response.status, error)
  return error
}

export async function setEventStatus(id: number, new_status: string) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/events/${id}/status`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'PATCH',
    body: JSON.stringify({
      new_status,
    })
  })
  const { error } = await response.json()
  console.log('[status]:', response.status, error)
  return error
}

export async function getEvents(orgId?: number, filters?: EventQueryFilters) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const searchParams = new URLSearchParams(filters)
  let requestUrl = new URL(`${process.env.API_HOST}/events?${searchParams.toString()}`)
  if (orgId) {
    requestUrl = new URL(`${process.env.API_HOST}/organizations/${orgId}/events?${searchParams.toString()}`)
  }
  console.log('[url]:', requestUrl.toString());
  
  const response = await fetch(requestUrl, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status !== 200) {
    return []
  }
  const { data = [], error } = await response.json()
  if (response.status !== 200 && error) {
    console.log('[error]:', error)
  }
  return data as Event[]
}

export async function getEventById(id: number): Promise<Event | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/events/${id}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status === 401) {
    $cookies.delete('token')
    redirect('/login')
  }
  const { data, error } = await response.json()
  if (response.status !== 200 && error) {
    console.log('[error]:', error)
  }
  return data
}

export async function createTicket(data: NewTicketRequestPayload) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/events/${data.event}/tickets`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'POST',
    body: JSON.stringify(data),
  })
  const { id, error } = await response.json()
  console.log('[status]:', response.status, error)

  return { id, error }
}

export async function publishTicket(id: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/tickets/${id}/publish`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'PATCH',
  })
  const { error } = await response.json()
  console.log('[status]:', response.status, error)
  return error
}

export async function getTickets(id: number, orgId?: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  let requestUrl = `${process.env.API_HOST}/events/${id}/tickets`
  if (orgId) {
    requestUrl = `${process.env.API_HOST}/organizations/${orgId}/events/${id}/tickets`
  }
  const response = await fetch(requestUrl, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status !== 200) {
    throw notFound()
  }
  const { data, error } = await response.json()
  if (error) {
    console.error('Error:', error)
    return []
  }
  
  return data as Ticket[]
}

export async function getTicket(id: number) {}

export async function registerUser(email: string) {
  const response = await fetch(`${process.env.API_HOST}/auth/register`, {
    headers: {
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'POST',
    body: JSON.stringify({
      email,
    }),
  })
  const { error } = await response.json()
  return {
    error,
    status: response.status,
  }
}

export async function loginUser(email: string) {
  const response = await fetch(`${process.env.API_HOST}/auth/login`, {
    headers: {
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'POST',
    body: JSON.stringify({
      email,
    }),
  })
  const { token, error } = await response.json()
  if (token) {
    const $cookies = await cookies()
    $cookies.set('token', token)
  }
  return {
    error,
    status: response.status,
  }
}

export async function createCheckoutSession(items: { qty: number, ticket: number }[]): Promise<{ url?: string, error?: string, status?: number }> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/checkout`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'POST',
    body: JSON.stringify({
      items,
    })
  })
  if (![200, 400].includes(response.status)) {
    console.error('API returned a non-200 status:', response.status)
    return {}
  }
  const { url, error } = await response.json()
  if (response.status === 200) {
    return { url }
  } else if (response.status === 400) {
    return { status: response.status, error }
  } else {
    return { status: response.status, error: response.statusText }
  }
}

export async function resumeCheckoutSession(id: string, checkoutId: string) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/transactions/checkout`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'POST',
    body: JSON.stringify({
      checkout_id: checkoutId,
      id,
    }),
  })
  const { url, error } = await response.json()
  return { url, error }
}

export async function cancelReservation(bookingId: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/bookings/${bookingId}/cancel`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'PUT',
  })
  if (response.status !== 204) {
    const { error } = await response.json()
    return { ok: false, error }
  }
  return { ok: true }
}

export async function cancelTransaction({ id, bookings }: { id?: string, bookings?: number[] }) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const body: Record<string, any> = {}
  if (id) {
    body['txn_id'] = id
    body['type'] = 'transaction'
  }
  if (bookings) {
    body['ids'] = bookings
    body['type'] = 'reservation'
  }
  const response = await fetch(`${process.env.API_HOST}/bookings/cancel`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'PUT',
    body: JSON.stringify(body),
  })
  if (response.status !== 204) {
    const { error } = await response.json()
    return { ok: false, error, status: response.status }
  }
  return { ok: true }
}

export async function getReservations(org = false) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const requestUrl = new URL(`${process.env.API_HOST}/reservations?org=${org}`)
  if (org) {
    requestUrl.searchParams.set('org', 'true')
  }
  const response = await fetch(requestUrl, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  const { data, error } = await response.json()
  
  return { data, error }
}

export async function getBookingTickets(id: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/bookings/${id}/reservations`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  const { data, count, error } = await response.json()
  
  return { data, count, error }
}

export async function getTicketStats(id: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/tickets/${id}/seats`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  const { free, reserved, error } = await response.json()
  
  return { free, reserved, error }
}

export async function subscribeToEvent(eventId: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/events/${eventId}/subscribe`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'POST',
  })
  const { id, error } = await response.json()
  
  return { id, error }
}

export async function getWaitlist() {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/events/waitlist`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  const { data, count, error } = await response.json()
  
  return { data, count, error }
}

export async function downloadTicket(id: number, resId: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/tickets/${id}/download/${resId}/code`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'POST',
  })
  if (response.status === 400) {
    const { error } = await response.json()
    return { error, status: response.status }
  } else if (response.status !== 200) {
    console.error('Error while requesting to download ticket.')
    return {}
  }
  const resblob = await response.blob()
  return { blob: resblob }
}

export async function logout() {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/auth/logout`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
    method: 'POST',
  })
  console.log('[logout]:', response.status)
  if ($cookies.has('token')) {
    $cookies.delete('token')
  }
}

export async function cfSiteverify(token: string): Promise<boolean> {
  const cfSecretKey = process.env.CF_TURNSTILE_SECRET_KEY ?? ''

  const response = await fetch(`https://challenges.cloudflare.com/turnstile/v0/siteverify`, {
    method: 'POST',
    body: `secret=${encodeURIComponent(cfSecretKey)}&response=${encodeURIComponent(token)}`,
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
    },
  })
  const validationResponse = await response.json() as TurnstileServerValidationResponse
  return validationResponse.success
}

export async function getBookings(org: number): Promise<Booking[]> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${org}/bookings`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status === 400) {
    const { error } = await response.json()
    console.error('Error response from API: ', error)
    return []
  } else if (response.status !== 200) {
    console.error('Something went wrong')
    return []
  }

  const { data } = await response.json()
  return data
}

export async function getAdmissions(orgId: number): Promise<Admission[]> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${orgId}/admissions`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status === 400) {
    const { error } = await response.json()
    console.error('Error response from API: ', error)
    return []
  } else if (response.status !== 200) {
    console.error('Something went wrong')
    return []
  }

  const { data } = await response.json()
  console.log('[data]:', data);
  
  return data
}

export async function getAdmission(id: number): Promise<Admission | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/admissions/${id}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status === 400) {
    const { error } = await response.json()
    console.error('Error response from API: ', error)
    return null
  } else if (response.status !== 200) {
    console.error('Something went wrong')
    return null
  }

  const { data } = await response.json()
  console.log('[data]:', data);
  
  return data
}

export async function getOrganizationTickets(org: number): Promise<Ticket[]> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${org}/tickets`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status === 400) {
    const { error } = await response.json()
    console.error('Error response from API: ', error)
    return []
  } else if (response.status !== 200) {
    console.error('Something went wrong')
    return []
  }

  const { data } = await response.json()
  return data
}

export async function getSoldTickets(org: number): Promise<any> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${org}/tickets/sold`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status === 400) {
    const { error } = await response.json()
    console.error('Error response from API: ', error)
    return []
  } else if (response.status !== 200) {
    console.error('Something went wrong')
    return []
  }

  const { data } = await response.json()
  return data
}

export async function getTransaction(id: number): Promise<Transaction | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/transactions/${id}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status === 400) {
    const { error } = await response.json()
    console.error('Error response from API: ', error)
    return null
  } else if (response.status !== 200) {
    console.error('Something went wrong')
    return null
  }

  const { data } = await response.json()
  return data
}

export async function aboutOrganization({ id, slug }: { id?: number, slug?: string }): Promise<Organization | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const url = new URL(`${process.env.API_HOST}/organizations/about`)
  if (id) {
    url.searchParams.set('id', `${id}`)
  }
  if (slug) {
    url.searchParams.set('slug', slug)
  }
  const response = await fetch(url, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status === 400) {
    const { error } = await response.json()
    console.error('Error response from API: ', error)
    return null
  } else if (response.status !== 200) {
    console.error('Something went wrong')
    return null
  }

  const { data } = await response.json()
  return data
}

export async function getEventSubscription(id: number): Promise<number | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/events/${id}/subscription`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'origin': `${process.env.APP_HOST}`,
      'x-secret': `${process.env.API_SECRET}`,
    },
  })
  if (response.status === 400) {
    const { error } = await response.json()
    console.error('Error response from API: ', error)
    return null
  } else if (response.status !== 200) {
    console.error('Something went wrong')
    return null
  }

  const { data } = await response.json()
  return data
}
