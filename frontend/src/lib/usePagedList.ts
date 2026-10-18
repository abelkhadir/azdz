'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { errorMessage, request } from './api'

const PAGE_SIZE = 10
const defaultPage = <T,>(value: unknown) => value as T[]

export function usePagedList<T extends { id: number }>(path: string, readPage: (value: unknown) => T[] = defaultPage<T>) {
  const [state, setState] = useState({ items: [] as T[], hasMore: true, loading: false, error: '' })
  const busy = useRef(false)
  const hasMore = useRef(true)
  const pendingReset = useRef(false)
  const loadedItems = useRef<T[]>([])

  const load = useCallback(async (reset = false) => {
    if (!path) {
      hasMore.current = false
      loadedItems.current = []
      setState({ items: [], hasMore: false, loading: false, error: '' })
      return
    }
    if (busy.current) {
      if (reset) pendingReset.current = true
      return
    }
    if (!reset && !hasMore.current) return
    busy.current = true
    setState(current => ({ ...current, loading: true, error: '' }))
    try {
      const before = !reset && loadedItems.current.length > 0 ? `&beforeId=${loadedItems.current[loadedItems.current.length - 1].id}` : ''
      const page = readPage(await request<unknown>(`${path}${path.includes('?') ? '&' : '?'}limit=${PAGE_SIZE}${before}`))
      hasMore.current = page.length === PAGE_SIZE
      setState(current => {
        const nextItems = reset ? page : [...current.items, ...page.filter(item => !current.items.some(old => old.id === item.id))]
        loadedItems.current = nextItems
        return { items: nextItems, hasMore: page.length === PAGE_SIZE, loading: false, error: '' }
      })
    } catch (error) {
      setState(current => ({ ...current, loading: false, error: errorMessage(error) }))
    } finally {
      busy.current = false
      if (pendingReset.current) {
        pendingReset.current = false
        void load(true)
      }
    }
  }, [path, readPage])
  const reset = useCallback(() => load(true), [load])
  useEffect(() => { loadedItems.current = []; hasMore.current = true; void load(true) }, [path, load])

  return { ...state, loadMore: () => load(false), reset }
}

