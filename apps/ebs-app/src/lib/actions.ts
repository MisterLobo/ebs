'use server'

import { cookies } from 'next/headers'
import { Admission, Booking, Country, Event, EventQueryFilters, Geocoded, MFADevice, NewEventRequestPayload, NewOrganizationRequestPayload, NewTicketRequestPayload, Organization, Reservation, Ticket, Transaction, User } from './types'
import { notFound, redirect } from 'next/navigation'
import { TurnstileServerValidationResponse } from '@marsidev/react-turnstile'
import { Stripe } from 'stripe'
import getStripeApiClient from './stripe.server'
import { isProd } from './utils'

export async function getActiveOrganization(): Promise<Organization | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/active`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  if (response.status === 401) {
    $cookies.delete('token')
    return null
  }
  if (response.status !== 200) {
    return null
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
  
  return { id, error }
}

export async function organizationOnboarding(id: number): Promise<{ completed?: boolean, account_id?: number, url?: string, error?: string, data?: Record<string, any> }> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${id}/onboarding`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  const { completed, account_id, url, error, data } = await response.json()
  
  return { completed, account_id, url, error, data }
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
  
  return { url, account_id, error }
}

export async function listOrganizations() {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations?owned=true`, {
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
    /* $cookies.set('token', access_token, {
      secure: process.env.APP_ENV !== 'local',
      httpOnly: true,
      path: '/',
      sameSite: 'lax',
      domain: process.env.APP_HOST,
      expires: 60 * 60 * 24, //
    }) */
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
  return error
}

export async function cancelEvent(id: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/events/${id}/cancel`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'PUT',
  })
  const { error } = await response.json()
  return error
}

export async function setEventStatus(id: number, new_status: string) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/events/${id}/status`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'PATCH',
    body: JSON.stringify({
      new_status,
    })
  })
  if (response.status !== 204) {
    const { error } = await response.json()
    return error
  }
}

export async function getEvents(orgId?: number, filters?: EventQueryFilters) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const searchParams = new URLSearchParams(filters)
  let requestUrl = new URL(`${process.env.API_HOST}/events?${searchParams.toString()}`)
  if (orgId) {
    requestUrl = new URL(`${process.env.API_HOST}/organizations/${orgId}/events?${searchParams.toString()}`)
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

export async function getOrgEventById(orgId: number, id: number): Promise<Event | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${orgId}/events/${id}`, {
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
  return error
}

