import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { AlertTriangle, CheckCircle, XCircle, RefreshCw, Bell } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'

const typeConfig: Record<string, { icon: typeof Bell; color: string }> = {
  low_stock: { icon: AlertTriangle, color: 'text-danger bg-danger/10' },
  supply_approved: { icon: CheckCircle, color: 'text-success bg-success/10' },
  supply_rejected: { icon: XCircle, color: 'text-danger bg-danger/10' },
  sync_failed: { icon: RefreshCw, color: 'text-warning bg-warning/10' },
}

export function NotificationsPage() {
  const queryClient = useQueryClient()

  const { data: notifications = [] } = useQuery({
    queryKey: ['notifications'],
    queryFn: () => api.get('/notifications').then((r) => r.data),
  })

  const markRead = useMutation({
    mutationFn: (id: string) => api.post(`/notifications/${id}/read`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  })

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="Notifications" showBack />

      <main className="flex-1 px-4 pt-4 pb-20 space-y-2">
        {notifications.map((n: any) => {
          const config = typeConfig[n.type] || { icon: Bell, color: 'text-gray bg-bg-alt' }
          const Icon = config.icon
          return (
            <button
              key={n.id}
              onClick={() => !n.is_read && markRead.mutate(n.id)}
              className={cn(
                'w-full bg-white rounded-[12px] p-4 shadow-sm text-left transition-opacity',
                n.is_read && 'opacity-60'
              )}
            >
              <div className="flex items-start gap-3">
                <div className={cn('w-10 h-10 rounded-full flex items-center justify-center shrink-0', config.color)}>
                  <Icon className="h-5 w-5" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-semibold text-dark">{n.title}</p>
                    {!n.is_read && <div className="w-2 h-2 rounded-full bg-primary shrink-0" />}
                  </div>
                  <p className="text-xs text-gray mt-0.5 line-clamp-2">{n.message}</p>
                  <p className="text-[10px] text-gray-light mt-1">
                    {new Date(n.created_at).toLocaleString()}
                  </p>
                </div>
              </div>
            </button>
          )
        })}

        {notifications.length === 0 && (
          <div className="text-center py-12">
            <Bell className="h-12 w-12 text-gray-light mx-auto mb-3" />
            <p className="text-sm text-gray">No notifications</p>
          </div>
        )}
      </main>

      <Tabbar />
    </div>
  )
}
