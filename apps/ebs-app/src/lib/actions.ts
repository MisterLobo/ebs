'use server'

import { cookies } from 'next/headers'
import { Event, EventQueryFilters, NewEventRequestPayload, NewOrganizationRequestPayload, NewTicketRequestPayload, Organization, Ticket } from './types'
import { notFound, redirect } from 'next/navigation'

export async function getActiveOrganization(): Promise<Organization | undefined> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/active`, {
    headers: {
      'Authorization': `Bearer ${token}`,
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
    },
    method: 'POST',
    body: JSON.stringify(data),
  })
  const { id, error } = await response.json()
  console.log('[status]:', response.status)
  
  return { id, error }
}

export async function organizationOnboarding(id: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${id}/onboarding`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  const { completed, account_id, url, error } = await response.json()
  console.log('[status]:', completed, url)
  
  return { completed, account_id, url, error }
}

export async function organizationOnboardingBegin(id: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${id}/onboarding`, {
    headers: {
      'Authorization': `Bearer ${token}`,
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
    },
    method: 'PATCH',
  })
  const { error } = await response.json()
  console.log('[status]:', response.status, error)
  return error
}

export async function getEvents(orgId?: number, filters?: EventQueryFilters) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const searchParams = new URLSearchParams(filters)
  let requestUrl = new URL(`/events?${searchParams.toString()}`, process.env.API_HOST)
  if (orgId) {
    requestUrl = new URL(`/organizations/${orgId}/events?${searchParams.toString()}`, process.env.API_HOST)
  }
  const response = await fetch(requestUrl, {
    headers: {
      'Authorization': `Bearer ${token}`,
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
  console.log('[data]:', data);
  
  return data as Ticket[]
}

export async function getTicket(id: number) {}

export async function registerUser(email: string) {
  const response = await fetch(`${process.env.API_HOST}/register`, {
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
  const response = await fetch(`${process.env.API_HOST}/login`, {
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

export async function createCheckoutSession(items: { qty: number, ticket: number }[]) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/checkout`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'POST',
    body: JSON.stringify({
      items,
    })
  })
  const { url, error } = await response.json()
  return { url, error }
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
    },
  })
  const { data, count, error } = await response.json()
  console.log('res: ', data, count, error);
  
  return { data, error }
}

export async function getBookingTickets(id: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/bookings/${id}/reservations`, {
    headers: {
      'Authorization': `Bearer ${token}`,
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
    },
    method: 'POST',
  })
  const resblob = await response.blob()
  return resblob
}

export async function logout() {
  const $cookies = await cookies()
  if ($cookies.has('token')) {
    $cookies.delete('token')
  }
}