import { ChevronLeft, Bell, ChevronDown } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'

interface HeaderProps {
  title: string
  subtitle?: string
  showBack?: boolean
  showNotification?: boolean
  badgeCount?: number
  onSubtitleClick?: () => void
  className?: string
}

export function Header({ title, subtitle, showBack = false, showNotification = false, badgeCount = 0, onSubtitleClick, className }: HeaderProps) {
  const navigate = useNavigate()

  return (
    <header className={cn('flex h-14 items-center justify-between px-4 bg-white', className)}>
      <div className="flex items-center gap-2">
        {showBack && (
          <button onClick={() => navigate(-1)} className="p-1 -ml-1">
            <ChevronLeft className="h-6 w-6 text-dark" />
          </button>
        )}
        <div>
          <h1 className="text-lg font-semibold text-dark">{title}</h1>
          {subtitle && (
            <button onClick={onSubtitleClick} className="flex items-center gap-1 -mt-0.5">
              <span className="text-xs text-primary font-medium">{subtitle}</span>
              {onSubtitleClick && <ChevronDown className="h-3 w-3 text-primary" />}
            </button>
          )}
        </div>
      </div>
      {showNotification && (
        <button onClick={() => navigate('/notifications')} className="p-1 relative">
          <Bell className="h-6 w-6 text-dark" />
          {badgeCount > 0 && (
            <span className="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 rounded-full bg-danger text-white text-[10px] font-bold flex items-center justify-center px-1">
              {badgeCount > 99 ? '99+' : badgeCount}
            </span>
          )}
        </button>
      )}
    </header>
  )
}
