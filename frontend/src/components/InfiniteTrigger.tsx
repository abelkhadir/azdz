'use client'

import { useEffect, useRef } from 'react'

export default function InfiniteTrigger({ hasMore, loading, onLoad }: { hasMore: boolean; loading: boolean; onLoad: () => void }) {
  const target = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const element = target.current
    if (!element || !hasMore) return
    const observer = new IntersectionObserver(entries => {
      if (entries[0].isIntersecting && !loading) onLoad()
    }, { rootMargin: '200px' })
    observer.observe(element)
    return () => observer.disconnect()
  }, [hasMore, loading, onLoad])

  return <div ref={target} className="infinite-trigger">{loading ? 'Loading…' : ''}</div>
}

