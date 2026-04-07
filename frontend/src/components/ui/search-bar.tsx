import { forwardRef, type InputHTMLAttributes } from 'react'
import { Search, X } from 'lucide-react'
import { cn } from '@/lib/utils'

interface SearchBarProps extends InputHTMLAttributes<HTMLInputElement> {
  onClear?: () => void
}

const SearchBar = forwardRef<HTMLInputElement, SearchBarProps>(
  ({ className, value, onClear, ...props }, ref) => (
    <div className={cn('relative', className)}>
      <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-light" />
      <input
        ref={ref}
        value={value}
        className="h-10 w-full rounded-[10px] bg-bg-alt pl-9 pr-9 text-sm text-dark placeholder:text-gray-light focus:outline-none focus:ring-1 focus:ring-primary"
        {...props}
      />
      {value && onClear && (
        <button
          onClick={onClear}
          className="absolute right-3 top-1/2 -translate-y-1/2"
        >
          <X className="h-4 w-4 text-gray-light" />
        </button>
      )}
    </div>
  )
)
SearchBar.displayName = 'SearchBar'

export { SearchBar }
