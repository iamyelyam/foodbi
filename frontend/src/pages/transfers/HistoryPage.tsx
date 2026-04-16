import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { CardSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { FilterChip } from '@/components/ui/filter-chip'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import { ArrowRightLeft, Filter, Clock } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { useT, useI18nStore } from '@/i18n'

export function HistoryPage() {
  const t = useT()
  const locale = useI18nStore((s) => s.locale)
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const [showFilters, setShowFilters] = useState(false)
  const [filters, setFilters] = useState<Record<string, string>>({})

  const { data: locations = [] } = useQuery({
    queryKey: ['locations'],
    queryFn: () => api.get('/locations').then((r) => r.data),
  })

  const { data: transfers = [], isLoading } = useQuery({
    queryKey: ['transfer-history', filters],
    queryFn: () => api.get('/transfers', { params: { ...filters, status: 'completed' } }).then((r) => r.data),
  })

  const getLocName = (id: string) => locations.find((l: any) => l.id === id)?.name || id.slice(0, 8)

  const activeFilters = Object.entries(filters).filter(([, v]) => v)

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={t('transfers.historyTitle')} showBack showNotification badgeCount={unreadCount} />

      <div className="px-4 pt-3 pb-2 flex items-center justify-between">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs text-gray">{t('transfers.countLabel', { count: transfers.length })}</span>
          {activeFilters.map(([key, val]) => (
            <FilterChip key={key} label={`${key}: ${val}`} onRemove={() => setFilters((f) => { const n = { ...f }; delete n[key]; return n })} />
          ))}
        </div>
        <button onClick={() => setShowFilters(true)} className="flex items-center gap-1 text-xs font-medium text-primary">
          <Filter className="h-3.5 w-3.5" /> {t('common.filter')}
        </button>
      </div>

      <main className="flex-1 px-4 pb-28 space-y-2">
        {isLoading ? (
          <>
            <CardSkeleton />
            <CardSkeleton />
            <CardSkeleton />
          </>
        ) : (
        <>
        {transfers.map((tr: any) => (
          <div key={tr.id} className="bg-white rounded-[12px] p-4 shadow-sm">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-info-light flex items-center justify-center">
                  <ArrowRightLeft className="h-5 w-5 text-info" />
                </div>
                <div>
                  <p className="text-sm font-semibold text-dark">
                    {getLocName(tr.from_location_id)} → {getLocName(tr.to_location_id)}
                  </p>
                  <div className="flex items-center gap-1 mt-0.5">
                    <Clock className="h-3 w-3 text-gray" />
                    <p className="text-xs text-gray">{new Date(tr.created_at).toLocaleDateString(locale)}</p>
                  </div>
                </div>
              </div>
              <span className={cn(
                'text-xs px-2 py-0.5 rounded-full font-medium',
                tr.status === 'completed' ? 'bg-success/10 text-success' : 'bg-gray/10 text-gray'
              )}>{tr.status === 'completed' ? t('transfers.statusCompleted') : tr.status}</span>
            </div>
          </div>
        ))}

        {transfers.length === 0 && (
          <EmptyState icon={ArrowRightLeft} title={t('transfers.noHistoryTitle')} description={t('transfers.noHistoryDesc')} />
        )}
        </>
        )}
      </main>

      <Tabbar />

      <BottomSheet isOpen={showFilters} onClose={() => setShowFilters(false)} title={t('common.filter')}>
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium text-gray">{t('statistics.dateFromLabel')}</label>
            <input type="date" className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4"
              value={filters.date_from || ''} onChange={(e) => setFilters((f) => ({ ...f, date_from: e.target.value }))} />
          </div>
          <div>
            <label className="text-sm font-medium text-gray">{t('statistics.dateToLabel')}</label>
            <input type="date" className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4"
              value={filters.date_to || ''} onChange={(e) => setFilters((f) => ({ ...f, date_to: e.target.value }))} />
          </div>
          <div className="flex gap-3">
            <Button variant="secondary" fullWidth onClick={() => { setFilters({}); setShowFilters(false) }}>{t('common.clear')}</Button>
            <Button fullWidth onClick={() => setShowFilters(false)}>{t('common.apply')}</Button>
          </div>
        </div>
      </BottomSheet>
    </div>
  )
}
