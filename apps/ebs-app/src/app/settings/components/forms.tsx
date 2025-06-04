'use client'

import { Button } from '@/components/ui/button'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { use, useCallback, useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { Separator } from '@/components/ui/separator'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { ExclamationTriangleIcon } from '@radix-ui/react-icons'
import { Organization } from '@/lib/types'

const formSchema = z.object({
  name: z.string().min(2),
  email: z.string().email(),
  about: z.string().optional(),
  currency: z.string().optional(),
  country: z.string().optional(),
})

type Props = {
  resolver: Promise<Organization | undefined>,
}
export default function SettingsGeneralForm({ resolver }: Props) {
  const organization = use(resolver)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const [editMode, setEditMode] = useState(false)
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: organization?.name,
      email: organization?.email,
      about: organization?.about,
      country: organization?.country,
      currency: 'USD',
    },
  })

  const formSubmit = useCallback(async (data: z.infer<typeof formSchema>) => {
    setError(undefined)
    setBusy(true)
    form.control._disableForm(true)
  }, [form])

  return (
    <Form {...form}>
      {error &&
      <Alert variant="destructive">
        <ExclamationTriangleIcon className="size-4" />
        <AlertTitle className="text-md">An error occurred</AlertTitle>
        <AlertDescription className="text-lg">{ error }</AlertDescription>
      </Alert>
      }
      <p><span className="text-red-600">*</span> <span className="text-gray-400 text-sm">indicates required field</span></p>
      <form onSubmit={form.handleSubmit(formSubmit)} autoComplete="off" className="space-y-4 mt-5 grid grid-cols-2">
        <FormField
          control={form.control}
          name="email"
          render={({ field }) => (
            <FormItem className="w-full my-4 col-span-2">
              <FormLabel htmlFor="email">Email<span className="text-red-600">*</span></FormLabel>
              <div className="flex flex-row gap-2">
                <FormControl>
                  <Input
                    id="email"
                    type="email"
                    placeholder="Provide an email address"
                    {...field}
                    readOnly={!editMode}
                  />
                </FormControl>
                <Button>Verify email</Button>
              </div>
              <FormDescription>Contact email for this organization</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem className="w-full col-span-2 my-4">
              <FormLabel htmlFor="name">Name<span className="text-red-600">*</span></FormLabel>
              <FormControl>
                <Input
                  id="name"
                  type="text"
                  placeholder=""
                  {...field}
                  readOnly
                />
              </FormControl>
              <FormDescription>The name of your organization.</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="about"
          render={({ field }) => (
            <FormItem className="w-full col-span-2 my-4">
              <FormLabel htmlFor="about">About</FormLabel>
              <FormControl>
                <Textarea
                  id="about"
                  {...field}
                  readOnly={!editMode}
                  maxLength={500}
                  rows={10}
                />
              </FormControl>
              <FormDescription>Short introduction about this organization</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <Separator className="col-span-2" />
        <div className="flex items-center gap-2 col-span-2">
          {editMode ? 
          <Button type="submit" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none" disabled={!editMode || !form.formState.isValid || busy}>Save Changes</Button> :
          <Button onClick={() => setEditMode(true)}>Edit</Button>
          }
        </div>
      </form>
    </Form>
  )
}