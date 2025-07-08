'use client'

import { Button } from '@/components/ui/button'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { createEvent, geocode, getActiveOrganization } from '@/lib/actions'
import { Geocoded, NewEventRequestPayload } from '@/lib/types'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { useRouter } from 'next/navigation'
import { format } from 'date-fns'
import { Checkbox } from '@/components/ui/checkbox'
import { Separator } from '@/components/ui/separator'
import { API_DATE_TIME, categories } from '@/lib/constants'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { ExclamationTriangleIcon } from '@radix-ui/react-icons'
import { CheckIcon, ChevronsUpDownIcon, Map } from 'lucide-react'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command'
import { cn } from '@/lib/utils'

const formSchema = z.object({
  name: z.string().min(2),
  title: z.string().min(2),
  about: z.string().optional(),
  location: z.string().min(2),
  date: z.string().date(),
  time: z.coerce.string(),
  deadline_date: z.string().date(),
  deadline_time: z.coerce.string(),
  seats: z.coerce.number(),
  scheduled: z.coerce.boolean().optional(),
  opens_at_date: z.string().date().optional(),
  opens_at_time: z.coerce.string().optional(),
  category: z.string().optional(),
  type: z.string().optional(),
})

type Props = {
  onboardingComplete?: boolean;
}
export function NewEventForm({ onboardingComplete }: Props) {
  const router = useRouter()
  const [busy, setBusy] = useState(false)
  const [scheduled, setScheduled] = useState(false)
  const [error, setError] = useState<string>()
  const [timezone, setTimezone] = useState<Geocoded>()
  const [types, setTypes] = useState<string[]>([
    'Online',
    'Virtual',
    'Physical',
  ])
  const [category, setCategory] = useState<string>()
  const [eventType, setEventType] = useState<string>()
  const [typeOpen, setTypeOpen] = useState(false)
  const [categoryOpen, setCategoryOpen] = useState(false)
  const [search, setSearch] = useState<string>()
  const searchResults = useMemo(() => {
    if (search) {
      return categories.filter(c => c?.localeCompare(search) !== -1)
    }
    return categories
  }, [])
  const { todayDate, todayTime } = useMemo(() => {
    const today = new Date()
    const todayDate = format(today, 'yyyy-MM-dd')
    const todayTime = format(today, 'HH:mm')
    return { todayDate, todayTime }
  }, [])
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      seats: 0,
      time: todayTime,
      date: todayDate,
      scheduled,
    },
  })

  useEffect(() => {
    if (navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(async (position) => {
        const lat = position.coords.latitude
        const long = position.coords.longitude
        const timezone = await geocode(lat, long)
        setTimezone(timezone)
      })
    }
  }, [])

  useEffect(() => {
    setTypes([
      'Online',
      'Virtual',
      'Physical',
    ])
    setCategory('General')
  }, [])

  const selectCategory = useCallback((value: string) => {
    if (value !== category) {
      const c = categories.find(c => c === value)
      form.setValue('category', c)
      setCategory(c)
    } else {
      form.setValue('category', undefined)
      setCategory(undefined)
    }
    setSearch(undefined)
    setCategoryOpen(false)
  }, [category])
  const onInputCategory = useCallback((value: string) => {
    setSearch(value)
  }, [])

  const selectType = useCallback((value: string) => {
    if (value !== eventType) {
      const t = types.find(t => t === value)
      form.setValue('type', t)
      setEventType(value)
    } else {
      form.setValue('type', undefined)
      setEventType(undefined)
    }
    setSearch(undefined)
    setTypeOpen(false)
  }, [eventType])
  const onInputType = useCallback((value: string) => {
    setSearch(value)
  }, [])

  const formSubmit = useCallback(async (data: z.infer<typeof formSchema>) => {
    setError(undefined)
    setBusy(true)
    form.control._disableForm(true)
    let when: string | undefined
    if (data.date && data.time) {
      const dt = new Date(`${data.date} ${data.time}`)
      when = format(dt, API_DATE_TIME)
    }

    let deadline: string | undefined
    if (data.deadline_date && data.deadline_time) {
      const dt = new Date(`${data.deadline_date} ${data.deadline_time}`)
      deadline = format(dt, API_DATE_TIME)
    }

    let opens_at: string | undefined
    if (data.opens_at_date && data.opens_at_time) {
      const dt = new Date(`${data.opens_at_date} ${data.opens_at_time}`)
      opens_at = format(dt, API_DATE_TIME)
    }
    const organization = await getActiveOrganization()
    const formData = {
      ...data,
      deadline,
      organization: organization?.id,
      opens_at,
      mode: scheduled ? 'scheduled' : 'default',
      date_time: when,
      timezone: timezone?.timeZoneId,
      type: eventType,
      category,
    } as NewEventRequestPayload

    const { error } = await createEvent(formData)
    if (error) {
      setError(error)
      form.control._disableForm(false)
      setBusy(false)
      console.error('[error]:', error)
      return
    }
    router.push('/dashboard/events')
  }, [form, scheduled])

  const createAndPublish = useCallback(async () => {
    form.trigger()
    const data = form.getValues()
    let when: string | undefined
    if (data.date && data.time) {
      const dt = new Date(`${data.date} ${data.time}`)
      when = format(dt, API_DATE_TIME)
    }

    let deadline: string | undefined
    if (data.deadline_date && data.deadline_time) {
      const dt = new Date(`${data.deadline_date} ${data.deadline_time}`)
      deadline = format(dt, API_DATE_TIME)
    }

    let opens_at: string | undefined
    if (data.opens_at_date && data.opens_at_time) {
      const dt = new Date(`${data.opens_at_date} ${data.opens_at_time}`)
      opens_at = format(dt, API_DATE_TIME)
    }
    setError(undefined)
    setBusy(true)
    form.control._disableForm(true)
    const organization = await getActiveOrganization()
    const formData = {
      ...data,
      deadline,
      organization: organization?.id,
      publish: true,
      opens_at,
      mode: scheduled ? 'scheduled' : 'default',
      date_time: when,
      timezone: timezone?.timeZoneId,
      type: eventType,
      category,
    } as NewEventRequestPayload
    
    const { error } = await createEvent(formData)
    if (error) {
      setError(error)
      form.control._disableForm(false)
      setBusy(false)
      console.error('[error]:', error)
      return
    }
    router.push('/dashboard/events')
  }, [form, scheduled])

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
      <form onSubmit={form.handleSubmit(formSubmit)} autoComplete="off" className="space-y-4 my-10 max-w-2xl">
        <FormField
          control={form.control}
          name="title"
          render={({ field }) => (
            <FormItem className="w-full">
              <FormLabel htmlFor="title">WHAT<span className="text-red-600">*</span></FormLabel>
              <FormControl>
                <Input
                  id="title"
                  type="text"
                  placeholder="Provide a title"
                  {...field}
                  autoComplete="off"
                />
              </FormControl>
              <FormDescription>Title of the event</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <div className="flex w-full gap-2 justify-between items-center">
          <FormField
            control={form.control}
            name="category"
            render={() => (
              <FormItem>
                <FormLabel>Category</FormLabel>
                <FormControl>
                  <Popover open={categoryOpen} onOpenChange={setCategoryOpen}>
                    <PopoverTrigger asChild className="inline-flex">
                      <FormControl>
                        <Button variant="outline" role="combobox" className="w-96 justify-between">
                          {category ?? 'Select category'}
                          <ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                        </Button>
                      </FormControl>
                    </PopoverTrigger>
                    <PopoverContent className="w-96 p-0">
                      <Command>
                        <CommandInput placeholder="Search category" onInput={e => onInputCategory(e.currentTarget.value)} />
                        <CommandList>
                          <CommandEmpty>No results</CommandEmpty>
                          <CommandGroup>
                            {searchResults.map(c => (
                              <CommandItem
                                key={c}
                                value={c}
                                onSelect={v => selectCategory(v)}
                                className="cursor-pointer"
                              >
                                <CheckIcon
                                  className={cn(
                                    'mr-2 h-4 w-4',
                                    category === c ? 'opacity-100' : 'opacity-0',
                                  )}
                                />
                                {c}
                              </CommandItem>
                            ))}
                          </CommandGroup>
                        </CommandList>
                      </Command>
                    </PopoverContent>
                  </Popover>
                </FormControl>
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="type"
            render={() => (
              <FormItem>
                <FormLabel>Event Type</FormLabel>
                <FormControl>
                  <Popover open={typeOpen} onOpenChange={setTypeOpen}>
                    <PopoverTrigger asChild className="inline-flex">
                      <FormControl>
                        <Button variant="outline" role="combobox" className="w-68 justify-between">
                          {eventType ?? 'Select event type'}
                          <ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                        </Button>
                      </FormControl>
                    </PopoverTrigger>
                    <PopoverContent className="w-68 p-0">
                      <Command>
                        <CommandInput placeholder="Search event type" onInput={e => onInputType(e.currentTarget.value)} />
                        <CommandList>
                          <CommandEmpty>No results</CommandEmpty>
                          <CommandGroup>
                            {types.map(t => (
                              <CommandItem
                                key={t}
                                value={t}
                                onSelect={v => selectType(v)}
                                className="cursor-pointer"
                              >
                                <CheckIcon
                                  className={cn(
                                    'mr-2 h-4 w-4',
                                    eventType === t ? 'opacity-100' : 'opacity-0',
                                  )}
                                />
                                {t}
                              </CommandItem>
                            ))}
                          </CommandGroup>
                        </CommandList>
                      </Command>
                    </PopoverContent>
                  </Popover>
                </FormControl>
              </FormItem>
            )}
          />
        </div>
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem className="w-full">
              <FormLabel htmlFor="name">Name<span className="text-red-600">*</span></FormLabel>
              <FormControl>
                <Input
                  id="name"
                  type="text"
                  placeholder=""
                  {...field}
                />
              </FormControl>
              <FormDescription>The name of the event. You can use this name multiple times for recurring events that happen at different dates and locations</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="about"
          render={({ field }) => (
            <FormItem className="w-full">
              <FormLabel htmlFor="about">About</FormLabel>
              <FormControl>
                <Textarea
                  id="about"
                  {...field}
                  value={field.value as string}
                />
              </FormControl>
              <FormDescription>About the event</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <div className="flex w-full items-center gap-2">
          <FormField
            control={form.control}
            name="location"
            render={({ field }) => (
              <FormItem className="w-full">
                <FormLabel htmlFor="location">WHERE<span className="text-red-600">*</span></FormLabel>
                <FormControl>
                  <Input
                    id="name"
                    type="text"
                    placeholder=""
                    {...field}
                  />
                </FormControl>
                <FormDescription>The venue where the event will take place</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <Button type="button" size="icon" variant="ghost" className="rounded-full"><Map /></Button>
        </div>
        <div className="flex flex-col">
          <h2 className="flex items-center gap-2 text-sm font-medium leading-none select-none">WHEN<span className="text-red-600">*</span></h2>
          <div className="flex w-full gap-2">
            <FormField
              control={form.control}
              name="date"
              render={({ field }) => (
                <FormItem className="w-fit">
                  <FormLabel htmlFor="date">Date</FormLabel>
                  <FormControl>
                    <Input
                      id="date"
                      type="date"
                      placeholder=""
                      className="w-fit"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>The date of the event</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="time"
              render={({ field }) => (
                <FormItem className="w-fit">
                  <FormLabel htmlFor="time">Time</FormLabel>
                  <FormControl>
                    <Input
                      id="time"
                      type="time"
                      placeholder=""
                      className="w-fit"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>The time of the event</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>
        <div className="flex flex-col">
          <h2 className="flex items-center gap-2 text-sm font-medium leading-none select-none">DEADLINE<span className="text-red-600">*</span></h2>
          <div className="flex w-full gap-2">
            <FormField
              control={form.control}
              name="deadline_date"
              render={({ field }) => (
                <FormItem className="w-fit">
                  <FormLabel htmlFor="deadline">Date</FormLabel>
                  <FormControl>
                    <Input
                      id="deadline"
                      type="date"
                      placeholder=""
                      className="w-fit"
                      {...field}
                      value={field.value as string}
                    />
                  </FormControl>
                  <FormDescription className="text-wrap w-32 break-words">The date when the event registration would close</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="deadline_time"
              render={({ field }) => (
                <FormItem className="w-fit">
                  <FormLabel htmlFor="time">Time</FormLabel>
                  <FormControl>
                    <Input
                      id="time"
                      type="time"
                      placeholder=""
                      className="w-fit"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription className="text-wrap w-32 break-words">The time when the event registration would close</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>
        <FormField
          control={form.control}
          name="scheduled"
          render={({ field: { onChange, value} }) => (
            <FormItem className="w-fit">
              <FormLabel>Scheduled opening</FormLabel>
              <FormControl>
                <div className="flex items-center space-x-2">
                  <Checkbox id="opensAt" checked={value} disabled={!onboardingComplete} onCheckedChange={e => {
                    onChange(e)
                    setScheduled(e as boolean)
                  }} />
                  <label htmlFor="opensAt" className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">Set schedule to open ticket reservations</label>
                </div>
              </FormControl>
            </FormItem>
          )}
        />
        {scheduled && <div className="flex flex-col">
          <h2 className="flex items-center gap-2 text-sm font-medium leading-none select-none">Opens at</h2>
          <div className="flex w-full gap-2">
            <FormField
              control={form.control}
              name="opens_at_date"
              render={({ field }) => (
                <FormItem className="w-fit">
                  <FormLabel htmlFor="opens_at_date">Date</FormLabel>
                  <FormControl>
                    <Input
                      id="opens_at_date"
                      type="date"
                      placeholder=""
                      className="w-fit"
                      {...field}
                      value={field.value as string}
                    />
                  </FormControl>
                  <FormDescription className="text-wrap w-32 break-words">The date when the event registration would open</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="opens_at_time"
              render={({ field }) => (
                <FormItem className="w-fit">
                  <FormLabel htmlFor="opens_at_time">Time</FormLabel>
                  <FormControl>
                    <Input
                      id="opens_at_time"
                      type="time"
                      placeholder=""
                      className="w-fit"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription className="text-wrap w-32 break-words">The time when the event registration would open</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>}
        <Separator />
        <div className="flex items-center gap-2">
          <Button type="submit" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none" disabled={!form.formState.isValid || busy}>CREATE</Button>
          <Button type="button" variant="secondary" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none" onClick={createAndPublish} disabled={!onboardingComplete || !form.formState.isValid || scheduled || busy}>CREATE AND PUBLISH</Button>
        </div>
      </form>
    </Form>
  )
}

/* function MapPreview() {
  return (
    <iframe
      style={{ border: 0 }}
      referrerPolicy="no-referrer-when-downgrade"
      loading="lazy"
      src={`https://www.google.com/maps/embed/v1/place?key=${process.env.NEXT_PUBLIC_GEOTZ_API_KEY}&q=${coords?.latitude} ${coords?.longitude}`}
      allowFullScreen>
    </iframe>
  )
} */