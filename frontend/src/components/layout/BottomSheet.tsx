import { useEffect, useState, type ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface BottomSheetProps {
  isOpen: boolean
  onClose: () => void
  title?: string
  children: ReactNode
  className?: string
}

const ANIM_MS = 300

export function BottomSheet({ isOpen, onClose, title, children, className }: BottomSheetProps) {
  // `mounted` controls DOM presence, `visible` drives translate+opacity animation
  const [mounted, setMounted] = useState(isOpen)
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    if (isOpen) {
      setMounted(true)
      // next frame: slide in
      requestAnimationFrame(() => {
        requestAnimationFrame(() => setVisible(true))
      })
    } else if (mounted) {
      // slide out, then unmount
      setVisible(false)
      const t = setTimeout(() => setMounted(false), ANIM_MS)
      return () => clearTimeout(t)
    }
  }, [isOpen, mounted])

  useEffect(() => {
    if (mounted) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = ''
    }
    return () => {
      document.body.style.overflow = ''
    }
  }, [mounted])

  if (!mounted) return null

  return (
    <>
      <div
        className="fixed inset-0 bg-overlay z-40 transition-opacity duration-300 ease-out"
        style={{ opacity: visible ? 1 : 0 }}
        onClick={onClose}
      />
      <div
        className={cn(
          'fixed bottom-0 inset-x-0 mx-auto w-full max-w-[375px] bg-white rounded-t-[24px] z-50',
          'transition-transform duration-300 ease-out will-change-transform',
          className
        )}
        style={{
          transform: visible ? 'translateY(0)' : 'translateY(100%)',
        }}
      >
        <div className="flex justify-center pt-3 pb-2">
          <div className="w-10 h-1 rounded-full bg-bg-alt" />
        </div>
        {title && (
          <div className="px-4 pb-3 border-b border-bg-alt">
            <h2 className="text-lg font-semibold text-center">{title}</h2>
          </div>
        )}
        <div className="px-4 py-4 max-h-[70vh] overflow-y-auto">
          {children}
        </div>
      </div>
    </>
  )
}
