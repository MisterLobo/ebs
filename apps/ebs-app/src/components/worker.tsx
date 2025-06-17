'use client'

import { useEffect, useRef } from 'react'

export function WebWorker() {
  const workerRef = useRef<Worker>(null)
  useEffect(() => {
    if (!workerRef.current) {
      workerRef.current = new Worker(new URL('../lib/worker.ts', import.meta.url))
    }
    workerRef.current.onmessage = (event: MessageEvent<any>) => {
      alert(`message from worker: ${event.data}`)
    }
    workerRef.current.postMessage({ ping: true })
    return () => {
      workerRef.current?.terminate()
    }
  }, [workerRef.current])
  return <></>
}