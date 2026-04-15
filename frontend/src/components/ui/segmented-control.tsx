import { cn } from '@/lib/utils'

interface SegmentedControlProps<T extends string> {
  value: T
  onChange: (value: T) => void
  options: { value: T; label: string }[]
  className?: string
}

export function SegmentedControl<T extends string>({
  value,
  onChange,
  options,
  className,
}: SegmentedControlProps<T>) {
  return (
    <div className={cn('flex bg-bg-alt rounded-full p-1', className)}>
      {options.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onChange(opt.value)}
          className={cn(
            'flex-1 py-2.5 text-base font-medium rounded-full transition-colors',
            value === opt.value ? 'bg-primary text-black shadow-sm' : 'text-black'
          )}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}
