import { useState } from 'react'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { RevenueChart } from '@/components/charts/RevenueChart'
import { ProfitChart } from '@/components/charts/ProfitChart'
import { useRevenueStats, useProfitStats } from '@/hooks/useApi'
import { cn } from '@/lib/utils'

type Tab = 'revenue' | 'profit'

export function StatisticsPage() {
  const [tab, setTab] = useState<Tab>('revenue')
  const [period, setPeriod] = useState<'7' | '30' | '90'>('30')

  const dateFrom = new Date(Date.now() - Number(period) * 86400000).toISOString().split('T')[0]
  const dateTo = new Date().toISOString().split('T')[0]

  const { data: revenueData = [] } = useRevenueStats(dateFrom, dateTo)
  const { data: profitData = [] } = useProfitStats(dateFrom, dateTo)

  const totalRevenue = revenueData.reduce((s: number, p: any) => s + p.revenue, 0)
  const totalOrders = revenueData.reduce((s: number, p: any) => s + (p.orders || 0), 0)
  const totalProfit = profitData.reduce((s: number, p: any) => s + p.profit, 0)
  const totalCost = profitData.reduce((s: number, p: any) => s + p.cost, 0)

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="Statistics" showBack showNotification />

      {/* Tab control */}
      <div className="px-4 pt-2 pb-3">
        <div className="flex bg-bg-alt rounded-[12px] p-1">
          {(['revenue', 'profit'] as Tab[]).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={cn(
                'flex-1 py-2 text-sm font-medium rounded-[10px] transition-colors capitalize',
                tab === t ? 'bg-white text-dark shadow-sm' : 'text-gray'
              )}
            >
              {t}
            </button>
          ))}
        </div>
      </div>

      {/* Period selector */}
      <div className="px-4 pb-3 flex gap-2">
        {([['7', '7D'], ['30', '30D'], ['90', '90D']] as const).map(([val, label]) => (
          <button
            key={val}
            onClick={() => setPeriod(val)}
            className={cn(
              'px-3 py-1.5 text-xs font-medium rounded-full transition-colors',
              period === val ? 'bg-primary text-white' : 'bg-white text-gray'
            )}
          >
            {label}
          </button>
        ))}
      </div>

      <main className="flex-1 px-4 pb-20 space-y-3">
        {tab === 'revenue' && (
          <>
            {/* Summary cards */}
            <div className="grid grid-cols-2 gap-3">
              <div className="bg-white rounded-[12px] p-3 shadow-sm">
                <p className="text-xs text-gray">Total Revenue</p>
                <p className="text-lg font-bold text-dark mt-1">
                  €{totalRevenue.toLocaleString('en', { minimumFractionDigits: 2 })}
                </p>
              </div>
              <div className="bg-white rounded-[12px] p-3 shadow-sm">
                <p className="text-xs text-gray">Total Orders</p>
                <p className="text-lg font-bold text-dark mt-1">{totalOrders}</p>
              </div>
            </div>

            {/* Chart */}
            <div className="bg-white rounded-[16px] p-4 shadow-sm">
              <h3 className="text-sm font-semibold text-dark mb-3">Revenue Over Time</h3>
              {revenueData.length > 0 ? (
                <RevenueChart data={revenueData} height={220} />
              ) : (
                <p className="text-sm text-gray text-center py-8">No data for this period</p>
              )}
            </div>
          </>
        )}

        {tab === 'profit' && (
          <>
            <div className="grid grid-cols-2 gap-3">
              <div className="bg-white rounded-[12px] p-3 shadow-sm">
                <p className="text-xs text-gray">Gross Profit</p>
                <p className={cn('text-lg font-bold mt-1', totalProfit >= 0 ? 'text-success' : 'text-danger')}>
                  €{totalProfit.toLocaleString('en', { minimumFractionDigits: 2 })}
                </p>
              </div>
              <div className="bg-white rounded-[12px] p-3 shadow-sm">
                <p className="text-xs text-gray">Total Cost</p>
                <p className="text-lg font-bold text-warning mt-1">
                  €{totalCost.toLocaleString('en', { minimumFractionDigits: 2 })}
                </p>
              </div>
            </div>

            <div className="bg-white rounded-[16px] p-4 shadow-sm">
              <h3 className="text-sm font-semibold text-dark mb-3">Revenue vs Cost</h3>
              {profitData.length > 0 ? (
                <ProfitChart data={profitData} height={220} />
              ) : (
                <p className="text-sm text-gray text-center py-8">No data for this period</p>
              )}
            </div>
          </>
        )}
      </main>

      <Tabbar />
    </div>
  )
}
