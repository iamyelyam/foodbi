import { useEffect, type ReactNode } from 'react'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ModalProps {
  isOpen: boolean
  onClose: () => void
  title?: string
  children: ReactNode
  className?: string
}

export function Modal({ isOpen, onClose, title, children, className }: ModalProps) {
  useEffect(() => {
    if (isOpen) document.body.style.overflow = 'hidden'
    else document.body.style.overflow = ''
    return () => { document.body.style.overflow = '' }
  }, [isOpen])

  if (!isOpen) return null

  return (
    <>
      <div className="fixed inset-0 bg-overlay z-50" onClick={onClose} />
      <div className="fixed inset-0 z-50 flex items-center justify-center p-6">
        <div className={cn('bg-white rounded-[20px] w-full max-w-[311px] p-6 shadow-xl', className)}>
          {title && (
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-base font-semibold text-dark">{title}</h3>
              <button onClick={onClose}><X className="h-5 w-5 text-gray" /></button>
            </div>
          )}
          {children}
        </div>
      </div>
    </>
  )
}
