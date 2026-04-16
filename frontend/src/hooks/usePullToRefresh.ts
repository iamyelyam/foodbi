import { useRef, useEffect, useCallback, useState } from 'react'

interface UsePullToRefreshOptions {
  onRefresh: () => Promise<unknown>
  threshold?: number
}

export function usePullToRefresh({ onRefresh, threshold = 80 }: UsePullToRefreshOptions) {
  const [pulling, setPulling] = useState(false)
  const [pullDistance, setPullDistance] = useState(0)
  const [refreshing, setRefreshing] = useState(false)
  const startY = useRef(0)
  const active = useRef(false)

  const onTouchStart = useCallback((e: TouchEvent) => {
    if (window.scrollY === 0 && !refreshing) {
      startY.current = e.touches[0].clientY
      active.current = true
    }
  }, [refreshing])

  const onTouchMove = useCallback((e: TouchEvent) => {
    if (!active.current) return
    const dy = e.touches[0].clientY - startY.current
    if (dy > 0) {
      setPulling(true)
      setPullDistance(Math.min(dy * 0.5, threshold * 1.5))
    } else {
      setPulling(false)
      setPullDistance(0)
    }
  }, [threshold])

  const onTouchEnd = useCallback(async () => {
    if (!active.current) return
    active.current = false
    if (pullDistance >= threshold) {
      setRefreshing(true)
      setPullDistance(threshold * 0.6)
      try {
        await onRefresh()
      } finally {
        setRefreshing(false)
      }
    }
    setPulling(false)
    setPullDistance(0)
  }, [pullDistance, threshold, onRefresh])

  useEffect(() => {
    window.addEventListener('touchstart', onTouchStart, { passive: true })
    window.addEventListener('touchmove', onTouchMove, { passive: true })
    window.addEventListener('touchend', onTouchEnd)
    return () => {
      window.removeEventListener('touchstart', onTouchStart)
      window.removeEventListener('touchmove', onTouchMove)
      window.removeEventListener('touchend', onTouchEnd)
    }
  }, [onTouchStart, onTouchMove, onTouchEnd])

  return { pulling: pulling || refreshing, pullDistance, refreshing }
}
