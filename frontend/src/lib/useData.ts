'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { errorMessage } from './api'

export function useData<T>(loader: () => Promise<T>, key: unknown = '') {
  const [state, setState] = useState<{ data: T | null; error: string }>({ data: null, error: '' })
  const loaderRef = useRef(loader)

  useEffect(() => { loaderRef.current = loader }, [loader])

  // Pages use the same function for their first load and later refreshes.
  const reload = useCallback(async () => {
    try {
      setState({ data: await loaderRef.current(), error: '' })
    } catch (error) {
      setState({ data: null, error: errorMessage(error) })
    }
  }, [])

  useEffect(() => { void reload() }, [key, reload])
  return { ...state, reload }
}

