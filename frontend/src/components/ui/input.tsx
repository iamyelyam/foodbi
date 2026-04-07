import { forwardRef, type InputHTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
}

const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, error, ...props }, ref) => (
    <div className="flex flex-col gap-1.5">
      {label && (
        <label className="text-sm font-medium text-gray">{label}</label>
      )}
      <input
        className={cn(
          'h-12 w-full rounded-[12px] border border-bg-alt bg-white px-4 text-base text-dark',
          'placeholder:text-gray-light',
          'focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary',
          error && 'border-danger focus:border-danger focus:ring-danger',
          className
        )}
        ref={ref}
        {...props}
      />
      {error && <p className="text-sm text-danger">{error}</p>}
    </div>
  )
)
Input.displayName = 'Input'

export { Input }
