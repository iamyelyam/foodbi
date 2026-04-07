import { forwardRef, type InputHTMLAttributes } from 'react'
import { Check } from 'lucide-react'
import { cn } from '@/lib/utils'

interface CheckboxProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> {
  label?: string
}

const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(
  ({ className, label, checked, ...props }, ref) => (
    <label className={cn('flex items-center gap-3 cursor-pointer', className)}>
      <div className="relative">
        <input ref={ref} type="checkbox" checked={checked} className="sr-only peer" {...props} />
        <div className={cn(
          'w-5 h-5 rounded-[6px] border-2 transition-colors flex items-center justify-center',
          checked ? 'bg-primary border-primary' : 'border-gray-light bg-white'
        )}>
          {checked && <Check className="h-3 w-3 text-white" strokeWidth={3} />}
        </div>
      </div>
      {label && <span className="text-sm text-dark">{label}</span>}
    </label>
  )
)
Checkbox.displayName = 'Checkbox'

export { Checkbox }