export async function closeTicket(id: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/tickets/${id}/close`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'PATCH',
  })
  const { error } = await response.json()
  return error
}

export async function archiveTicket(id: number) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/tickets/${id}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'DELETE',
  })
  const { error } = await response.json()
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
  
  return data as Ticket[]
}

export async function getTicket(id: number): Promise<Ticket | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/tickets/${id}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  if (response.status !== 200) {
    return null
  }
  const { data, error } = await response.json()
  if (error) {
    console.error('Error:', error)
    return null
  }
  
  return data as Ticket
}

export async function getBookingById(id: number): Promise<Booking | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/bookings/${id}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  if (response.status !== 200) {
    return null
  }
  const { data, error } = await response.json()
  if (error) {
    console.error('Error:', error)
    return null
  }
  
  return data as Booking
}

export async function registerUser(email: string, idToken: string) {
  const response = await fetch(`${process.env.API_HOST}/auth/register`, {
    headers: {
      'Authorization': idToken,
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

export async function loginUser(email: string, idToken: string): Promise<{ ok: boolean, publicKey?: any, error?: string, status: number }> {
  const response = await fetch(`${process.env.API_HOST}/auth/login`, {
    headers: {
      'Authorization': idToken,
    },
    method: 'POST',
    body: JSON.stringify({
      email,
    }),
  })
  const $cookies = await cookies()
  const secure = process.env.APP_HOST?.startsWith('https://') || process.env.APP_ENV !== 'local'
  const expiry = 86_400_000 // 24h
  const cookieOpts = {
    secure,
    httpOnly: true,
    path: '/',
    sameSite: 'lax',
    domain: process.env.APP_DOMAIN ?? '',
    expires: Date.now()+expiry,
  }
  $cookies.set('id_token', idToken, cookieOpts as any)
  if (!isProd()) {
    console.log('sample cookie:', cookieOpts)
  }
  if (response.status === 401 && response.headers.get('X-Authenticate-MFA') === 'true') {
    const flowId = response.headers.get('X-MFA-Flow-ID')
    const challenge = response.headers.get('X-MFA-Challenge')
    $cookies.set('mfa_challenge', challenge ?? '', cookieOpts as any)
    $cookies.set('mfa_flow_id', flowId ?? '', cookieOpts as any)
    const data = await loginPasskeyMFA(email)
    return {
      ...data,
      status: response.status,
    }
  }
  const { token, error } = await response.json()
  if (token) {
    $cookies.set('token', token, cookieOpts as any)
  }
  return {
    ok: false,
    error,
    status: response.status,
  }
}

export async function createCheckoutSession(items: { qty: number, ticket: number }[], promoCode?: string): Promise<{ url?: string, error?: string, status?: number }> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/checkout`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'POST',
    body: JSON.stringify({
      items,
      promo_code: promoCode,
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

export async function checkPromoCode(code: string): Promise<{ error?: string, found?: boolean }> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/stripe/coupon?code=${code}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  const { found, error } = await response.json()
  if (response.status === 404) {
    return { error }
  }
  if (response.status !== 200) {
    console.error('API returned a non-200 status:', response.status)
    return { found: false }
  }
  return { found }
}

export async function resumeCheckoutSession(id: string, checkoutId: string) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/transactions/checkout`, {
    headers: {
      'Authorization': `Bearer ${token}`,
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

export async function getTicketBookings(id: number): Promise<{ data?: Booking[], error?: string } | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/tickets/${id}/bookings`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  const { data, error } = await response.json()
  if (response.status !== 200) {
    return { error }
  }
  
  return { data }
}

export async function getTicketReservations(id: number): Promise<{ data?: Reservation[], error?: string } | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/tickets/${id}/reservations`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  const { data, error } = await response.json()
  if (response.status !== 200) {
    return { error }
  }
  
  return { data }
}

export async function getBookingReservations(id: number): Promise<{ data?: Booking, error?: string } | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/bookings/${id}/reservations`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  const { data, error } = await response.json()
  if (response.status !== 200) {
    return { error }
  }
  
  return { data }
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
  if (error) {
    console.error('error:', error)
  }
  
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

export async function getTicketShareLink(id: number, resId: number): Promise<string | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/tickets/${id}/download/${resId}/code?share_link=true`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'POST',
  })
  if (response.status === 400) {
    const { error } = await response.json()
    console.error('error retrieving share URL for ticket:', error)
    return null
  } else if (response.status !== 200) {
    console.error('Error while requesting to download ticket.')
    return null
  }
  const { url } = await response.json()
  return url
}

export async function logout() {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/auth/logout`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'POST',
  })
  if (response.status !== 200) {
    console.log('something went wrong')
    return false
  }
  console.log('[logout]:', response.status)
  if ($cookies.has('token')) {
    $cookies.delete('token')
  }
  return true
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

export async function getAdmission(id: number): Promise<Admission | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/admissions/${id}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
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

export async function getOrganizationTickets(org: number): Promise<Ticket[]> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${org}/tickets`, {
    headers: {
      'Authorization': `Bearer ${token}`,
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

export async function getMonthlyCustomers(org: number): Promise<number | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${org}/customers/count`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  if (response.status === 400) {
    const { error } = await response.json()
    console.error('Error response from API: ', error)
    return 0
  } else if (response.status !== 200) {
    console.error('Something went wrong')
    return 0
  }

  const { data } = await response.json()
  return data
}

