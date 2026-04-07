import { useState } from 'react'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { useOrders, useProducts } from '@/hooks/useApi'
import { Filter, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

type Tab = 'orders' | 'products'

export function RevenuePage() {
  const [tab, setTab] = useState<Tab>('orders')
  const [showFilters, setShowFilters] = useState(false)
  const [filters, setFilters] = useState<Record<string, string>>({})

  const { data: ordersData } = useOrders(filters)
  const { data: products = [] } = useProducts(filters)

  const orders = ordersData?.orders ?? []

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="Revenue" showBack showNotification />

      {/* Segmented control */}
      <div className="px-4 pt-2 pb-3">
        <div className="flex bg-bg-alt rounded-[12px] p-1">
          {(['orders', 'products'] as Tab[]).map((t) => (
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

      {/* Filter bar */}
      <div className="px-4 pb-3 flex items-center justify-between">
        <span className="text-xs text-gray">
          {tab === 'orders' ? `${ordersData?.total ?? 0} orders` : `${products.length} products`}
        </span>
        <button
          onClick={() => setShowFilters(true)}
          className="flex items-center gap-1 text-xs font-medium text-primary"
        >
          <Filter className="h-3.5 w-3.5" /> Filters
        </button>
      </div>

      <main className="flex-1 px-4 pb-20 space-y-2">
        {tab === 'orders' &&
          orders.map((order: any) => (
            <div key={order.id} className="bg-white rounded-[12px] p-4 shadow-sm">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-semibold text-dark">
                    ${order.revenue.toFixed(2)}
                  </p>
                  <p className="text-xs text-gray mt-0.5">
                    {new Date(order.order_date).toLocaleDateString()} - {order.item_count} items
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <span
                    className={cn(
                      'text-xs px-2 py-0.5 rounded-full font-medium',
                      order.status === 'closed' ? 'bg-success/10 text-success' : 'bg-warning/10 text-warning'
                    )}
                  >
                    {order.status}
                  </span>
                  <ChevronRight className="h-4 w-4 text-gray-light" />
                </div>
              </div>
              {order.waiter_name && (
                <p className="text-xs text-gray mt-1">Waiter: {order.waiter_name}</p>
              )}
            </div>
          ))}

        {tab === 'products' &&
          products.map((p: any) => (
            <div key={p.product_id} className="bg-white rounded-[12px] p-4 shadow-sm">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-semibold text-dark">{p.product_name}</p>
                  <p className="text-xs text-gray mt-0.5">{p.category || 'Uncategorized'}</p>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold text-dark">${p.total_revenue.toFixed(2)}</p>
                  <p className="text-xs text-gray">{p.total_quantity} sold</p>
                </div>
              </div>
            </div>
          ))}

        {((tab === 'orders' && orders.length === 0) || (tab === 'products' && products.length === 0)) && (
          <div className="text-center py-12">
            <p className="text-sm text-gray">No data yet. Sync with iiko to see {tab}.</p>
          </div>
        )}
      </main>

      <Tabbar />

      <BottomSheet isOpen={showFilters} onClose={() => setShowFilters(false)} title="Filters">
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium text-gray">Date from</label>
            <input
              type="date"
              className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4"
              value={filters.date_from || ''}
              onChange={(e) => setFilters((f) => ({ ...f, date_from: e.target.value }))}
            />
          </div>
          <div>
            <label className="text-sm font-medium text-gray">Date to</label>
            <input
              type="date"
              className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4"
              value={filters.date_to || ''}
              onChange={(e) => setFilters((f) => ({ ...f, date_to: e.target.value }))}
            />
          </div>
          <div className="flex gap-3">
            <Button variant="secondary" fullWidth onClick={() => { setFilters({}); setShowFilters(false) }}>
              Clear
            </Button>
            <Button fullWidth onClick={() => setShowFilters(false)}>
              Apply
            </Button>
          </div>
        </div>
      </BottomSheet>
    </div>
  )
}
