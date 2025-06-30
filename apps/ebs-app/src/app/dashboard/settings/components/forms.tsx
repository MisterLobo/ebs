'use client'

import { Button } from '@/components/ui/button'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { use, useCallback, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { Separator } from '@/components/ui/separator'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { ExclamationTriangleIcon } from '@radix-ui/react-icons'
import { Country, Organization } from '@/lib/types'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { CheckIcon, ChevronsUpDownIcon } from 'lucide-react'
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command'
import { cn } from '@/lib/utils'

const formSchema = z.object({
  name: z.string().min(2),
  email: z.string().email(),
  about: z.string().optional(),
  currency: z.string().optional(),
  country: z.string().optional(),
})

type Props = {
  organizationResolver: Promise<Organization | undefined>,
  countriesResolver: Promise<Country[]>,
}
export default function SettingsGeneralForm({ organizationResolver, countriesResolver }: Props) {
  const organization = use(organizationResolver)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const [editMode, setEditMode] = useState(false)
  const countries = use(countriesResolver)
  const [country, setCountry] = useState<string>()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState<string>()
  const searchResults = useMemo(() => {
    if (search) {
      return countries.filter(c => c.cca2?.localeCompare(search) !== -1)
    }
    return countries.slice(0, 10)
  }, [search, countries])
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

  const discardChanges = useCallback(() => {
    form.reset()
    setEditMode(false)
  }, [])

  const selectCountry = useCallback((value: string) => {
    if (value !== country) {
      const c = countries.find(c => c.cca2 === value)
      form.setValue('country', c?.cca2)
      setCountry(c?.cca2)
    }
    setSearch(undefined)
    setOpen(false)
  }, [country])

  const onOpenChange = useCallback((open: boolean) => {
    setSearch(undefined)
    setOpen(open)
  }, [])

  const onInput = useCallback((value: string) => {
    setSearch(value)
  }, [])

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
              <FormLabel htmlFor="email">Contact Email<span className="text-red-600">*</span></FormLabel>
              <div className="flex flex-row gap-2">
                <FormControl>
                  <Input
                    id="email"
                    type="email"
                    placeholder="Provide an email address"
                    {...field}
                    readOnly
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
          name="country"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Country</FormLabel>
              <FormControl>
                <Popover open={open} onOpenChange={onOpenChange}>
                  <PopoverTrigger asChild>
                    <FormControl>
                      <Button variant="outline" role="combobox" className="w-64 justify-between" disabled={!editMode}>
                        {field.value ? countries.find(c => c.cca2 === field.value)?.name?.common : 'Select country'}
                        <ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                      </Button>
                    </FormControl>
                  </PopoverTrigger>
                  <PopoverContent className="w-64 p-0">
                    <Command>
                      <CommandInput placeholder="Search countries by code" onInput={e => onInput(e.currentTarget.value)} />
                      <CommandList>
                        <CommandEmpty>No results</CommandEmpty>
                        <CommandGroup>
                          {searchResults.map(c => (
                            <CommandItem
                              key={c.cca2}
                              value={c.cca2}
                              onSelect={v => selectCountry(v)}
                              className="cursor-pointer"
                              disabled={c.cca2 === country}
                            >
                              <CheckIcon
                                className={cn(
                                  'mr-2 h-4 w-4',
                                  country === c.cca2 ? 'opacity-100' : 'opacity-0',
                                )}
                              />
                              {c.flag}
                              {c.name?.common}
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
          {editMode ? (
            <>
            <Button type="submit" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none" disabled={!editMode || !form.formState.isValid || busy}>Save Changes</Button>
            <Button variant="link" type="button" className="cursor-pointer disabled:opacity-50 disabled:pointer-events-none" disabled={!editMode || busy} onClick={() => discardChanges()}>Discard</Button>
            </>
          ) :
          <Button onClick={() => setEditMode(true)}>Edit</Button>
          }
        </div>
      </form>
    </Form>
  )
}