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

export function EmployeeHomePage() {
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
              <span className="text-sm text-gray">Today's Revenue</span>
              <span className="text-xs text-primary font-medium bg-primary-lighter px-2 py-0.5 rounded-full">Today</span>
            </div>
            <p className="text-3xl font-bold text-dark">
              {(summary?.today_revenue ?? 0).toLocaleString('ru-KZ', { maximumFractionDigits: 0 })}{cs}
            </p>
            <div className="flex items-center gap-1.5 mt-1">
              {isPositive ? <TrendingUp className="h-3.5 w-3.5 text-success" /> : <TrendingDown className="h-3.5 w-3.5 text-danger" />}
              <span className={cn('text-xs font-medium', isPositive ? 'text-success' : 'text-danger')}>
                {isPositive ? '+' : ''}{changePercent.toFixed(1)}% vs last week
              </span>
            </div>
            <p className="text-xs text-gray mt-0.5">{summary?.today_orders ?? 0} orders</p>
          </div>
        )}

        <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-3">
          <h3 className="text-sm font-semibold text-dark">Revenue Trend</h3>
          <PeriodPills value={trendDays} onChange={setTrendDays} />
          {trend.length > 0 ? (
            <RevenueChart data={trend} height={180} />
          ) : (
            <p className="text-sm text-gray text-center py-4">No data for this period</p>
          )}
        </div>

        <div className="bg-white rounded-[16px] p-4 shadow-sm">
          <p className="text-sm text-gray text-center">
            You're viewing data for your assigned location. Contact your manager for full access.
          </p>
        </div>
      </main>

      <Tabbar />
    </div>
  )
}
