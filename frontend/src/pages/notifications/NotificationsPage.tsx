import { useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { AlertTriangle, CheckCircle, XCircle, RefreshCw, Bell } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { useT, useI18nStore } from '@/i18n'

const typeConfig: Record<string, { icon: typeof Bell; color: string }> = {
  low_stock: { icon: AlertTriangle, color: 'text-danger bg-danger/10' },
  supply_approved: { icon: CheckCircle, color: 'text-success bg-success/10' },
  supply_rejected: { icon: XCircle, color: 'text-danger bg-danger/10' },
  sync_failed: { icon: RefreshCw, color: 'text-warning bg-warning/10' },
}

function getDateGroup(dateStr: string): 'today' | 'yesterday' | 'earlier' {
  const date = new Date(dateStr)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)

  const d = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  if (d.getTime() === today.getTime()) return 'today'
  if (d.getTime() === yesterday.getTime()) return 'yesterday'
  return 'earlier'
}

export function NotificationsPage() {
  const t = useT()
  const locale = useI18nStore((s) => s.locale)
  const queryClient = useQueryClient()

  const { data: notifications = [], isLoading } = useQuery({
    queryKey: ['notifications'],
    queryFn: () => api.get('/notifications').then((r) => r.data),
  })

  const markRead = useMutation({
    mutationFn: (id: string) => api.post(`/notifications/${id}/read`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
      queryClient.invalidateQueries({ queryKey: ['notifications-unread-count'] })
    },
  })

  const markAllRead = useMutation({
    mutationFn: () => api.post('/notifications/read-all'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
      queryClient.invalidateQueries({ queryKey: ['notifications-unread-count'] })
    },
  })

  const hasUnread = notifications.some((n: any) => !n.is_read)

  const grouped = useMemo(() => {
    const groups: { label: string; items: any[] }[] = []
    const groupMap = new Map<string, any[]>()
    const order = ['today', 'yesterday', 'earlier']

    for (const n of notifications) {
      const label = getDateGroup(n.created_at)
      if (!groupMap.has(label)) groupMap.set(label, [])
      groupMap.get(label)!.push(n)
    }

    for (const label of order) {
      const items = groupMap.get(label)
      if (items && items.length > 0) {
        groups.push({ label, items })
      }
    }

    return groups
  }, [notifications])

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={t('notifications.title')} showBack />

      {hasUnread && (
        <div className="px-4 pt-2 flex justify-end">
          <button
            onClick={() => markAllRead.mutate()}
            disabled={markAllRead.isPending}
            className="text-xs font-semibold text-primary"
          >
            {markAllRead.isPending ? '...' : t('notifications.markAllRead')}
          </button>
        </div>
      )}

      <main className="flex-1 px-4 pt-2 pb-20">
        {isLoading ? (
          <div className="space-y-2 mt-4">
            <ListItemSkeleton />
            <ListItemSkeleton />
            <ListItemSkeleton />
          </div>
        ) : (
        <>
        {grouped.map((group) => (
          <div key={group.label}>
            <p className="text-xs font-semibold text-gray uppercase tracking-wide mb-2 mt-4">
              {t(`common.${group.label}`)}
            </p>
            <div className="space-y-2">
              {group.items.map((n: any) => {
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
                          {new Date(n.created_at).toLocaleString(locale)}
                        </p>
                      </div>
                    </div>
                  </button>
                )
              })}
            </div>
          </div>
        ))}

        {notifications.length === 0 && (
          <div className="text-center py-12">
            <Bell className="h-12 w-12 text-gray-light mx-auto mb-3" />
            <p className="text-sm text-gray">{t('notifications.noNotifications')}</p>
          </div>
        )}
        </>
        )}
      </main>

      <Tabbar />
    </div>
  )
}
