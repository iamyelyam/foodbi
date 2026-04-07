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
    <div className={cn('flex bg-bg-alt rounded-[12px] p-1', className)}>
      {options.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onChange(opt.value)}
          className={cn(
            'flex-1 py-2 text-sm font-medium rounded-[10px] transition-colors',
            value === opt.value ? 'bg-white text-dark shadow-sm' : 'text-gray'
          )}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}
