import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import api from '@/lib/api'
import { CardSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { DateRangePicker } from '@/components/ui/date-range-picker'
import { useOrders, useProducts, useUnreadNotificationCount } from '@/hooks/useApi'
import { Filter, ChevronRight, ShoppingBag, Calendar, Coins, ShoppingCart, Receipt, Package } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useCurrency } from '@/stores/app'
import { useT } from '@/i18n'
import { formatProductName, formatPersonName } from '@/lib/format'
import { RevenueChart } from '@/components/charts/RevenueChart'
import { PeriodPills } from '@/components/ui/period-pills'

type Tab = 'orders' | 'products'

function formatDay(dateStr: string): string {
  const d = new Date(dateStr + 'T00:00:00')
  return d.toLocaleDateString('en', { month: 'long', day: 'numeric' })
}

function formatOrderDateTime(iso: string): string {
  const d = new Date(iso)
  const dd = String(d.getDate()).padStart(2, '0')
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const yyyy = d.getFullYear()
  const hh = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  return `${dd}.${mm}.${yyyy} • ${hh}:${min}`
}

function isoDaysAgo(days: number): string {
  const d = new Date()
  d.setDate(d.getDate() - days)
  return d.toISOString().split('T')[0]
}

function todayIso(): string {
  return new Date().toISOString().split('T')[0]
}

const formatMoney = (v: number) => v.toLocaleString('ru-KZ', { maximumFractionDigits: 0 })

