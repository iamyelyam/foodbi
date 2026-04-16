import { useRef, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'

const ROOT_PATHS = ['/', '/login', '/register', '/register/employee', '/verify-otp', '/onboarding', '/forgot-password']

/**
 * Swipe from left edge → navigate(-1).
 * Disabled on root-level pages where there's nowhere to go back.
 */
export function useSwipeBack() {
  const navigate = useNavigate()
  const location = useLocation()
  const startX = useRef(0)
  const startY = useRef(0)
  const swiping = useRef(false)

  useEffect(() => {
    const isRoot = ROOT_PATHS.includes(location.pathname)

    function onTouchStart(e: TouchEvent) {
      if (isRoot) return
      const x = e.touches[0].clientX
      // Only trigger from the left 30px edge
      if (x <= 30) {
        startX.current = x
        startY.current = e.touches[0].clientY
        swiping.current = true
      }
    }

    function onTouchEnd(e: TouchEvent) {
      if (!swiping.current) return
      swiping.current = false
      const dx = e.changedTouches[0].clientX - startX.current
      const dy = Math.abs(e.changedTouches[0].clientY - startY.current)
      // Horizontal swipe: at least 80px right, and more horizontal than vertical
      if (dx > 80 && dx > dy * 1.5) {
        navigate(-1)
      }
    }

    window.addEventListener('touchstart', onTouchStart, { passive: true })
    window.addEventListener('touchend', onTouchEnd, { passive: true })
    return () => {
      window.removeEventListener('touchstart', onTouchStart)
      window.removeEventListener('touchend', onTouchEnd)
    }
  }, [navigate, location.pathname])
}
