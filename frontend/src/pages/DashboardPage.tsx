import { useNavigate } from 'react-router-dom'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { RevenueChart } from '@/components/charts/RevenueChart'
import { useDashboard, useRevenueTrend } from '@/hooks/useApi'
import { TrendingUp, TrendingDown, ShoppingCart, Package, BarChart3 } from 'lucide-react'
import { cn } from '@/lib/utils'

export function DashboardPage() {
  const navigate = useNavigate()
  const { data: summary } = useDashboard()
  const { data: trend = [] } = useRevenueTrend(7)

  const changePercent = summary?.revenue_change_percent ?? 0
  const isPositive = changePercent >= 0

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="FoodBI" showNotification />

      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {/* Revenue card */}
        <div className="bg-white rounded-[16px] p-4 shadow-sm">
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm text-gray">Today's Revenue</span>
            <span className="text-xs text-primary font-medium bg-primary-lighter px-2 py-0.5 rounded-full">
              Today
            </span>
          </div>
          <p className="text-3xl font-bold text-dark">
            ${(summary?.today_revenue ?? 0).toLocaleString('en', { minimumFractionDigits: 2 })}
          </p>
          <div className="flex items-center gap-1.5 mt-1">
            {isPositive ? (
              <TrendingUp className="h-3.5 w-3.5 text-success" />
            ) : (
              <TrendingDown className="h-3.5 w-3.5 text-danger" />
            )}
            <span className={cn('text-xs font-medium', isPositive ? 'text-success' : 'text-danger')}>
              {isPositive ? '+' : ''}{changePercent.toFixed(1)}% vs last week
            </span>
          </div>
          <p className="text-xs text-gray mt-0.5">{summary?.today_orders ?? 0} orders</p>
        </div>

        {/* Purchases card */}
        <div className="bg-white rounded-[16px] p-4 shadow-sm">
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm text-gray">Today's Purchases</span>
            <span className="text-xs text-warning font-medium bg-warning/10 px-2 py-0.5 rounded-full">
              Cost
            </span>
          </div>
          <p className="text-3xl font-bold text-dark">
            ${(summary?.today_purchases ?? 0).toLocaleString('en', { minimumFractionDigits: 2 })}
          </p>
        </div>

        {/* Revenue trend chart */}
        {trend.length > 0 && (
          <div className="bg-white rounded-[16px] p-4 shadow-sm">
            <h3 className="text-sm font-semibold text-dark mb-3">Revenue Trend (7 days)</h3>
            <RevenueChart data={trend} height={180} />
          </div>
        )}

        {/* Quick actions */}
        <div>
          <h2 className="text-sm font-semibold text-dark mb-2">Quick Actions</h2>
          <div className="grid grid-cols-2 gap-3">
            {[
              { label: 'Revenue', icon: TrendingUp, to: '/revenue', color: 'bg-primary-lighter text-primary' },
              { label: 'Purchases', icon: ShoppingCart, to: '/purchases', color: 'bg-warning/10 text-warning' },
              { label: 'Stock', icon: Package, to: '/stock', color: 'bg-info-light text-info' },
              { label: 'Statistics', icon: BarChart3, to: '/statistics', color: 'bg-success/10 text-success' },
            ].map(({ label, icon: Icon, to, color }) => (
              <button
                key={label}
                onClick={() => navigate(to)}
                className="bg-white rounded-[12px] p-4 flex items-center gap-3 shadow-sm text-left"
              >
                <div className={cn('w-10 h-10 rounded-full flex items-center justify-center', color)}>
                  <Icon className="h-5 w-5" />
                </div>
                <span className="text-sm font-medium text-dark">{label}</span>
              </button>
            ))}
          </div>
        </div>
      </main>

      <Tabbar />
    </div>
  )
}
