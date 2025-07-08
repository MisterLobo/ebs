'use client'

import { Button } from '@/components/ui/button'
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command'
import Filters from '@/components/ui/filters'
import { Form, FormControl, FormField, FormItem, FormLabel } from '@/components/ui/form'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { categories } from '@/lib/constants'
import { cn } from '@/lib/utils'
import { zodResolver } from '@hookform/resolvers/zod'
import { CheckIcon, ChevronsUpDownIcon } from 'lucide-react'
import { useSearchParams } from 'next/navigation'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

const searchSchema = z.object({
  location: z.string().optional(),
  category: z.string().optional(),
})
type Props = {
  category?: string,
}
export default function EventFiltersHeader({ }: Props) {
  const searchParams = useSearchParams()
  const [open, setOpen] = useState(false)
  const [category, setCategory] = useState<string>()
  const [search, setSearch] = useState<string>()
  const form = useForm<z.infer<typeof searchSchema>>({
    resolver: zodResolver(searchSchema),
  })
  const onOpenChange = (open: boolean) => {
    setOpen(open)
  }
  const searchResults = useMemo(() => {
    if (search) {
      return categories.filter(c => c.localeCompare(search) !== -1)
    }
    return categories
  }, [search, categories])
  const selectCategory = useCallback((value: string) => {
    if (value !== category) {
      const c = categories.find(c => c === value)
      form.setValue('category', c)
      setCategory(c as string)
    } else {
      form.reset()
      setCategory(undefined)
    }
    setSearch(undefined)
    setOpen(false)
  }, [category])
  useEffect(() => {
    const cat = searchParams.get('q')
    if (cat) {
      selectCategory(decodeURIComponent(cat))
    }
  }, [searchParams])

  return (
    <>
    <Form {...form}>
      <form className="my-10">
        <FormField
          control={form.control}
          name="category"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Category</FormLabel>
              <FormControl>
                <Popover open={open} onOpenChange={onOpenChange}>
                  <PopoverTrigger asChild>
                    <FormControl>
                      <Button variant="outline" role="combobox" className="w-96 justify-between">
                        {field.value ? categories.find(c => c === field.value) : 'Select category'}
                        <ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                      </Button>
                    </FormControl>
                  </PopoverTrigger>
                  <PopoverContent className="w-96 p-0">
                    <Command>
                      <CommandInput placeholder="Search categories" />
                      <CommandList>
                        <CommandEmpty>No results</CommandEmpty>
                        <CommandGroup>
                          {searchResults.map(c => (
                            <CommandItem
                              key={c}
                              value={c}
                              className="cursor-pointer"
                              onSelect={selectCategory}
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
      </form>
    </Form>
    <Filters filters={[]} setFilters={() => {}} />
    </>
  )
}