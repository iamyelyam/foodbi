import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { LocationSwitcher } from '@/components/layout/LocationSwitcher'
import { SegmentedControl } from '@/components/ui/segmented-control'
import { CardSkeleton } from '@/components/ui/skeleton'
import { RevenueChart } from '@/components/charts/RevenueChart'
import { useDashboard, useRevenueTrend } from '@/hooks/useApi'
import { useAppStore } from '@/stores/app'
import { TrendingUp, TrendingDown, ShoppingCart, Package, BarChart3, ArrowRightLeft, Sparkles } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'

type View = 'revenue' | 'purchase'

export function DashboardPage() {
  const navigate = useNavigate()
  const [view, setView] = useState<View>('revenue')
  const [showLocations, setShowLocations] = useState(false)
  const activeLocationId = useAppStore((s) => s.activeLocationId)

  const { data: locations = [] } = useQuery({
    queryKey: ['locations'],
    queryFn: () => api.get('/locations').then((r) => r.data),
  })

  const activeLoc = locations.find((l: any) => l.id === activeLocationId)
  const locationName = activeLoc?.name || 'All locations'

  const { data: summary, isLoading } = useDashboard()
  const { data: trend = [] } = useRevenueTrend(7)

  const changePercent = summary?.revenue_change_percent ?? 0
  const isPositive = changePercent >= 0

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header
        title="FoodBI"
        subtitle={locationName}
        onSubtitleClick={() => setShowLocations(true)}
        showNotification
      />

      <div className="px-4 pt-2 pb-3">
        <SegmentedControl
          value={view}
          onChange={setView}
          options={[
            { value: 'revenue', label: 'Revenue' },
            { value: 'purchase', label: 'Purchases' },
          ]}
        />
      </div>

      <main className="flex-1 px-4 pb-20 space-y-3">
        {isLoading ? (
          <>
            <CardSkeleton />
            <CardSkeleton />
          </>
        ) : view === 'revenue' ? (
          <>
            {/* Revenue card */}
            <div className="bg-white rounded-[16px] p-4 shadow-sm">
              <div className="flex items-center justify-between mb-1">
                <span className="text-sm text-gray">Today's Revenue</span>
                <span className="text-xs text-primary font-medium bg-primary-lighter px-2 py-0.5 rounded-full">Today</span>
              </div>
              <p className="text-3xl font-bold text-dark">
                €{(summary?.today_revenue ?? 0).toLocaleString('en', { minimumFractionDigits: 2 })}
              </p>
              <div className="flex items-center gap-1.5 mt-1">
                {isPositive ? <TrendingUp className="h-3.5 w-3.5 text-success" /> : <TrendingDown className="h-3.5 w-3.5 text-danger" />}
                <span className={cn('text-xs font-medium', isPositive ? 'text-success' : 'text-danger')}>
                  {isPositive ? '+' : ''}{changePercent.toFixed(1)}% vs last week
                </span>
              </div>
              <p className="text-xs text-gray mt-0.5">{summary?.today_orders ?? 0} orders</p>
            </div>

            {/* Week summary */}
            <div className="grid grid-cols-2 gap-3">
              <div className="bg-white rounded-[12px] p-3 shadow-sm">
                <p className="text-xs text-gray">Week Revenue</p>
                <p className="text-lg font-bold text-dark mt-1">
                  €{(summary?.week_revenue ?? 0).toLocaleString('en', { minimumFractionDigits: 0 })}
                </p>
              </div>
              <div className="bg-white rounded-[12px] p-3 shadow-sm">
                <p className="text-xs text-gray">Week Orders</p>
                <p className="text-lg font-bold text-dark mt-1">{summary?.week_orders ?? 0}</p>
              </div>
            </div>

            {/* Revenue trend chart */}
            {trend.length > 0 && (
              <div className="bg-white rounded-[16px] p-4 shadow-sm">
                <h3 className="text-sm font-semibold text-dark mb-3">Revenue Trend (7 days)</h3>
                <RevenueChart data={trend} height={180} />
              </div>
            )}
          </>
        ) : (
          <>
            {/* Purchases card */}
            <div className="bg-white rounded-[16px] p-4 shadow-sm">
              <div className="flex items-center justify-between mb-1">
                <span className="text-sm text-gray">Today's Purchases</span>
                <span className="text-xs text-warning font-medium bg-warning/10 px-2 py-0.5 rounded-full">Cost</span>
              </div>
              <p className="text-3xl font-bold text-dark">
                €{(summary?.today_purchases ?? 0).toLocaleString('en', { minimumFractionDigits: 2 })}
              </p>
            </div>
          </>
        )}

        {/* Quick actions */}
        <div>
          <h2 className="text-sm font-semibold text-dark mb-2">Quick Actions</h2>
          <div className="grid grid-cols-3 gap-2">
            {[
              { label: 'Revenue', icon: TrendingUp, to: '/revenue', color: 'bg-primary-lighter text-primary' },
              { label: 'Purchases', icon: ShoppingCart, to: '/purchases', color: 'bg-warning/10 text-warning' },
              { label: 'Stock', icon: Package, to: '/stock', color: 'bg-info-light text-info' },
              { label: 'Statistics', icon: BarChart3, to: '/statistics', color: 'bg-success/10 text-success' },
              { label: 'Transfers', icon: ArrowRightLeft, to: '/transfers', color: 'bg-info-light text-info' },
              { label: 'AI Tips', icon: Sparkles, to: '/ai-suggestions', color: 'bg-primary-lighter text-primary' },
            ].map(({ label, icon: Icon, to, color }) => (
              <button key={label} onClick={() => navigate(to)}
                className="bg-white rounded-[12px] p-3 flex flex-col items-center gap-2 shadow-sm">
                <div className={cn('w-10 h-10 rounded-full flex items-center justify-center', color)}>
                  <Icon className="h-5 w-5" />
                </div>
                <span className="text-[11px] font-medium text-dark">{label}</span>
              </button>
            ))}
          </div>
        </div>
      </main>

      <Tabbar />
      <LocationSwitcher isOpen={showLocations} onClose={() => setShowLocations(false)} />
    </div>
  )
}
