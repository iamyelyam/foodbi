import { useState } from 'react'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { DatePicker } from '@/components/ui/date-picker'
import { FilterChip } from '@/components/ui/filter-chip'
import { RevenueChart } from '@/components/charts/RevenueChart'
import { ProfitChart } from '@/components/charts/ProfitChart'
import { useRevenueStats, useProfitStats, useUnreadNotificationCount } from '@/hooks/useApi'
import { Filter } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useCurrency } from '@/stores/app'
import { useT } from '@/i18n'

type Tab = 'revenue' | 'profit'

export function StatisticsPage() {
  const cs = useCurrency()
  const t = useT()
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const [tab, setTab] = useState<Tab>('revenue')
  const [period, setPeriod] = useState<'7' | '30' | '90' | 'custom'>('30')
  const [showFilters, setShowFilters] = useState(false)
  const [pickingDate, setPickingDate] = useState<'from' | 'to' | null>(null)
  const [customDateFrom, setCustomDateFrom] = useState('')
  const [customDateTo, setCustomDateTo] = useState('')

  const dateFrom = period === 'custom' && customDateFrom
    ? customDateFrom
    : new Date(Date.now() - Number(period === 'custom' ? 30 : period) * 86400000).toISOString().split('T')[0]
  const dateTo = period === 'custom' && customDateTo
    ? customDateTo
    : new Date().toISOString().split('T')[0]

  const { data: revenueData = [] } = useRevenueStats(dateFrom, dateTo)
  const { data: profitData = [] } = useProfitStats(dateFrom, dateTo)

  const totalRevenue = revenueData.reduce((s: number, p: any) => s + p.revenue, 0)
  const totalOrders = revenueData.reduce((s: number, p: any) => s + (p.orders || 0), 0)
  const totalProfit = profitData.reduce((s: number, p: any) => s + p.profit, 0)
  const totalCost = profitData.reduce((s: number, p: any) => s + p.cost, 0)

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={t('statistics.pageTitle')} showBack showNotification badgeCount={unreadCount} />

      {/* Tab control */}
      <div className="px-4 pt-2 pb-3">
        <div className="flex bg-bg-alt rounded-[12px] p-1">
          {(['revenue', 'profit'] as Tab[]).map((tabKey) => (
            <button
              key={tabKey}
              onClick={() => setTab(tabKey)}
              className={cn(
                'flex-1 py-2 text-sm font-medium rounded-[10px] transition-colors',
                tab === tabKey ? 'bg-white text-dark shadow-sm' : 'text-gray'
              )}
            >
              {tabKey === 'revenue' ? t('statistics.revenueTab') : t('statistics.profitTab')}
            </button>
          ))}
        </div>
      </div>

      {/* Period selector */}
      <div className="px-4 pb-3 flex items-center gap-2">
        {([['7', '7D'], ['30', '30D'], ['90', '90D']] as const).map(([val, label]) => (
          <button
            key={val}
            onClick={() => { setPeriod(val); setCustomDateFrom(''); setCustomDateTo('') }}
            className={cn(
              'px-3 py-1.5 text-xs font-medium rounded-full transition-colors',
              period === val ? 'bg-primary text-white' : 'bg-white text-gray'
            )}
          >
            {label}
          </button>
        ))}
        <button
          onClick={() => setShowFilters(true)}
          className={cn(
            'flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-full transition-colors',
            period === 'custom' ? 'bg-primary text-white' : 'bg-white text-gray'
          )}
        >
          <Filter className="h-3 w-3" /> {t('statistics.custom')}
        </button>
      </div>

      {/* Active date filter pills */}
      {period === 'custom' && (customDateFrom || customDateTo) && (
        <div className="px-4 pb-3 flex flex-wrap gap-2">
          {customDateFrom && (
            <FilterChip
              label={t('statistics.fromPrefix', { date: customDateFrom })}
              onRemove={() => { setCustomDateFrom(''); if (!customDateTo) setPeriod('30') }}
            />
          )}
          {customDateTo && (
            <FilterChip
              label={t('statistics.toPrefix', { date: customDateTo })}
              onRemove={() => { setCustomDateTo(''); if (!customDateFrom) setPeriod('30') }}
            />
          )}
        </div>
      )}

      <main className="flex-1 px-4 pb-28 space-y-3">
        {tab === 'revenue' && (
          <>
            {/* Summary cards */}
            <div className="grid grid-cols-2 gap-3">
              <div className="bg-white rounded-[12px] p-3 shadow-sm">
                <p className="text-xs text-gray">{t('statistics.totalRevenue')}</p>
                <p className="text-lg font-bold text-dark mt-1">
                  {totalRevenue.toLocaleString('ru-KZ', { maximumFractionDigits: 0 })}{cs}
                </p>
              </div>
              <div className="bg-white rounded-[12px] p-3 shadow-sm">
                <p className="text-xs text-gray">{t('statistics.totalOrders')}</p>
                <p className="text-lg font-bold text-dark mt-1">{totalOrders}</p>
              </div>
            </div>

            {/* Chart */}
            <div className="bg-white rounded-[16px] p-4 shadow-sm">
              <h3 className="text-sm font-semibold text-dark mb-3">{t('statistics.revenueOverTime')}</h3>
              {revenueData.length > 0 ? (
                <RevenueChart data={revenueData} height={220} />
              ) : (
                <p className="text-sm text-gray text-center py-8">{t('statistics.noDataPeriod')}</p>
              )}
            </div>
          </>
        )}

        {tab === 'profit' && (
          <>
            <div className="grid grid-cols-2 gap-3">
              <div className="bg-white rounded-[12px] p-3 shadow-sm">
                <p className="text-xs text-gray">{t('statistics.grossProfit')}</p>
                <p className={cn('text-lg font-bold mt-1', totalProfit >= 0 ? 'text-success' : 'text-danger')}>
                  {totalProfit.toLocaleString('ru-KZ', { maximumFractionDigits: 0 })}{cs}
                </p>
              </div>
              <div className="bg-white rounded-[12px] p-3 shadow-sm">
                <p className="text-xs text-gray">{t('statistics.totalCost')}</p>
                <p className="text-lg font-bold text-warning mt-1">
                  {totalCost.toLocaleString('ru-KZ', { maximumFractionDigits: 0 })}{cs}
                </p>
              </div>
            </div>

            <div className="bg-white rounded-[16px] p-4 shadow-sm">
              <h3 className="text-sm font-semibold text-dark mb-3">{t('statistics.revenueVsCost')}</h3>
              {profitData.length > 0 ? (
                <ProfitChart data={profitData} height={220} />
              ) : (
                <p className="text-sm text-gray text-center py-8">{t('statistics.noDataPeriod')}</p>
              )}
            </div>
          </>
        )}
      </main>

      <Tabbar />

      {/* Custom date range BottomSheet */}
      <BottomSheet isOpen={showFilters} onClose={() => { setShowFilters(false); setPickingDate(null) }} title={t('statistics.customRange')}>
        <div className="space-y-4">
          {/* Date from */}
          <div>
            <label className="text-xs font-medium text-gray mb-1 block">{t('statistics.dateFromLabel')}</label>
            <button
              onClick={() => setPickingDate(pickingDate === 'from' ? null : 'from')}
              className="w-full h-12 rounded-[12px] border border-bg-alt px-4 text-left text-sm text-dark"
            >
              {customDateFrom || t('statistics.selectDate')}
            </button>
            {pickingDate === 'from' && (
              <div className="mt-2">
                <DatePicker
                  value={customDateFrom}
                  onChange={(date) => { setCustomDateFrom(date); setPickingDate(null) }}
                  onClose={() => setPickingDate(null)}
                />
              </div>
            )}
          </div>

          {/* Date to */}
          <div>
            <label className="text-xs font-medium text-gray mb-1 block">{t('statistics.dateToLabel')}</label>
            <button
              onClick={() => setPickingDate(pickingDate === 'to' ? null : 'to')}
              className="w-full h-12 rounded-[12px] border border-bg-alt px-4 text-left text-sm text-dark"
            >
              {customDateTo || t('statistics.selectDate')}
            </button>
            {pickingDate === 'to' && (
              <div className="mt-2">
                <DatePicker
                  value={customDateTo}
                  onChange={(date) => { setCustomDateTo(date); setPickingDate(null) }}
                  onClose={() => setPickingDate(null)}
                />
              </div>
            )}
          </div>

          <div className="flex gap-3">
            <Button variant="secondary" fullWidth onClick={() => { setCustomDateFrom(''); setCustomDateTo(''); setPeriod('30'); setShowFilters(false); setPickingDate(null) }}>
              {t('common.clear')}
            </Button>
            <Button fullWidth onClick={() => { setPeriod('custom'); setShowFilters(false); setPickingDate(null) }}>
              {t('common.apply')}
            </Button>
          </div>
        </div>
      </BottomSheet>
    </div>
  )
}
