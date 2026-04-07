import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

interface FilterChipProps {
  label: string
  active?: boolean
  onRemove?: () => void
  onClick?: () => void
  className?: string
}

export function FilterChip({ label, active = true, onRemove, onClick, className }: FilterChipProps) {
  return (
    <button
      onClick={onClick || onRemove}
      className={cn(
        'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium transition-colors',
        active ? 'bg-primary/10 text-primary' : 'bg-bg-alt text-gray',
        className
      )}
    >
      {label}
      {onRemove && active && <X className="h-3 w-3" />}
    </button>
  )
}
