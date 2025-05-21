'use client'

import { Button } from '@/components/ui/button'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { useCallback, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { createTicket } from '@/lib/actions'
import { useParams, useRouter } from 'next/navigation'
import { Event } from '@/lib/types'

const formSchema = z.object({
  type: z.string(),
  tier: z.string().min(1),
  currency: z.string().min(3),
  price: z.coerce.number().gt(0),
  limited: z.coerce.boolean().optional(),
  limit: z.coerce.number(),
  event: z.number(),
})

type Props = {
  data?: Event,
}

export default function NewTicketForm({ data }: Props) {
  const params = useParams()
  const router = useRouter()
  const eventId = useMemo(() => parseInt(params.id as string), [params])
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      limited: true,
      type: 'standard',
      limit: 0,
      currency: 'USD',
      price: 0,
      event: eventId,
      tier: '',
    },
  })

  const formSubmit = useCallback(async (data: z.infer<typeof formSchema>) => {
    const formData = {
      ...data,
      currency: 'usd',
    }
    const { error } = await createTicket(formData)
    if (error) {
      console.error('error creating ticket: ', error)
      return
    }
    router.push(`/dashboard/events/${eventId}`)
  }, [form])

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(formSubmit)} autoComplete="off" className="space-y-4 min-w-96 w-xl">
        <h2 className="text-2xl">{ data?.title }</h2>
        <h2 className="text-sm">Event ID: <span className="font-semibold">{ eventId }</span></h2>
        <FormField
          control={form.control}
          name="tier"
          render={({ field }) => (
            <FormItem className="w-full">
              <FormLabel htmlFor="tier">Tier<span className="text-red-600">*</span></FormLabel>
              <FormControl>
                <Input
                  id="tier"
                  type="text"
                  placeholder="Name of the ticket. E.g VIP"
                  {...field}
                />
              </FormControl>
              <FormDescription>Name of the ticket. E.g VIP</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="price"
          render={({ field }) => (
            <FormItem className="w-full">
              <FormLabel htmlFor="price">Price (USD)<span className="text-red-600">*</span></FormLabel>
              <FormControl>
                <Input
                  id="price"
                  type="number"
                  placeholder="The cost of the ticket"
                  {...field}
                />
              </FormControl>
              <FormDescription>The cost of the ticket</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="type"
          render={({ field }) => (
            <FormItem className="w-full">
              <FormLabel htmlFor="type">Type<span className="text-red-600">*</span></FormLabel>
              <FormControl>
                <Input
                  id="type"
                  type="text"
                  placeholder="Ticket type. Standard for now"
                  {...field}
                  value="standard"
                  readOnly
                />
              </FormControl>
              <FormDescription>Ticket type. Standard for now</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="limit"
          render={({ field }) => (
            <FormItem className="w-full">
              <FormLabel htmlFor="limit">Limit<span className="text-red-600">*</span></FormLabel>
              <FormControl>
                <Input
                  id="type"
                  type="number"
                  className="w-24"
                  {...field}
                />
              </FormControl>
              <FormDescription>Ticket limit</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        {/* <FormField
          control={form.control}
          name="limited"
          render={({ field }) => (
            <FormItem className="w-full">
              <FormLabel htmlFor="limited">Limited</FormLabel>
              <FormControl>
                <Checkbox
                  id="limited"
                  {...field}
                  value={field.value as string}
                />
              </FormControl>
              <FormDescription>Set limit for ticket reservations</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        /> */}
        <Button type="submit" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none">SAVE</Button>
      </form>
    </Form>
  )
}