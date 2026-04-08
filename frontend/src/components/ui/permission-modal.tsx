import { Camera, FolderOpen, Bell } from 'lucide-react'
import { Modal } from '@/components/ui/modal'
import { cn } from '@/lib/utils'

interface PermissionModalProps {
  isOpen: boolean
  onAllow: () => void
  onDeny: () => void
  type: 'camera' | 'files' | 'notifications'
}

const config = {
  camera: {
    icon: Camera,
    color: 'bg-primary-lighter',
    iconColor: 'text-primary',
    title: 'Allow Camera Access',
    description:
      'We need camera access to scan invoices and receipts. Your photos are only used for document processing.',
  },
  files: {
    icon: FolderOpen,
    color: 'bg-info-light',
    iconColor: 'text-info',
    title: 'Allow File Access',
    description:
      'File access lets you upload invoices, receipts, and other documents directly from your device.',
  },
  notifications: {
    icon: Bell,
    color: 'bg-warning/10',
    iconColor: 'text-warning',
    title: 'Enable Notifications',
    description:
      'Get notified when your files are processed and when new insights are available.',
  },
} as const

export function PermissionModal({ isOpen, onAllow, onDeny, type }: PermissionModalProps) {
  const { icon: Icon, color, iconColor, title, description } = config[type]

  return (
    <Modal isOpen={isOpen} onClose={onDeny}>
      <div className="flex flex-col items-center text-center">
        <div
          className={cn(
            'w-16 h-16 rounded-full flex items-center justify-center mb-4',
            color
          )}
        >
          <Icon className={cn('h-8 w-8', iconColor)} />
        </div>

        <h3 className="text-lg font-semibold text-dark mb-2">{title}</h3>
        <p className="text-sm text-gray leading-relaxed mb-6">{description}</p>

        <button
          onClick={onAllow}
          className="w-full h-12 bg-primary text-white rounded-[12px] text-sm font-semibold"
        >
          Allow
        </button>
        <button
          onClick={onDeny}
          className="mt-3 text-sm font-medium text-gray"
        >
          Not Now
        </button>
      </div>
    </Modal>
  )
}
