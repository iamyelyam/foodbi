import { useState } from 'react'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { CardSkeleton } from '@/components/ui/skeleton'
import { RevenueChart } from '@/components/charts/RevenueChart'
import { PeriodPills } from '@/components/ui/period-pills'
import { useDashboard, useRevenueTrend, useUnreadNotificationCount } from '@/hooks/useApi'
import { TrendingUp, TrendingDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useCurrency } from '@/stores/app'
import { useT } from '@/i18n'

export function EmployeeHomePage() {
  const t = useT()
  const cs = useCurrency()
  const [trendDays, setTrendDays] = useState<number>(7)
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const { data: summary, isLoading } = useDashboard()
  const { data: trend = [] } = useRevenueTrend(trendDays)

  const changePercent = summary?.revenue_change_percent ?? 0
  const isPositive = changePercent >= 0

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="FoodBI" showNotification badgeCount={unreadCount} />

      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {isLoading ? (
          <CardSkeleton />
        ) : (
          <div className="bg-white rounded-[16px] p-4 shadow-sm">
            <div className="flex items-center justify-between mb-1">
              <span className="text-sm text-gray">{t('employeeHome.todaysRevenue')}</span>
              <span className="text-xs text-primary font-medium bg-primary-lighter px-2 py-0.5 rounded-full">{t('employeeHome.todayBadge')}</span>
            </div>
            <p className="text-3xl font-bold text-dark">
              {(summary?.today_revenue ?? 0).toLocaleString('ru-KZ', { maximumFractionDigits: 0 })}{cs}
            </p>
            <div className="flex items-center gap-1.5 mt-1">
              {isPositive ? <TrendingUp className="h-3.5 w-3.5 text-success" /> : <TrendingDown className="h-3.5 w-3.5 text-danger" />}
              <span className={cn('text-xs font-medium', isPositive ? 'text-success' : 'text-danger')}>
                {t('employeeHome.percentVsLastWeek', { percent: (isPositive ? '+' : '') + changePercent.toFixed(1) })}
              </span>
            </div>
            <p className="text-xs text-gray mt-0.5">{t('employeeHome.ordersCount', { count: summary?.today_orders ?? 0 })}</p>
          </div>
        )}

        <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-3">
          <h3 className="text-sm font-semibold text-dark">{t('employeeHome.revenueTrend')}</h3>
          <PeriodPills value={trendDays} onChange={setTrendDays} />
          {trend.length > 0 ? (
            <RevenueChart data={trend} height={180} />
          ) : (
            <p className="text-sm text-gray text-center py-4">{t('employeeHome.noDataPeriod')}</p>
          )}
        </div>

        <div className="bg-white rounded-[16px] p-4 shadow-sm">
          <p className="text-sm text-gray text-center">
            {t('employeeHome.limitedAccess')}
          </p>
        </div>
      </main>

      <Tabbar />
    </div>
  )
}
