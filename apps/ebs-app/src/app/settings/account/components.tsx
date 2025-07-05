'use client'

import { Button } from '@/components/ui/button'
import { geocode, registerPasskeyMFA, revokeMFADevice } from '@/lib/actions'
import { Geocoded, MFADevice, User } from '@/lib/types'
import { Check, Edit } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'

type Props = {
  user: User,
}
export function SettingsAccountGeneral({ user }: Props) {
  const [timezone, setTimezone] = useState<Geocoded>()
  useEffect(() => {
    if (navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(async (position) => {
        const timezone = await geocode(position.coords.latitude, position.coords.longitude)
        setTimezone(timezone)
      })
    }
  }, [])
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="col-span-1">
        <p>Email</p>
      </div>
      <div className="col-span-1 flex w-full items-center justify-center gap-2">
        <span>{user.email}</span>
        {user.email_verified ?
          <Check /> :
          <Button variant="link" className="size-8 rounded-full">verify</Button>
        }
      </div>
      <div className="col-span-1">
        <p>Phone: {user.phone}</p>
      </div>
      <div className="col-span-1 flex w-full items-center justify-center">
        {user.phone
          ? <Button className="w-32" disabled>Verify phone</Button>
          : <Button className="w-32" disabled><Edit /> Update</Button>
        }
      </div>
      <div className="col-span-1">
        <p>Preferred timezone:</p>
      </div>
      <div className="col-span-1 flex w-full items-center justify-center gap-2">
        <span>{timezone?.timeZoneId ?? 'Autodetect'}</span>
        <Button variant="ghost" className="size-8 rounded-full" size="icon" disabled><Edit /></Button>
      </div>
    </div>
  )
}

type SettingsAccountSecurityProps = {
  user?: User,
  devices?: MFADevice[],
}
export function SettingsAccountSecurity({ devices }: SettingsAccountSecurityProps) {
  const router = useRouter()
  const [busy, setBusy] = useState(false)
  const registerMFADevice = useCallback(async () => {
    setBusy(true)
    const options = await registerPasskeyMFA(null)
    if (!options) {
      const error = 'Something went wrong on our end.'
      console.error(error)
      return
    }
    try {
      const creds = await navigator.credentials.create({ publicKey: options })
      const cjson: Record<string, any> = JSON.parse(JSON.stringify(creds))
      const ok = await registerPasskeyMFA(cjson, 'finish') as boolean
      setBusy(false)
      if (ok) {
        toast('NOTICE', {
          description: 'MFA device registered successfully!',
        })
        router.refresh()
        return
      }
      toast('ERROR', {
        description: 'MFA device registration failed',
      })
    } catch (e: any) {
      console.error(e)
      toast('ERROR', {
        description: 'MFA device registration aborted',
      })
    } finally {
      setBusy(false)
    }
  }, [])
  const revokeDevice = useCallback(async (name: string) => {
    if (prompt(`Revoke ${name}? Type 'revoke' to continue.`) !== 'revoke') {
      return
    }
    setBusy(true)
    const success = await revokeMFADevice(name)
    setBusy(false)
    if (success) {
      toast('NOTICE', {
        description: 'MFA device has been revoked',
      })
      router.refresh()
      return
    }
    toast('ERROR', {
      description: 'MFA device revocation failed',
    })
  }, [])
  return (
    <div>
      <div>
        <div className="space-y-4 mb-10">
        {devices && devices.map((d, i) => (
          <div key={i} className="flex w-full items-center justify-between">
            <div className="col-span-1">
              <p className="uppercase">{ d.name || '(No name)' }</p>
            </div>
            <div className="col-span-1">
              <Button type="button" onClick={() => revokeDevice(d.name)} disabled={busy}>Revoke credentials</Button>
            </div>
          </div>
        ))}
        </div>
      </div>
      <Button onClick={registerMFADevice} disabled={busy}>Register MFA Device</Button>
    </div>
  )
}