import { useEffect, type ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface BottomSheetProps {
  isOpen: boolean
  onClose: () => void
  title?: string
  children: ReactNode
  className?: string
}

export function BottomSheet({ isOpen, onClose, title, children, className }: BottomSheetProps) {
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = ''
    }
    return () => { document.body.style.overflow = '' }
  }, [isOpen])

  if (!isOpen) return null

  return (
    <>
      <div className="fixed inset-0 bg-overlay z-40" onClick={onClose} />
      <div
        className={cn(
          'fixed bottom-0 left-1/2 -translate-x-1/2 w-full max-w-[375px] bg-white rounded-t-[24px] z-50',
          'animate-[slideUp_0.3s_ease-out]',
          className
        )}
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
