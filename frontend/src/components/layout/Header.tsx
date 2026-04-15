import { ChevronLeft, ChevronDown } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'

// TEMP: notifications hidden across the project until the feature is ready.
// Flip back to `true` (or delete this flag and uncomment the bell code below)
// when notifications/center is finished.
const NOTIFICATIONS_ENABLED = false

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
      {/* Bell hidden globally — see NOTIFICATIONS_ENABLED at top of file. */}
      {NOTIFICATIONS_ENABLED && showNotification && badgeCount >= 0 && (
        <button onClick={() => navigate('/notifications')} className="p-1 relative">
          {/* re-enable: import { Bell } and render <Bell className="h-6 w-6 text-dark" /> + badge */}
        </button>
      )}
    </header>
  )
}