export function RevenuePage() {
  const t = useT()
  const navigate = useNavigate()
  const cs = useCurrency()

  const [tab, setTab] = useState<Tab>('orders')
  const [showFilters, setShowFilters] = useState(false)
  const [showRangePicker, setShowRangePicker] = useState(false)
  const [dateFrom, setDateFrom] = useState<string>(todayIso())
  const [dateTo, setDateTo] = useState<string>(todayIso())
  const [selectedOrderId, setSelectedOrderId] = useState<string | null>(null)
  const [selectedProduct, setSelectedProduct] = useState<any | null>(null)
  const [selectedMetric, setSelectedMetric] = useState<null | 'revenue' | 'orders' | 'aov' | 'mit'>(null)
  const [metricDays, setMetricDays] = useState<number>(30)
  const [productDays, setProductDays] = useState<number>(30)

  const { data: metricTrend = [] } = useQuery<any[]>({
    queryKey: ['metric-trend', metricDays],
    queryFn: () =>
      api.get('/dashboard/revenue-trend', { params: { days: metricDays } }).then((r) => r.data),
    enabled: !!selectedMetric,
  })

  // Filters (applied in Filters sheet)
  // Default sort: by order number desc. Price toggle switches to revenue sort.
  const [sortOrder, setSortOrder] = useState<'default' | 'lowest' | 'highest'>('default')
  const [orderTypes, setOrderTypes] = useState<Set<string>>(new Set())
  const [waiters, setWaiters] = useState<Set<string>>(new Set())

  const { data: orderDetail } = useQuery<any>({
    queryKey: ['order-detail', selectedOrderId],
    queryFn: () => api.get(`/revenue/orders/${selectedOrderId}`).then((r) => r.data),
    enabled: !!selectedOrderId,
  })

  const { data: productTrend = [] } = useQuery<any[]>({
    queryKey: ['product-trend', selectedProduct?.product_id, productDays],
    queryFn: () => {
      const to = todayIso()
      const from = isoDaysAgo(productDays)
      return api
        .get(`/revenue/products/${selectedProduct?.product_id}/trend`, {
          params: { date_from: from, date_to: to },
        })
        .then((r) => r.data)
    },
    enabled: !!selectedProduct,
  })

  const params = { date_from: dateFrom, date_to: dateTo }
  const { data: ordersData, isLoading: ordersLoading } = useOrders(params)
  const { data: products = [], isLoading: productsLoading } = useProducts(params)
  const { data: unreadCount = 0 } = useUnreadNotificationCount()

  const rawOrders = ordersData?.orders ?? []
  const totalOrders = ordersData?.total ?? 0

  // List of unique waiter names from loaded orders (for filter dropdown)
  const uniqueWaiters = useMemo(
    () =>
      Array.from(
        new Set((rawOrders as any[]).map((o: any) => o.waiter_name).filter(Boolean))
      ).sort((a, b) => String(a).localeCompare(String(b), 'ru-RU')),
    [rawOrders]
  )

  // Client-side filtering: order_type + waiter + sort order
  const orders = useMemo(() => {
    let filtered = rawOrders as any[]
    if (orderTypes.size > 0) {
      filtered = filtered.filter((o: any) => orderTypes.has(String(o.order_type || '').toLowerCase()))
    }
    if (waiters.size > 0) {
      filtered = filtered.filter((o: any) => waiters.has(o.waiter_name))
    }
    const sorted = [...filtered].sort((a: any, b: any) => {
      if (sortOrder === 'highest') return (b.revenue || 0) - (a.revenue || 0)
      if (sortOrder === 'lowest') return (a.revenue || 0) - (b.revenue || 0)
      // default: order number desc (largest order number first)
      const aNum = parseInt(a.order_number || '0', 10) || 0
      const bNum = parseInt(b.order_number || '0', 10) || 0
      return bNum - aNum
    })
    return sorted
  }, [rawOrders, orderTypes, waiters, sortOrder])

  // Metrics computed from orders
  const metrics = useMemo(() => {
    const revenue = orders.reduce((s: number, o: any) => s + (o.revenue || 0), 0)
    const items = orders.reduce((s: number, o: any) => s + (o.item_count || 0), 0)
    const count = orders.length
    const aov = count > 0 ? revenue / count : 0
    const mit = count > 0 ? items / count : 0
    return { revenue, count, aov, mit }
  }, [orders])

  const rangeLabel = `${formatDay(dateFrom)} - ${formatDay(dateTo)}`

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <Header title="Revenue" showBack showNotification badgeCount={unreadCount} />

      {/* Date range picker button */}
      <div className="px-4 pt-2 pb-3">
        <button
          onClick={() => setShowRangePicker(true)}
          className="flex items-center gap-2 text-sm font-medium text-dark"
        >
          <Calendar className="h-4 w-4 text-gray" />
          <span>{rangeLabel}</span>
          <ChevronRight className="h-4 w-4 text-gray" />
        </button>
      </div>

      {/* 4 metric cards — light gray bg on white page.
          Revenue card gets extra width when its number has 6+ digits to avoid truncation. */}
      <div className="px-4 pb-3">
        <div
          className="grid gap-2"
          style={{
            gridTemplateColumns:
              metrics.revenue >= 100000
                ? '1.6fr 1fr 1.2fr 1fr'
                : 'repeat(4, minmax(0, 1fr))',
          }}
        >
          <MetricCard icon={<Coins className="h-4 w-4 text-primary" />} value={formatMoney(metrics.revenue) + cs} label="Revenue" onClick={() => setSelectedMetric('revenue')} />
          <MetricCard icon={<ShoppingCart className="h-4 w-4 text-primary" />} value={String(metrics.count)} label="Orders" onClick={() => setSelectedMetric('orders')} />
          <MetricCard icon={<Receipt className="h-4 w-4 text-primary" />} value={formatMoney(metrics.aov) + cs} label="AOV" onClick={() => setSelectedMetric('aov')} />
          <MetricCard icon={<Package className="h-4 w-4 text-primary" />} value={metrics.mit.toFixed(1)} label="MI/T" onClick={() => setSelectedMetric('mit')} />
        </div>
      </div>

      {/* Filters + pill-shaped segmented control */}
      <div className="px-4 pb-3 flex items-center gap-3">
        <button
          onClick={() => setShowFilters(true)}
          className="flex flex-col items-center gap-0.5 shrink-0"
          aria-label="Filters"
        >
          <Filter className="h-5 w-5 text-dark" />
          <span className="text-[10px] text-gray">Filters</span>
        </button>
        <div className="flex-1">
          <div className="flex bg-bg rounded-full p-1">
            {(['orders', 'products'] as Tab[]).map((val) => (
              <button
                key={val}
                onClick={() => setTab(val)}
                className={cn(
                  'flex-1 py-2 text-sm font-semibold rounded-full transition-colors capitalize',
                  tab === val ? 'bg-primary text-dark shadow-sm' : 'text-dark'
                )}
              >
                {val === 'orders' ? 'Orders' : 'Products'}
              </button>
            ))}
          </div>
        </div>
      </div>

      <main className="flex-1 px-4 pb-20">
        {(tab === 'orders' ? ordersLoading : productsLoading) ? (
          <div className="space-y-2">
            <CardSkeleton />
            <CardSkeleton />
            <CardSkeleton />
          </div>
        ) : (
          <>
            {tab === 'orders' && (
              <div className="divide-y divide-bg-alt">
                {orders.map((order: any) => (
                  <button
                    key={order.id}
                    className="w-full text-left py-3"
                    onClick={() => setSelectedOrderId(order.id)}
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-semibold text-dark">
                          Order #{order.order_number || order.id?.slice(0, 6)}
                        </p>
                        <p className="text-xs text-gray mt-0.5">
                          {formatOrderDateTime(order.order_date)}
                        </p>
                        {order.waiter_name && (
                          <p className="text-xs text-gray mt-0.5">
                            Waiter: {formatPersonName(order.waiter_name)}
                          </p>
                        )}
                      </div>
                      <p className="text-sm font-bold text-dark">
                        {formatMoney(order.revenue)}{cs}
                      </p>
                    </div>
                  </button>
                ))}
                {orders.length === 0 && <EmptyState label={t('revenue.orders')} />}
                {orders.length > 0 && totalOrders > orders.length && (
                  <p className="text-xs text-gray text-center py-3">
                    Showing {orders.length} of {totalOrders}
                  </p>
                )}
              </div>
            )}

            {tab === 'products' && (
              <div className="divide-y divide-bg-alt">
                {products.map((p: any) => {
                  const rev = p.total_revenue ?? 0
                  const cost = p.total_cost ?? 0
                  const marginPct = rev > 0 ? (1 - cost / rev) * 100 : 0
                  const positive = marginPct > 0
                  return (
                    <button
                      key={p.product_id}
                      className="w-full text-left py-3"
                      onClick={() => setSelectedProduct(p)}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-semibold text-dark truncate">
                            {formatProductName(p.product_name)}
                            <span className="ml-2 text-xs text-gray font-normal">
                              {Math.round(p.total_quantity || 0)} pc.
                            </span>
                          </p>
                          <p className="text-xs text-gray mt-0.5">Avg. margin</p>
                        </div>
                        <div className="text-right shrink-0">
                          <p className="text-sm font-semibold text-dark">
                            {formatMoney(rev)}{cs}
                          </p>
                          <p
                            className={cn(
                              'text-xs font-medium',
                              positive ? 'text-success' : 'text-danger'
                            )}
                          >
                            {positive ? '+' : ''}
                            {marginPct.toFixed(0)}%
                          </p>
                        </div>
                      </div>
                    </button>
                  )
                })}
                {products.length === 0 && <EmptyState label={t('revenue.products')} />}
              </div>
            )}
          </>
        )}
      </main>

      <Tabbar />

      {/* Order detail BottomSheet */}
      <BottomSheet
        isOpen={!!selectedOrderId}
        onClose={() => setSelectedOrderId(null)}
      >
        {orderDetail && (
          <div className="space-y-4">
            {/* Header row */}
            <div className="flex items-baseline justify-between">
              <p className="text-lg font-bold text-dark">
                Order #{orderDetail.order_number || orderDetail.id?.slice(0, 6)}
              </p>
              <div className="text-right">
                <p className="text-sm text-gray">
                  {formatOrderDateTime(orderDetail.order_date)}
                </p>
                {orderDetail.waiter_name && (
                  <p className="text-xs text-gray mt-0.5">
                    Waiter: {formatPersonName(orderDetail.waiter_name)}
                  </p>
                )}
              </div>
            </div>

            {/* 3 metric cards: Total / Expenses / Profit */}
            <div className="grid grid-cols-3 gap-2">
              <div className="bg-bg rounded-[12px] p-3">
                <p className="text-sm font-bold text-dark">
                  {formatMoney(orderDetail.revenue || 0)}{cs}
                </p>
                <p className="text-[10px] text-gray mt-0.5">Total</p>
              </div>
              <div className="bg-bg rounded-[12px] p-3">
                <p className="text-sm font-bold text-dark">
                  {formatMoney(orderDetail.total_cost || 0)}{cs}
                </p>
                <p className="text-[10px] text-gray mt-0.5">Expenses</p>
              </div>
              <div className="bg-bg rounded-[12px] p-3">
                <p
                  className={cn(
                    'text-sm font-bold',
                    (orderDetail.profit || 0) >= 0 ? 'text-dark' : 'text-danger'
                  )}
                >
                  {formatMoney(orderDetail.profit || 0)}{cs}
                </p>
                <p className="text-[10px] text-gray mt-0.5">Profit</p>
              </div>
            </div>

            {/* Items list */}
            <div className="divide-y divide-bg-alt">
              {(orderDetail.items || []).map((item: any, idx: number) => (
                <div key={idx} className="flex items-center justify-between py-3">
                  <p className="text-sm text-dark flex-1 min-w-0 truncate">
                    <span className="font-semibold">
                      {formatProductName(item.product_name)}
                    </span>
                    <span className="ml-2 text-xs text-gray font-normal">
                      {Math.round(item.quantity || 0)} pc.
                    </span>
                  </p>
                  <p className="text-sm font-bold text-dark shrink-0">
                    {formatMoney(item.revenue || 0)}{cs}
                  </p>
                </div>
              ))}
              {(!orderDetail.items || orderDetail.items.length === 0) && (
                <p className="text-center text-sm text-gray py-4">No items</p>
              )}
            </div>

            <button
              onClick={() => setSelectedOrderId(null)}
              className="w-full text-center text-primary font-semibold py-2"
            >
              Back
            </button>
          </div>
        )}
      </BottomSheet>

      {/* Filters BottomSheet */}
      <BottomSheet isOpen={showFilters} onClose={() => setShowFilters(false)}>
        <div className="space-y-5">
          {/* Date section */}
          <div>
            <p className="text-base font-bold text-dark mb-3">Date</p>
            <div className="space-y-2">
              {(
                [
                  ['today', 'Today'],
                  ['yesterday', 'Yesterday'],
                  ['this_week', 'This week'],
                  ['this_month', 'This month'],
                ] as const
              ).map(([key, label]) => {
                const active = isPresetActive(key, dateFrom, dateTo)
                return (
                  <button
                    key={key}
                    onClick={() => {
                      const [f, t] = presetRange(key)
                      setDateFrom(f)
                      setDateTo(t)
                    }}
                    className={cn(
                      'w-full py-3 rounded-[12px] text-sm font-medium transition-colors',
                      active
                        ? 'bg-primary-lighter text-dark border-2 border-primary'
                        : 'bg-bg text-dark'
                    )}
                  >
                    {label}
                  </button>
                )
              })}
            </div>

            {/* From / To inputs */}
            <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2 mt-3">
              <div>
                <p className="text-xs text-gray mb-1">From</p>
                <button
                  onClick={() => setShowRangePicker(true)}
                  className="w-full bg-bg rounded-[10px] px-3 py-2 flex items-center gap-1.5 text-xs text-dark"
                >
                  <Calendar className="h-3.5 w-3.5 text-gray" />
                  <span>{formatInputDate(dateFrom)}</span>
                </button>
              </div>
              <span className="text-gray mt-4">—</span>
              <div>
                <p className="text-xs text-gray mb-1">To</p>
                <button
                  onClick={() => setShowRangePicker(true)}
                  className="w-full bg-bg rounded-[10px] px-3 py-2 flex items-center gap-1.5 text-xs text-dark"
                >
                  <Calendar className="h-3.5 w-3.5 text-gray" />
                  <span>{formatInputDate(dateTo)}</span>
                </button>
              </div>
            </div>
          </div>

          <hr className="border-bg-alt" />

          {/* Price sort */}
          <div>
            <p className="text-base font-bold text-dark mb-3">Price</p>
            <div className="flex gap-2">
              {(['lowest', 'highest'] as const).map((key) => {
                const active = sortOrder === key
                const label = key === 'lowest' ? '↑↓ Lowest first' : '↑↓ Highest first'
                return (
                  <button
                    key={key}
                    onClick={() => setSortOrder(key)}
                    className={cn(
                      'flex-1 py-2 rounded-full text-xs font-medium transition-colors',
                      active
                        ? 'bg-primary-lighter text-dark border border-primary'
                        : 'bg-bg text-dark'
                    )}
                  >
                    {label}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Waiter multi-select */}
          {uniqueWaiters.length > 0 && (
            <div>
              <p className="text-base font-bold text-dark mb-3">Waiter</p>
              <div className="flex flex-wrap gap-2 max-h-40 overflow-y-auto">
                {uniqueWaiters.map((w: any) => {
                  const active = waiters.has(w as string)
                  return (
                    <button
                      key={w as string}
                      onClick={() => {
                        setWaiters((prev) => {
                          const next = new Set(prev)
                          if (next.has(w as string)) next.delete(w as string)
                          else next.add(w as string)
                          return next
                        })
                      }}
                      className={cn(
                        'px-3 py-1.5 rounded-full text-xs font-medium transition-colors',
                        active
                          ? 'bg-primary-lighter text-dark border border-primary'
                          : 'bg-bg text-dark'
                      )}
                    >
                      {formatPersonName(w as string)}
                    </button>
                  )
                })}
              </div>
            </div>
          )}

          {/* Order type multi-select */}
          <div className="flex bg-bg rounded-full p-1">
            {(['delivery', 'takeaway', 'dine-in'] as const).map((key) => {
              const active = orderTypes.has(key)
              const label = key === 'dine-in' ? 'Dine-in' : key.charAt(0).toUpperCase() + key.slice(1)
              return (
                <button
                  key={key}
                  onClick={() => {
                    setOrderTypes((prev) => {
                      const next = new Set(prev)
                      if (next.has(key)) next.delete(key)
                      else next.add(key)
                      return next
                    })
                  }}
                  className={cn(
                    'flex-1 py-2 text-sm font-semibold rounded-full transition-colors',
                    active ? 'bg-primary text-dark' : 'text-dark'
                  )}
                >
                  {label}
                </button>
              )
            })}
          </div>

          <button
            onClick={() => setShowFilters(false)}
            className="w-full bg-primary text-dark font-bold py-3 rounded-full"
          >
            Show {orders.length} results
          </button>
          <button
            onClick={() => setShowFilters(false)}
            className="w-full text-center text-primary font-semibold"
          >
            Back
          </button>
        </div>
      </BottomSheet>

      {/* Product detail BottomSheet */}
      <BottomSheet
        isOpen={!!selectedProduct}
        onClose={() => setSelectedProduct(null)}
      >
        {selectedProduct && (
          <div className="space-y-4">
            {/* Product header */}
            <div className="flex flex-col items-center gap-2">
              <div className="w-20 h-20 rounded-full bg-primary-lighter flex items-center justify-center">
                <Package className="h-10 w-10 text-primary" />
              </div>
              <p className="text-xl font-bold text-dark text-center">
                {formatProductName(selectedProduct.product_name)}
              </p>
              {selectedProduct.product_id && (
                <p className="text-xs text-gray flex items-center gap-1.5">
                  <span>Product ID: {selectedProduct.product_id}</span>
                </p>
              )}
            </div>

            {/* 3 metric cards */}
            <div className="grid grid-cols-3 gap-2">
              <div className="bg-bg rounded-[12px] p-3">
                <p className="text-sm font-bold text-dark">
                  {Math.round(selectedProduct.total_quantity || 0)}
                </p>
                <p className="text-[10px] text-gray mt-0.5">Items sold</p>
              </div>
              <div className="bg-bg rounded-[12px] p-3">
                <p className="text-sm font-bold text-dark">
                  {formatMoney(
                    selectedProduct.total_quantity > 0
                      ? (selectedProduct.total_revenue || 0) / selectedProduct.total_quantity
                      : 0
                  )}
                  {cs}
                </p>
                <p className="text-[10px] text-gray mt-0.5">Avg. Price</p>
              </div>
              <div className="bg-bg rounded-[12px] p-3">
                <p className="text-sm font-bold text-dark">
                  {formatMoney(selectedProduct.total_revenue || 0)}
                  {cs}
                </p>
                <p className="text-[10px] text-gray mt-0.5">Total Sales</p>
              </div>
            </div>

            {/* Period selector drives trend/metrics */}
            <PeriodPills value={productDays} onChange={setProductDays} />

            {/* Avg / Best / Worst day — computed from trend. ABOVE chart per design. */}
            {productTrend.length > 0 && (() => {
              const revenues = productTrend.map((d: any) => d.revenue || 0)
              const avg = revenues.reduce((s: number, v: number) => s + v, 0) / revenues.length
              const best = Math.max(...revenues)
              const worst = Math.min(...revenues)
              return (
                <div className="grid grid-cols-3 gap-2">
                  <div className="bg-bg rounded-[12px] p-3">
                    <p className="text-[11px] text-gray">Avg Daily</p>
                    <p className="text-sm font-bold text-dark mt-1">
                      {formatMoney(avg)}
                      {cs}
                    </p>
                  </div>
                  <div className="bg-bg rounded-[12px] p-3">
                    <p className="text-[11px] text-gray">Best Day</p>
                    <p className="text-sm font-bold text-success mt-1">
                      {formatMoney(best)}
                      {cs}
                    </p>
                  </div>
                  <div className="bg-bg rounded-[12px] p-3">
                    <p className="text-[11px] text-gray">Worst Day</p>
                    <p className="text-sm font-bold text-danger mt-1">
                      {formatMoney(worst)}
                      {cs}
                    </p>
                  </div>
                </div>
              )
            })()}

            {/* Statistics chart — last 30 days */}
            {productTrend.length > 0 && (
              <div>
                <p className="text-base font-bold text-dark mb-2">Statistics</p>
                <RevenueChart
                  data={productTrend.map((d: any) => ({
                    date: d.date,
                    revenue: d.revenue,
                    transactions: d.transactions,
                  }))}
                  height={200}
                />
              </div>
            )}

            {/* Category row */}
            {selectedProduct.category && (
              <div className="border-t border-bg-alt pt-3 flex items-center justify-between">
                <p className="text-sm font-bold text-dark">Category</p>
                <p className="text-sm text-dark">{formatProductName(selectedProduct.category)}</p>
              </div>
            )}

            <button
              onClick={() => setSelectedProduct(null)}
              className="w-full text-center text-primary font-semibold py-2"
            >
              Back
            </button>
          </div>
        )}
      </BottomSheet>

      {/* Metric detail BottomSheet */}
      <BottomSheet isOpen={!!selectedMetric} onClose={() => setSelectedMetric(null)}>
        {selectedMetric && (() => {
          const labels: Record<string, string> = {
            revenue: 'Revenue',
            orders: 'Orders',
            aov: 'AOV',
            mit: 'MI/T',
          }
          const chartData = metricTrend.map((p: any) => {
            let value = 0
            if (selectedMetric === 'revenue') value = p.revenue || 0
            else if (selectedMetric === 'orders') value = p.orders || 0
            else if (selectedMetric === 'aov') value = p.orders > 0 ? (p.revenue || 0) / p.orders : 0
            else if (selectedMetric === 'mit') value = p.orders > 0 ? (p.items || 0) / p.orders : 0
            return { date: p.date, revenue: value, transactions: p.orders }
          })

          const revSum = metricTrend.reduce((s: number, p: any) => s + (p.revenue || 0), 0)
          const ordSum = metricTrend.reduce((s: number, p: any) => s + (p.orders || 0), 0)
          const itemsSum = metricTrend.reduce((s: number, p: any) => s + (p.items || 0), 0)
          let totalLabel = ''
          if (selectedMetric === 'revenue') totalLabel = formatMoney(revSum) + cs
          else if (selectedMetric === 'orders') totalLabel = String(ordSum)
          else if (selectedMetric === 'aov') totalLabel = formatMoney(ordSum > 0 ? revSum / ordSum : 0) + cs
          else if (selectedMetric === 'mit') totalLabel = (ordSum > 0 ? itemsSum / ordSum : 0).toFixed(1)

          return (
            <div className="space-y-4">
              <div>
                <p className="text-base font-bold text-dark">{labels[selectedMetric]}</p>
                <p className="text-2xl font-bold text-dark mt-1">{totalLabel}</p>
                <p className="text-xs text-gray mt-0.5">Last {metricDays} days</p>
              </div>

              <PeriodPills value={metricDays} onChange={setMetricDays} />

              {chartData.length > 0 ? (
                <RevenueChart data={chartData} height={220} />
              ) : (
                <p className="text-sm text-gray text-center py-8">No data for this period</p>
              )}

              <button
                onClick={() => setSelectedMetric(null)}
                className="w-full text-center text-primary font-semibold py-2"
              >
                Back
              </button>
            </div>
          )
        })()}
      </BottomSheet>

      {/* Date range picker — rendered LAST so it stacks above Filters sheet */}
      <BottomSheet
        isOpen={showRangePicker}
        onClose={() => setShowRangePicker(false)}
        title="Select period"
      >
        <DateRangePicker
          startDate={dateFrom}
          endDate={dateTo}
          onConfirm={(start, end) => {
            setDateFrom(start)
            setDateTo(end)
            setShowRangePicker(false)
          }}
          onBack={() => setShowRangePicker(false)}
        />
      </BottomSheet>
    </div>
  )
}

// --- date preset helpers ---
function presetRange(key: 'today' | 'yesterday' | 'this_week' | 'this_month'): [string, string] {
  const now = new Date()
  const iso = (d: Date) => d.toISOString().split('T')[0]
  if (key === 'today') return [iso(now), iso(now)]
  if (key === 'yesterday') {
    const y = new Date(now)
    y.setDate(y.getDate() - 1)
    return [iso(y), iso(y)]
  }
  if (key === 'this_week') {
    const start = new Date(now)
    const dow = start.getDay() === 0 ? 6 : start.getDay() - 1
    start.setDate(start.getDate() - dow)
    return [iso(start), iso(now)]
  }
  // this_month
  const start = new Date(now.getFullYear(), now.getMonth(), 1)
  return [iso(start), iso(now)]
}

function isPresetActive(
  key: 'today' | 'yesterday' | 'this_week' | 'this_month',
  from: string,
  to: string
): boolean {
  const [f, t] = presetRange(key)
  return f === from && t === to
}

function formatInputDate(iso: string): string {
  const d = new Date(iso + 'T00:00:00')
  return `${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.${d.getFullYear()}`
}

function MetricCard({
  icon,
  value,
  label,
  onClick,
}: {
  icon: React.ReactNode
  value: string
  label: string
  onClick?: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="bg-bg rounded-[12px] p-2 text-left w-full active:opacity-70 transition-opacity"
    >
      <div className="w-6 h-6 rounded-full bg-primary-lighter flex items-center justify-center mb-1">
        {icon}
      </div>
      <p className="text-sm font-bold text-dark leading-tight truncate">{value}</p>
      <p className="text-[10px] text-gray mt-0.5">{label}</p>
    </button>
  )
}

function EmptyState({ label }: { label: string }) {
  return (
    <div className="text-center py-12">
      <ShoppingBag className="h-12 w-12 text-gray-light mx-auto mb-3" />
      <p className="text-sm text-gray">No {label} for selected period</p>
    </div>
  )
}
