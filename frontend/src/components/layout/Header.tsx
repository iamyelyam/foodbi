import { ChevronLeft, Bell } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'

interface HeaderProps {
  title: string
  showBack?: boolean
  showNotification?: boolean
  className?: string
}

export function Header({ title, showBack = false, showNotification = false, className }: HeaderProps) {
  const navigate = useNavigate()

  return (
    <header className={cn('flex h-14 items-center justify-between px-4 bg-white', className)}>
      <div className="flex items-center gap-2">
        {showBack && (
          <button onClick={() => navigate(-1)} className="p-1 -ml-1">
            <ChevronLeft className="h-6 w-6 text-dark" />
          </button>
        )}
        <h1 className="text-lg font-semibold text-dark">{title}</h1>
      </div>
      {showNotification && (
        <button onClick={() => navigate('/notifications')} className="p-1">
          <Bell className="h-6 w-6 text-dark" />
        </button>
      )}
    </header>
  )
}