export async function getDailyTransactions(org: number): Promise<any> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${org}/transactions/daily`, {
    headers: {
      'Authorization': `Bearer ${token}`,
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

export async function getOrgDashboard(org: number): Promise<any> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/organizations/${org}/dashboard`, {
    headers: {
      'Authorization': `Bearer ${token}`,
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

export async function getTransaction(id: number): Promise<Transaction | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/transactions/${id}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
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

export async function me(): Promise<{ me?: any, md?: Record<string, string> } | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/me`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  if (response.status !== 200) {
    console.error('Something went wrong')
    return null
  }

  const { data } = await response.json()
  return data
}

export async function getStripeAccount(): Promise<Stripe.Account | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/account`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  if (response.status !== 200) {
    console.error('Something went wrong')
    return null
  }

  const { data } = await response.json()
  return data as Stripe.Account
}

export async function getStripeCustomer(): Promise<Stripe.Customer | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/account`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  if (response.status !== 200) {
    console.error('Something went wrong')
    return null
  }

  const { data } = await response.json()
  return data as Stripe.Customer
}

export async function getSubscription(): Promise<Record<string, any> & {
  sub?: Stripe.Subscription,
  curr?: Stripe.SubscriptionItem,
  price?: Stripe.Price,
  prod?: Stripe.Product,
}> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/stripe/subscription`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  if (response.status !== 200) {
    console.error('Something went wrong')
    return {}
  }
  const { data } = await response.json()
  const stripe = getStripeApiClient()
  const subscriptionData = await stripe.subscriptions.retrieve(data)
  const subscription = subscriptionData as Stripe.Subscription
  const defaultSubscription = subscription.items.data?.[0]
  const price = await stripe.prices.retrieve(defaultSubscription?.price.id, {
    expand: ['product'],
  })
  const prod = price?.product as Stripe.Product
  console.log(price, prod);
  
  return {
    sub: subscription,
    curr: defaultSubscription,
    price,
    prod,
  }
}

export async function getPaymentMethods(): Promise<Stripe.PaymentMethod[] | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/stripe/payment_methods`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  if (response.status !== 200) {
    console.error('Something went wrong')
    return null
  }
  const { data } = await response.json()
  const user = data as User
  // const customer = await stripe.customers.retrieve(user.account_id as string)

  const stripe = getStripeApiClient()
  const { data: listData } = await stripe.paymentMethods.list({
    customer: user.account_id,
  })

  return listData as Stripe.PaymentMethod[]
}

export async function subscribeToFCMTopics(fcmToken: string, ...topics: string[]) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/fcm`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify({
      token: fcmToken,
      topics,
    }),
    method: 'POST',
  })
  if (response.status !== 200) {
    console.error('Something went wrong')
  }
}

