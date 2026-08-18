import { useCallback, useEffect, useRef, useState } from 'react'

interface AutosaveOptions<T, R> {
  save: (value: T) => Promise<R>
  onSaved: (value: R) => void
  onError: (error: Error) => void
  delay?: number
}

export function useDebouncedAutosave<T, R>({ save, onSaved, onError, delay = 650 }: AutosaveOptions<T, R>) {
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const pending = useRef<T | undefined>(undefined)
  const running = useRef(false)
  const mounted = useRef(true)
  const saveRef = useRef(save)
  const savedRef = useRef(onSaved)
  const errorRef = useRef(onError)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    saveRef.current = save
    savedRef.current = onSaved
    errorRef.current = onError
  }, [onError, onSaved, save])

  useEffect(() => () => {
    mounted.current = false
    if (timer.current) clearTimeout(timer.current)
  }, [])

  const flush = useCallback(async () => {
    if (running.current) return
    running.current = true
    if (mounted.current) setSaving(true)
    while (pending.current !== undefined) {
      const value = pending.current
      pending.current = undefined
      try {
        const result = await saveRef.current(value)
        if (mounted.current) savedRef.current(result)
      } catch (error) {
        pending.current = undefined
        if (mounted.current) errorRef.current(error instanceof Error ? error : new Error(String(error)))
      }
    }
    running.current = false
    if (mounted.current) setSaving(false)
  }, [])

  const schedule = useCallback((value: T, immediate = false) => {
    pending.current = value
    if (timer.current) clearTimeout(timer.current)
    if (immediate) {
      void flush()
      return
    }
    timer.current = setTimeout(() => void flush(), delay)
  }, [delay, flush])

  return { schedule, saving }
}
