import { useEffect, useState } from 'react'
import { CheckCircle, XCircle, AlertTriangle, X } from 'lucide-react'
import { cn } from '@/lib/utils'

type SnackbarType = 'success' | 'error' | 'warning'

interface SnackbarProps {
  message: string
  type?: SnackbarType
  isOpen: boolean
  onClose: () => void
  duration?: number
}

const icons = {
  success: CheckCircle,
  error: XCircle,
  warning: AlertTriangle,
}

const colors = {
  success: 'bg-success text-white',
  error: 'bg-danger text-white',
  warning: 'bg-warning text-white',
}

export function Snackbar({ message, type = 'success', isOpen, onClose, duration = 3000 }: SnackbarProps) {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    if (isOpen) {
      setVisible(true)
      const timer = setTimeout(() => {
        setVisible(false)
        setTimeout(onClose, 300)
      }, duration)
      return () => clearTimeout(timer)
    }
    setVisible(false)
  }, [isOpen, duration, onClose])

  if (!isOpen && !visible) return null

  const Icon = icons[type]

  return (
    <div
      className={cn(
        'fixed bottom-20 left-1/2 -translate-x-1/2 w-[343px] rounded-[12px] px-4 py-3 flex items-center gap-3 shadow-lg z-50 transition-all duration-300',
        colors[type],
        visible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-4'
      )}
    >
      <Icon className="h-5 w-5 shrink-0" />
      <span className="text-sm font-medium flex-1">{message}</span>
      <button onClick={() => { setVisible(false); setTimeout(onClose, 300) }}>
        <X className="h-4 w-4" />
      </button>
    </div>
  )
}
