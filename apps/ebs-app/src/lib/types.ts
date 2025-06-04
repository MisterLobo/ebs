type Timestamps = {
  created_at?: string,
  updated_at?: string,
  deleted_at?: string,
}

type EventStatus = 'draft' | 'open' | 'notify' | 'closed' | 'archived' | 'registration' | 'admission'

export type NewOrganizationRequestPayload = {
  name?: string,
  about?: string,
  country?: string,
  email?: string,
}

export type NewEventRequestPayload = {
  name: string,
  title: string,
  description?: string,
  location: string,
  date_time: string,
  time: string,
  deadline?: string,
  seats?: number,
  organization?: number,
  publish?: boolean,
  opens_at?: string,
  mode?: 'default' | 'scheduled',
}

export type NewTicketRequestPayload = {
  tier: string,
  type: string,
  currency: string,
  price: number,
  event: number,
}

export type Metadata = Record<string, any>
export type Resource = {
  resource_id?: string,
}

export type User = {
  id?: number,
  email?: string,
  name?: string,
  role?: string,
  uid?: string,
  email_verified?: boolean,
  phone_verified?: boolean,
  verified_at?: string,
  account_id?: string,
  subscription_id?: string,
  customer_id?: string,
  metadata?: Metadata,
}

export type Organization = {
  id: number,
  name?: string,
  country?: string,
  type?: string,
  owner_id?: number,
  stripe_account_id?: string,
  metadata?: Metadata,
  email?: string,
  connect_onboarding_url?: string,
  status?: string,
  verified?: boolean,
  payment_verified?: boolean,
  slug?: string,
  about?: string,

  events?: Event[],
  owner?: User,
} & Timestamps & Resource

export type Event = {
  id: number,
  title?: string,
  name: string,
  about?: string,
  location?: string,
  date_time?: string,
  status?: EventStatus,
  type?: string,
  organizer?: number,
  seats?: number,
  created_by?: number,
  opens_at?: string,
  deadline?: string,
  mode?: 'default' | 'scheduled',
  organization?: Organization,
} & Timestamps & Resource

export type Ticket = {
  id: number,
  tier?: string,
  type?: string,
  status?: string,
  price?: number,
  currency?: string,
  limited?: boolean,
  limit?: number,
  event_id?: number,
  stats?: TicketStats,
  event?: Event,
} & Timestamps & Resource

type CartItemTicket = Pick<Ticket, 'id' | 'tier' | 'currency' | 'price' | 'limit' | 'limited' | 'stats'>

type TicketStats = {
  free?: number,
  reserved?: number,
}

export type CartItem = {
  ticket?: CartItemTicket,
  stats?: TicketStats,
  qty?: number,
  subTotal?: number,
}

export type Booking = {
  id: number,
  ticket_id?: number,
  status?: string,
  qty?: number,
  unit_price?: number,
  subtotal?: number,
  currency?: string,
  user_id?: number,
  event_id?: number,
  event?: Event,
  user?: User,
  txn_id?: string,
  reserved_tickets?: Ticket[],
  reservations?: Reservation[],
  checkout_session_id?: string,
  payment_intent_id?: string,
  metadata?: Metadata,
  ticket?: Ticket,
  txn?: Transaction,
  slots_wanted?: number,
  slots_taken?: number,
} & Timestamps & Resource

export type Reservation = {
  id: number,
  ticket_id?: number,
  booking_id?: number,
  valid_until?: string,
  ticket?: Ticket,
  booking?: Booking,
  status?:string,
} & Timestamps & Resource

export type Transaction = {
  id?: string,
  currency?: string,
  amount?: number,
  source_name?: string,
  source_value?: string,
  reference_id?: string,
  metadata?: Metadata,
  checkout_session_id?: string,
  payment_intent_id?: string,
  status?: string,
  coupon_code?: string,
} & Timestamps & Resource

export type Admission = {
  id: number,
  by?: number,
  reservation_id?: number,
  type?: string,
  status?: string,
  reservation?: Reservation,
} & Timestamps & Resource

export type EventQueryFilters = {
  opens_at?: string,
  opens_before?: string,
  opens_after?: string,
  organizer?: string,
  created_at?: string,
  created_before?: string,
  created_after?: string,
  public?: string,
}

export type Waitlist = {
  id?: number,
  status?: string,
  event_id?: number,
  created_at?: string,
} & Timestamps & Resource