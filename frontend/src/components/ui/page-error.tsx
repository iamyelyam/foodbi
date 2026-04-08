import { AlertTriangle } from 'lucide-react'
import { Button } from './button'

interface PageErrorProps {
  message?: string
  onRetry?: () => void
}

export function PageError({
  message = 'Failed to load data',
  onRetry,
}: PageErrorProps) {
  return (
    <div className="flex flex-col items-center justify-center py-12 px-6 text-center">
      <AlertTriangle className="h-10 w-10 text-gray-light mb-3" />
      <p className="text-sm text-gray mb-4">{message}</p>
      {onRetry && (
        <Button size="sm" variant="secondary" onClick={onRetry}>
          Retry
        </Button>
      )}
    </div>
  )
}
