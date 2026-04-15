import { cn } from '@/lib/utils'

interface PeriodPillsProps {
  value: number
  onChange: (days: number) => void
  options?: number[]
  className?: string
}

/** Reusable 7 / 30 / 90 day period selector used across all charts. */
export function PeriodPills({
  value,
  onChange,
  options = [7, 30, 90],
  className,
}: PeriodPillsProps) {
  return (
    <div className={cn('flex bg-bg rounded-full p-1', className)}>
      {options.map((d) => (
        <button
          key={d}
          onClick={() => onChange(d)}
          className={cn(
            'flex-1 py-2 text-sm font-semibold rounded-full transition-colors',
            value === d ? 'bg-primary text-dark' : 'text-dark'
          )}
        >
          {d} days
        </button>
      ))}
    </div>
  )
}
