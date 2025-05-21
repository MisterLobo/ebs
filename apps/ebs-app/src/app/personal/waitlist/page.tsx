import { getWaitlist } from '@/lib/actions'
import { AlertCircleIcon, BellIcon } from 'lucide-react'
import WaitlistItem from './components/waitlist-item'

export default async function WaitlistPage() {
  const { data, count, error } = await getWaitlist()
  
  return (
    <div className="m-6">
      <h1 className="text-xl flex items-center gap-2"><BellIcon /> Notify me of upcoming events</h1>
      {error &&
      <div className="flex w-full items-center rounded border bg-red-800 my-4 p-2 max-w-2xl">
        <div className="inline-flex w-fit mr-2"><AlertCircleIcon /></div>
        <h2 className="text-xl">{ error }</h2>
      </div>
      }
      {count === 0 ?
      <div className="flex flex-col">
        <div className="flex w-full items-center justify-center my-2">
          <BellIcon size={64} />
        </div>
        <p className="text-center italic text-neutral-300 text-xl">Your waitlist is empty</p>
      </div> :
      <>
      <h2 className="text-xl">Subscription: { data?.length }</h2>
      <div className="flex flex-col gap-4 items-center">
      {data.map((res: any, index: number) => (
        <WaitlistItem key={index} data={res} />
      ))}
      </div>
      </>
      }
    </div>
  )
}