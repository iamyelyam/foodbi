import { cn } from '@/lib/utils'

interface ToggleProps {
  checked: boolean
  onChange: (checked: boolean) => void
  label?: string
  className?: string
}

export function Toggle({ checked, onChange, label, className }: ToggleProps) {
  return (
    <label className={cn('flex items-center justify-between cursor-pointer', className)}>
      {label && <span className="text-sm text-dark">{label}</span>}
      <button
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        className={cn(
          'relative w-[51px] h-[31px] rounded-full transition-colors',
          checked ? 'bg-primary' : 'bg-gray-light/40'
        )}
      >
        <div
          className={cn(
            'absolute top-[2px] w-[27px] h-[27px] rounded-full bg-white shadow transition-transform',
            checked ? 'translate-x-[22px]' : 'translate-x-[2px]'
          )}
        />
      </button>
    </label>
  )
}
