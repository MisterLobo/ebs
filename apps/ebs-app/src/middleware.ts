import { createNEMO, type GlobalMiddlewareConfig, type MiddlewareConfig } from '@rescale/nemo'
import { cookies } from 'next/headers'
import { NextRequest, NextResponse } from 'next/server'
import { getActiveOrganization, logout } from './lib/actions'
import { Organization } from './lib/types'

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - api (API routes)
     * - _next/static (static files)
     * - favicon.ico (favicon file)
     */
    '/((?!api|_next/static|favicon.ico|login|register|.well-known).*)',
  ],
}

const globalMiddlewares = {
  before: async (request, event) => {
    const requestHeaders = new Headers(request.headers)
    const url = request.url
    requestHeaders.set('x-url', url)
  }
} satisfies GlobalMiddlewareConfig

const middlewares = {
  '/maintenance': [
    async (request: NextRequest) => {
      if (process.env.MAINTENANCE_MODE !== 'true') {
        return NextResponse.rewrite(new URL('/not-found', request.url))
      }
      return NextResponse.next()
    }
  ],
  '/:path*': [
    async (request: NextRequest) => {
      if (request.url.endsWith('/maintenance') && process.env.MAINTENANCE_MODE !== 'true') {
        return NextResponse.rewrite(new URL('/not-found', request.url))
      }
      return NextResponse.next()
    },
    async (request: NextRequest) => {
      if (!request.url.endsWith('/maintenance') && process.env.MAINTENANCE_MODE === 'true') {
        return NextResponse.redirect(new URL('/maintenance', request.url))
      }
      const requestHeaders = new Headers(request.headers)
      const url = request.url

      const $cookies = await cookies()
      const token = $cookies.get('token')?.value
      if (!token) {
        return NextResponse.redirect(new URL('/login', request.url))
      }
      const org = await getActiveOrganization() as Organization
      if (!org) {
        await logout()
        return NextResponse.redirect(new URL('/login', request.url))
      }

      // You can also set request headers in NextResponse.next
      const response = NextResponse.next({
        request: {
          // New request headers
          headers: requestHeaders,
        },
      })
    
      response.headers.set('x-url', url)
      return response
    },
  ],
} satisfies MiddlewareConfig

export const middleware = createNEMO(middlewares, globalMiddlewares)