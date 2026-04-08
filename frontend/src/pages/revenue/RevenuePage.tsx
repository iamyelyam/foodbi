import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CardSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { SegmentedControl } from '@/components/ui/segmented-control'
import { Button } from '@/components/ui/button'
import { DatePicker } from '@/components/ui/date-picker'
import { FilterChip } from '@/components/ui/filter-chip'
import { useOrders, useProducts, useUnreadNotificationCount } from '@/hooks/useApi'
import { Filter, ChevronRight, ShoppingBag } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useCurrency } from '@/stores/app'
import { useT } from '@/i18n'

type Tab = 'orders' | 'products'
type StatusFilter = '' | 'open' | 'closed'

export function RevenuePage() {
  const t = useT()
  const [tab, setTab] = useState<Tab>('orders')
  const [showFilters, setShowFilters] = useState(false)
  const [filters, setFilters] = useState<Record<string, string>>({})
  const [pickingDate, setPickingDate] = useState<'from' | 'to' | null>(null)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('')

  const navigate = useNavigate()
  const cs = useCurrency()
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const appliedFilters = { ...filters, ...(statusFilter ? { status: statusFilter } : {}) }
  const { data: ordersData, isLoading: ordersLoading } = useOrders(appliedFilters)
  const { data: products = [], isLoading: productsLoading } = useProducts(appliedFilters)

  const orders = ordersData?.orders ?? []

  const hasActiveFilters = filters.date_from || filters.date_to || statusFilter

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={t('revenue.title')} showBack showNotification badgeCount={unreadCount} />

      <div className="px-4 pt-2 pb-3">
        <SegmentedControl
          value={tab}
          onChange={setTab}
          options={[
            { value: 'orders', label: t('revenue.orders') },
            { value: 'products', label: t('revenue.products') },
          ]}
        />
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
          <Filter className="h-3.5 w-3.5" /> {t('common.filter')}
        </button>
      </div>

      {/* Active filter pills */}
      {hasActiveFilters && (
        <div className="px-4 pb-3 flex flex-wrap gap-2">
          {filters.date_from && (
            <FilterChip
              label={`From: ${filters.date_from}`}
              onRemove={() => setFilters((f) => { const { date_from, ...rest } = f; return rest })}
            />
          )}
          {filters.date_to && (
            <FilterChip
              label={`To: ${filters.date_to}`}
              onRemove={() => setFilters((f) => { const { date_to, ...rest } = f; return rest })}
            />
          )}
          {statusFilter && (
            <FilterChip
              label={`Status: ${statusFilter}`}
              onRemove={() => setStatusFilter('')}
            />
          )}
        </div>
      )}

      <main className="flex-1 px-4 pb-20 space-y-2">
        {(tab === 'orders' ? ordersLoading : productsLoading) ? (
          <>
            <CardSkeleton />
            <CardSkeleton />
            <CardSkeleton />
          </>
        ) : (
        <>
        {tab === 'orders' &&
          orders.map((order: any) => (
            <button key={order.id} className="w-full text-left bg-white rounded-[12px] p-4 shadow-sm" onClick={() => navigate(`/revenue/orders/${order.id}`)}>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-semibold text-dark">
                    {order.revenue.toFixed(2)}{cs}
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
            </button>
          ))}

        {tab === 'products' &&
          products.map((p: any) => (
            <button key={p.product_id} className="w-full text-left bg-white rounded-[12px] p-4 shadow-sm" onClick={() => navigate(`/revenue/products/${p.product_id}`)}>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-semibold text-dark">{p.product_name}</p>
                  <p className="text-xs text-gray mt-0.5">{p.category || 'Uncategorized'}</p>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold text-dark">{p.total_revenue.toFixed(2)}{cs}</p>
                  <p className="text-xs text-gray">{p.total_quantity} sold</p>
                </div>
              </div>
            </button>
          ))}

        {((tab === 'orders' && orders.length === 0) || (tab === 'products' && products.length === 0)) && (
          <div className="text-center py-12">
            <ShoppingBag className="h-12 w-12 text-gray-light mx-auto mb-3" />
            <p className="text-sm text-gray">No data yet. Sync with iiko to see {tab}.</p>
          </div>
        )}
        </>
        )}
      </main>

      <Tabbar />

      <BottomSheet isOpen={showFilters} onClose={() => { setShowFilters(false); setPickingDate(null) }} title="Filters">
        <div className="space-y-4">
          {/* Status filter */}
          <div>
            <label className="text-xs font-medium text-gray mb-2 block">Status</label>
            <div className="flex gap-2">
              {([['', 'All'], ['open', 'Open'], ['closed', 'Closed']] as const).map(([val, label]) => (
                <FilterChip
                  key={val}
                  label={label}
                  active={statusFilter === val}
                  onClick={() => setStatusFilter(val as StatusFilter)}
                />
              ))}
            </div>
          </div>

          {/* Date from */}
          <div>
            <label className="text-xs font-medium text-gray mb-1 block">Date from</label>
            <button
              onClick={() => setPickingDate(pickingDate === 'from' ? null : 'from')}
              className="w-full h-12 rounded-[12px] border border-bg-alt px-4 text-left text-sm text-dark"
            >
              {filters.date_from || 'Select date'}
            </button>
            {pickingDate === 'from' && (
              <div className="mt-2">
                <DatePicker
                  value={filters.date_from}
                  onChange={(date) => { setFilters((f) => ({ ...f, date_from: date })); setPickingDate(null) }}
                  onClose={() => setPickingDate(null)}
                />
              </div>
            )}
          </div>

          {/* Date to */}
          <div>
            <label className="text-xs font-medium text-gray mb-1 block">Date to</label>
            <button
              onClick={() => setPickingDate(pickingDate === 'to' ? null : 'to')}
              className="w-full h-12 rounded-[12px] border border-bg-alt px-4 text-left text-sm text-dark"
            >
              {filters.date_to || 'Select date'}
            </button>
            {pickingDate === 'to' && (
              <div className="mt-2">
                <DatePicker
                  value={filters.date_to}
                  onChange={(date) => { setFilters((f) => ({ ...f, date_to: date })); setPickingDate(null) }}
                  onClose={() => setPickingDate(null)}
                />
              </div>
            )}
          </div>

          <div className="flex gap-3">
            <Button variant="secondary" fullWidth onClick={() => { setFilters({}); setStatusFilter(''); setShowFilters(false); setPickingDate(null) }}>
              {t('common.clear')}
            </Button>
            <Button fullWidth onClick={() => { setShowFilters(false); setPickingDate(null) }}>
              {t('common.apply')}
            </Button>
          </div>
        </div>
      </BottomSheet>
    </div>
  )
}
