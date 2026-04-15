import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { useAppStore, useCurrency } from '@/stores/app'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import { Package, AlertTriangle, Calendar, ChevronRight, Coins, TrendingDown, Filter, Download } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { formatProductName, isUuid } from '@/lib/format'

function formatDay(dateStr: string): string {
  const d = new Date(dateStr + 'T00:00:00')
  return d.toLocaleDateString('en', { month: 'long', day: 'numeric' })
}
function todayIso(): string {
  return new Date().toISOString().split('T')[0]
}
const formatMoney = (v: number) => v.toLocaleString('ru-KZ', { maximumFractionDigits: 0 })

type Chip = 'stale' | 'lowstock' | null

export function StockPage() {
  const cs = useCurrency()
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const locationId = useAppStore((s) => s.activeLocationId)
  const [selectedItem, setSelectedItem] = useState<any>(null)
  const [activeChip, setActiveChip] = useState<Chip>(null)
  const [dateFrom] = useState<string>(todayIso())
  const [dateTo] = useState<string>(todayIso())

  const { data: stock = [], isLoading } = useQuery<any[]>({
    queryKey: ['stock', locationId],
    queryFn: () => api.get('/stock', { params: { location_id: locationId } }).then((r) => r.data),
  })

  const { data: lowStock = [] } = useQuery<any[]>({
    queryKey: ['low-stock', locationId],
    queryFn: () => api.get('/stock/low-stock', { params: { location_id: locationId } }).then((r) => r.data),
  })

  const lowStockIds = useMemo(
    () => new Set(lowStock.map((i: any) => i.product_id)),
    [lowStock]
  )

  // Items not updated for >30 days (based on snapshot_at)
  const staleIds = useMemo(() => {
    const threshold = Date.now() - 30 * 24 * 60 * 60 * 1000
    return new Set(
      stock
        .filter((i: any) => i.snapshot_at && new Date(i.snapshot_at).getTime() < threshold)
        .map((i: any) => i.product_id)
    )
  }, [stock])

  const filtered = useMemo(() => {
    // Hide garbage stock rows: amount <= 0 (data-entry errors or fully depleted)
    const base = (stock as any[]).filter((i: any) => (i.amount || 0) > 0)
    if (activeChip === 'stale') return base.filter((i: any) => staleIds.has(i.product_id))
    if (activeChip === 'lowstock') return base.filter((i: any) => lowStockIds.has(i.product_id))
    return base
  }, [stock, activeChip, staleIds, lowStockIds])

  // Metrics
  const totals = useMemo(() => {
    const inStock = stock.reduce((s: number, i: any) => s + (i.cost_sum || 0), 0)
    // "Write-off" proxy: monetary value of low-stock items (items at risk)
    const writeOff = stock
      .filter((i: any) => lowStockIds.has(i.product_id))
      .reduce((s: number, i: any) => s + (i.cost_sum || 0), 0)
    return { inStock, writeOff }
  }, [stock, lowStockIds])

  const rangeLabel = `${formatDay(dateFrom)} - ${formatDay(dateTo)}`

  const downloadExcel = () => {
    // Simple CSV export (opens in Excel natively)
    const header = ['Product', 'Amount', 'Unit', 'Cost'].join(',')
    const rows = (filtered as any[]).map((i: any) => {
      const unit = i.unit && !isUuid(i.unit) ? i.unit : 'шт'
      return [
        `"${(i.product_name || '').replace(/"/g, '""')}"`,
        i.amount,
        unit,
        i.cost_sum ?? 0,
      ].join(',')
    })
    const blob = new Blob(['\uFEFF' + [header, ...rows].join('\n')], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `stock-${todayIso()}.csv`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <Header title="Stock Management" showBack showNotification badgeCount={unreadCount} />

      {/* Date range (info-only for now — stock is a snapshot) */}
      <div className="px-4 pt-2 pb-3">
        <button className="flex items-center gap-2 text-sm font-medium text-dark" disabled>
          <Calendar className="h-4 w-4 text-gray" />
          <span>{rangeLabel}</span>
          <ChevronRight className="h-4 w-4 text-gray" />
        </button>
      </div>

      {/* 2 metric cards */}
      <div className="px-4 pb-3">
        <div className="grid grid-cols-2 gap-2">
          <MetricCard
            icon={<Coins className="h-4 w-4 text-primary" />}
            value={formatMoney(totals.inStock) + cs}
            label="In Stock"
          />
          <MetricCard
            icon={<TrendingDown className="h-4 w-4 text-primary" />}
            value={formatMoney(totals.writeOff) + cs}
            label="Write-off"
          />
        </div>
      </div>

      {/* Filters + Download Excel */}
      <div className="px-4 pb-3 flex items-center gap-3">
        <button
          className="flex flex-col items-center gap-0.5 shrink-0"
          aria-label="Filters"
          onClick={() => setActiveChip(null)}
        >
          <Filter className="h-5 w-5 text-dark" />
          <span className="text-[10px] text-gray">Filters</span>
        </button>
        <button
          onClick={downloadExcel}
          className="flex-1 flex items-center justify-center gap-2 bg-primary text-dark font-semibold py-3 rounded-full"
        >
          <Download className="h-4 w-4" />
          Download Excel
        </button>
      </div>

      {/* Chip filters */}
      <div className="px-4 pb-3 flex items-center gap-2 overflow-x-auto">
        <ChipButton
          active={activeChip === 'stale'}
          onClick={() => setActiveChip(activeChip === 'stale' ? null : 'stale')}
        >
          More than 30 days
        </ChipButton>
        <ChipButton
          active={activeChip === 'lowstock'}
          onClick={() => setActiveChip(activeChip === 'lowstock' ? null : 'lowstock')}
        >
          Write-off soon
        </ChipButton>
      </div>

      <main className="flex-1 px-4 pb-20">
        {isLoading ? (
          <div className="space-y-2">
            <ListItemSkeleton />
            <ListItemSkeleton />
            <ListItemSkeleton />
          </div>
        ) : filtered.length === 0 ? (
          <div className="text-center py-12">
            <Package className="h-12 w-12 text-gray-light mx-auto mb-3" />
            <p className="text-sm text-gray">No stock matching filters</p>
          </div>
        ) : (
          <div className="divide-y divide-bg-alt">
            {filtered.map((item: any) => (
              <button
                key={item.product_id}
                onClick={() => setSelectedItem(item)}
                className="w-full text-left py-3"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold text-dark truncate">
                      {formatProductName(item.product_name)}
                    </p>
                    <p className="text-xs text-gray mt-0.5">
                      {Math.round(item.amount || 0)} {item.unit && !isUuid(item.unit) ? item.unit : 'шт'}
                    </p>
                  </div>
                  <p className="text-sm font-bold text-dark shrink-0">
                    {formatMoney(item.cost_sum ?? 0)}
                    {cs}
                  </p>
                </div>
              </button>
            ))}
          </div>
        )}
      </main>

      <Tabbar />

      <BottomSheet isOpen={!!selectedItem} onClose={() => setSelectedItem(null)}>
        {selectedItem && (
          <div className="space-y-4">
            <p className="text-lg font-bold text-dark">{formatProductName(selectedItem.product_name)}</p>

            <div className="grid grid-cols-2 gap-2">
              <div className="bg-bg rounded-[12px] p-3">
                <p className="text-sm font-bold text-dark">
                  {Math.round(selectedItem.amount || 0)}{' '}
                  {selectedItem.unit && !isUuid(selectedItem.unit) ? selectedItem.unit : 'шт'}
                </p>
                <p className="text-[10px] text-gray mt-0.5">Quantity</p>
              </div>
              <div className="bg-bg rounded-[12px] p-3">
                <p className="text-sm font-bold text-dark">
                  {formatMoney(selectedItem.cost_sum ?? 0)}
                  {cs}
                </p>
                <p className="text-[10px] text-gray mt-0.5">Cost value</p>
              </div>
            </div>

            <div className="flex items-center justify-between text-sm border-t border-bg-alt pt-3">
              <span className="text-gray">Last synced</span>
              <span className="font-semibold text-dark">
                {new Date(selectedItem.snapshot_at).toLocaleString('ru-RU')}
              </span>
            </div>

            {lowStockIds.has(selectedItem.product_id) && (
              <div className="bg-danger/5 border border-danger/20 rounded-[10px] p-3 flex items-center gap-2">
                <AlertTriangle className="h-4 w-4 text-danger" />
                <span className="text-xs font-medium text-danger">Low stock — consider restocking</span>
              </div>
            )}

            <button
              onClick={() => setSelectedItem(null)}
              className="w-full text-center text-primary font-semibold py-2"
            >
              Back
            </button>
          </div>
        )}
      </BottomSheet>
    </div>
  )
}

function MetricCard({
  icon,
  value,
  label,
}: {
  icon: React.ReactNode
  value: string
  label: string
}) {
  return (
    <div className="bg-bg rounded-[12px] p-2">
      <div className="w-6 h-6 rounded-full bg-primary-lighter flex items-center justify-center mb-1">
        {icon}
      </div>
      <p className="text-sm font-bold text-dark leading-tight truncate">{value}</p>
      <p className="text-[10px] text-gray mt-0.5">{label}</p>
    </div>
  )
}

function ChipButton({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'shrink-0 px-4 py-1.5 rounded-full text-xs font-medium border transition-colors',
        active
          ? 'bg-primary-lighter text-dark border-primary'
          : 'bg-white text-dark border-primary/40'
      )}
    >
      {children}
    </button>
  )
}