export async function sendFCMMessage(topic: string) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/fcm/send`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify({
      topic,
    }),
    method: 'POST',
  })
  if (response.status !== 200) {
    console.error('Something went wrong')
  }
}

export async function geocode(lat: number, long: number): Promise<Geocoded> {
  const url = new URL('https://maps.googleapis.com/maps/api/timezone/json')
  url.searchParams.set('location', `${lat},${long}`)
  url.searchParams.set('timestamp', (Date.now()/1e3).toString())
  url.searchParams.set('key', process.env.GEOTZ_API_KEY as string)
  const response = await fetch(url)
  const data = await response.json()
  return data
}

export async function placeInfo(lat: number, long: number): Promise<any> {
  const url = new URL('https://maps.googleapis.com/maps/api/timezone/json')
  url.searchParams.set('location', `${lat},${long}`)
  url.searchParams.set('timestamp', (Date.now()/1e3).toString())
  url.searchParams.set('key', process.env.GEOTZ_API_KEY as string)
  const response = await fetch(url)
  const data = await response.json()
  return data
}

export async function listCountries(): Promise<Country[]> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const response = await fetch(`${process.env.API_HOST}/countries`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  if (response.status !== 200) {
    return []
  }
  const { countries } = await response.json()
  return countries
}

export async function registerPasskeyMFA(creds: { [key:string]: any } | null, step: 'start' | 'finish' = 'start') {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const url = new URL(`${process.env.API_HOST}/accounts/passkey/register/${step}`)
  let body = JSON.stringify(creds)
  const response = await fetch(url, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    method: 'POST',
    body,
  })
  if (response.status !== 200) {
    return false
  }
  if (step === 'start') {
    const publicKey = await response.json()
    if (!publicKey) {
      return false
    }
    publicKey.challenge = Uint8Array.from(Buffer.from(publicKey.challenge, 'base64url'))
    publicKey.user.id = Uint8Array.from(Buffer.from(publicKey.user.id, 'base64url'))
    return publicKey
  }
  return true
}
export async function loginPasskeyMFA(email: string, creds?: { [key:string]: any } | null, step: 'start' | 'finish' = 'start'): Promise<{ ok: boolean, publicKey?: any, error?: string }> {
  const url = new URL(`${process.env.API_HOST}/passkey/login/${step}`)
  const body: Record<string, any> = creds ?? {}
  if (step === 'start') {
    body.email = email
  } else if (step === 'finish') {
    url.searchParams.set('email', email)
  }
  const $cookies = await cookies()
  const ftoken = $cookies.get('id_token')?.value
  const challenge = $cookies.get('mfa_challenge')?.value
  const flowId = $cookies.get('mfa_flow_id')?.value
  const headers = {
    'Authorization': `${ftoken}`,
    'Content-Type': 'application/json',
    'X-MFA-Flow-ID': flowId,
    'X-Authenticate-MFA': 'true',
    'X-MFA-Challenge': challenge,
  }
  const requestInit = {
    headers,
    method: 'POST',
    body: JSON.stringify(body),
  } as RequestInit
  const response = await fetch(url, requestInit)
  if (response.status === 401) {
    const { error } = await response.json()
    return { ok: false, error }
  }
  if (response.status !== 200) {
    return { ok: false }
  }
  const { token, publicKey, error } = await response.json()
  if (error) {
    return { ok: false, error }
  }
  if (step === 'start') {
    if (!publicKey) {
      return { ok: false, error }
    }
    const challenge = Uint8Array.from(Buffer.from(publicKey.challenge, 'base64url'))
    const allowCredentials = Array.from(publicKey.allowCredentials).map((ac: any) => {
      const id = Uint8Array.from(Buffer.from(ac?.id, 'base64url'))
      ac.id = id
      return ac
    })
    publicKey.challenge = challenge
    publicKey.allowCredentials = allowCredentials
    return { ok: true, publicKey }
  } else if (step === 'finish') {
    $cookies
      .delete('id_token')
      .delete('mfa_flow_id')
      .delete('mfa_challenge')
    if (token) {
      const secure = process.env.APP_HOST?.startsWith('https://') || process.env.APP_ENV !== 'local'
      const $cookies = await cookies()
      $cookies.set('token', token, {
        secure,
        httpOnly: true,
        path: '/',
        sameSite: 'lax',
        domain: process.env.APP_DOMAIN,
        expires: Date.now()+(3600*1e3),
      })
      return { ok: true }
    }
  }
  return { ok: false }
}
export async function getMFADevices(): Promise<MFADevice[]> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const url = new URL(`${process.env.API_HOST}/accounts/devices`)
  const response = await fetch(url, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  if (response.status !== 200) {
    return []
  }
  const { data } = await response.json()
  return data
}
export async function revokeMFADevice(name: string): Promise<boolean> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const url = new URL(`${process.env.API_HOST}/accounts/devices/revoke`)
  const response = await fetch(url, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'PUT',
    body: JSON.stringify({
      name,
    })
  })
  if (response.status !== 200) {
    return false
  }
  return true
}
export async function requestVerificationCode(email: string): Promise<boolean> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const url = new URL(`${process.env.API_HOST}/passkey/login/request_code`)
  const response = await fetch(url, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'POST',
    body: JSON.stringify({ email })
  })
  if (response.status !== 200) {
    return false
  }
  return true
}
export async function verifyVerificationCode(email: string, code: string) {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const url = new URL(`${process.env.API_HOST}/passkey/login/verify_code`)
  const response = await fetch(url, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'POST',
    body: JSON.stringify({ email, code })
  })
  if (response.status !== 200) {
    return false
  }
  return true
}

export async function encodeBase64url(s: string): Promise<string> {
  return Buffer.from(s).toString('base64url')
}

export async function connectCalendar(redirect: string): Promise<string | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const reqUrl = new URL(`${process.env.API_HOST}/accounts/calendar/connect`)
  const response = await fetch(reqUrl, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    method: 'POST',
    body: JSON.stringify({
      redirect: `${process.env.APP_HOST}${redirect}`,
    })
  })
  if (response.status !== 200) {
    return null
  }
  const { url } = await response.json()
  return url
}

export async function getCalendar(id: number): Promise<string | null> {
  const $cookies = await cookies()
  const token = $cookies.get('token')?.value
  const reqUrl = new URL(`${process.env.API_HOST}/accounts/${id}/calendar?type=org`)
  const response = await fetch(reqUrl, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  })
  if (response.status !== 200) {
    return null
  }
  const { url } = await response.json()
  return url
}

export const tokenRetrieved = async (token: string) => {
  await subscribeToFCMTopics(token, 'EventSubscription', 'Events', 'Personal')
}