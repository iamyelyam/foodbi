import { useState, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { useAppStore, useCurrency } from '@/stores/app'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import { Package, AlertTriangle, Calendar, ChevronRight, Coins, TrendingDown, Filter, Download, Pencil, Check, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { formatProductName, isUuid } from '@/lib/format'
import { useT, useI18nStore } from '@/i18n'

function formatDay(dateStr: string, locale: string = 'en'): string {
  const d = new Date(dateStr + 'T00:00:00')
  return d.toLocaleDateString(locale, { month: 'long', day: 'numeric' })
}
function todayIso(): string {
  return new Date().toISOString().split('T')[0]
}
const formatMoney = (v: number) => v.toLocaleString('ru-KZ', { maximumFractionDigits: 0 })

// Recipe amounts come from iiko at very small magnitudes (e.g. 0.00086 kg per dish).
// Show the precise value: 2 significant digits below 1 (capped at 6 decimals), 2 fraction
// digits between 1-10, 1 fraction digit above. Russian comma as decimal separator.
function formatRecipeAmount(v: number): string {
  if (!v || v <= 0) return '0'
  if (v < 1) {
    return v.toLocaleString('ru-KZ', {
      maximumSignificantDigits: 2,
      maximumFractionDigits: 6,
    })
  }
  if (v < 10) return v.toLocaleString('ru-KZ', { maximumFractionDigits: 2 })
  return v.toLocaleString('ru-KZ', { maximumFractionDigits: 1 })
}

type DishUsage = {
  dish_iiko_id: string
  dish_name: string
  amount: number
  unit: string
  dish_unit: string
}

type Chip = 'stale' | 'lowstock' | null

export function StockPage() {
  const t = useT()
  const locale = useI18nStore((s) => s.locale)
  const cs = useCurrency()
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const locationId = useAppStore((s) => s.activeLocationId)
  const [selectedItem, setSelectedItem] = useState<any>(null)
  const [editingProductId, setEditingProductId] = useState<string | null>(null)
  const [aliasDraft, setAliasDraft] = useState<string>('')
  // Inline override editing — null when not editing, else the field being edited.
  const [editingOverride, setEditingOverride] = useState<'amount' | 'price' | null>(null)
  const [overrideDraft, setOverrideDraft] = useState<string>('')
  const queryClient = useQueryClient()

  const overrideMutation = useMutation({
    mutationFn: ({
      productId,
      manual_amount,
      manual_price_per_unit,
    }: {
      productId: string
      manual_amount?: number | null
      manual_price_per_unit?: number | null
    }) =>
      api.put(`/stock/products/${productId}/override`, {
        manual_amount,
        manual_price_per_unit,
      }),
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({ queryKey: ['stock'] })
      queryClient.invalidateQueries({ queryKey: ['low-stock'] })
      // Optimistic update on the open sheet
      setSelectedItem((prev: any) => {
        if (!prev) return prev
        const next = { ...prev }
        if (vars.manual_amount != null) next.amount = vars.manual_amount
        if (vars.manual_price_per_unit != null) next.price_per_unit = vars.manual_price_per_unit
        // Recompute cost_sum locally so the third card updates instantly
        next.cost_sum = (next.amount || 0) * (next.price_per_unit || 0)
        next.override_at = new Date().toISOString()
        return next
      })
      setEditingOverride(null)
    },
  })

  const aliasMutation = useMutation({
    mutationFn: ({ productId, displayName }: { productId: string; displayName: string }) =>
      api.put(`/stock/products/${productId}/alias`, { display_name: displayName }),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['stock'] })
      queryClient.invalidateQueries({ queryKey: ['low-stock'] })
      // Optimistically update the open sheet so user sees the new name immediately
      setSelectedItem((prev: any) => prev ? { ...prev, product_name: vars.displayName || prev.product_name } : prev)
      setEditingProductId(null)
    },
  })
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

  // "Used in dishes" — fetched lazily, only when bottom sheet is open with a selected item.
  const { data: usedIn = [], isLoading: usedInLoading } = useQuery<DishUsage[]>({
    queryKey: ['stock-used-in', selectedItem?.product_id],
    enabled: !!selectedItem?.product_id,
    queryFn: () =>
      api.get(`/stock/products/${selectedItem.product_id}/used-in`).then((r) => r.data),
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
    // Show all rows including negatives (data entry errors flagged with red).
    // Hide orphan products whose name is a raw UUID (deleted from iiko nomenclature).
    const base = [...(stock as any[])]
      .filter((i: any) => !isUuid(i.product_name))
      .sort((a: any, b: any) => {
      const aNeg = (a.amount || 0) < 0 ? -1 : 1
      const bNeg = (b.amount || 0) < 0 ? -1 : 1
      if (aNeg !== bNeg) return aNeg - bNeg
      return (b.cost_sum || 0) - (a.cost_sum || 0)
    })
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

  const rangeLabel = `${formatDay(dateFrom, locale)} - ${formatDay(dateTo, locale)}`

  const downloadExcel = () => {
    // Simple CSV export (opens in Excel natively)
    const header = ['Product', 'Amount', 'Unit', 'Cost'].join(',')
    const rows = (filtered as any[]).map((i: any) => {
      const unit = i.unit && !isUuid(i.unit) ? i.unit : t('common.piecesShort')
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
      <Header title={t('stock.title')} showBack showNotification badgeCount={unreadCount} />

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
            label={t('stock.inStock')}
          />
          <MetricCard
            icon={<TrendingDown className="h-4 w-4 text-primary" />}
            value={formatMoney(totals.writeOff) + cs}
            label={t('stock.writeOff')}
          />
        </div>
      </div>

      {/* Filters + Download Excel */}
      <div className="px-4 pb-3 flex items-center gap-3">
        <button
          className="flex flex-col items-center gap-0.5 shrink-0"
          aria-label={t('common.filter')}
          onClick={() => setActiveChip(null)}
        >
          <Filter className="h-5 w-5 text-dark" />
          <span className="text-[10px] text-gray">{t('common.filter')}</span>
        </button>
        <button
          onClick={downloadExcel}
          className="flex-1 flex items-center justify-center gap-2 bg-primary text-dark font-semibold py-3 rounded-full"
        >
          <Download className="h-4 w-4" />
          {t('stock.downloadExcel')}
        </button>
      </div>

      {/* Chip filters */}
      <div className="px-4 pb-3 flex items-center gap-2 overflow-x-auto">
        <ChipButton
          active={activeChip === 'stale'}
          onClick={() => setActiveChip(activeChip === 'stale' ? null : 'stale')}
        >
          {t('stock.chipStale')}
        </ChipButton>
        <ChipButton
          active={activeChip === 'lowstock'}
          onClick={() => setActiveChip(activeChip === 'lowstock' ? null : 'lowstock')}
        >
          {t('stock.chipLowStock')}
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
            <p className="text-sm text-gray">{t('stock.noStockMatching')}</p>
          </div>
        ) : (
          <div className="divide-y divide-bg-alt">
            {filtered.map((item: any) => {
              const isNegative = (item.amount || 0) < 0
              return (
                <button
                  key={item.product_id}
                  onClick={() => setSelectedItem(item)}
                  className="w-full text-left py-3"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1 min-w-0">
                      <p
                        className={cn(
                          'text-sm font-semibold truncate',
                          isNegative ? 'text-danger' : 'text-dark'
                        )}
                      >
                        {formatProductName(item.product_name)}
                      </p>
                      <p
                        className={cn(
                          'text-xs mt-0.5',
                          isNegative ? 'text-danger' : 'text-gray'
                        )}
                      >
                        {Math.round(item.amount || 0)}{' '}
                        {item.unit && !isUuid(item.unit) ? item.unit : t('common.piecesShort')}
                      </p>
                      {isNegative && (
                        <p className="text-[11px] text-danger font-medium mt-1 flex items-center gap-1">
                          <AlertTriangle className="h-3 w-3" />
                          {t('stock.makeInventory')}
                        </p>
                      )}
                    </div>
                    <p
                      className={cn(
                        'text-sm font-bold shrink-0',
                        isNegative ? 'text-danger' : 'text-dark'
                      )}
                    >
                      {formatMoney(item.cost_sum ?? 0)}
                      {cs}
                    </p>
                  </div>
                </button>
              )
            })}
          </div>
        )}
      </main>

      <Tabbar />

      <BottomSheet isOpen={!!selectedItem} onClose={() => setSelectedItem(null)}>
        {selectedItem && (
          <div className="space-y-4">
            {editingProductId === selectedItem.product_id ? (
              <div className="flex items-center gap-2">
                <input
                  autoFocus
                  value={aliasDraft}
                  onChange={(e) => setAliasDraft(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      aliasMutation.mutate({ productId: selectedItem.product_id, displayName: aliasDraft })
                    } else if (e.key === 'Escape') {
                      setEditingProductId(null)
                    }
                  }}
                  placeholder={t('stock.productNamePlaceholder')}
                  className="flex-1 min-w-0 bg-bg rounded-[10px] px-3 py-2 text-base font-bold text-dark outline-none border border-primary"
                />
                <button
                  onClick={() => aliasMutation.mutate({ productId: selectedItem.product_id, displayName: aliasDraft })}
                  disabled={aliasMutation.isPending}
                  className="w-8 h-8 rounded-full bg-primary flex items-center justify-center shrink-0"
                  aria-label={t('common.save')}
                >
                  <Check className="h-4 w-4 text-dark" />
                </button>
                <button
                  onClick={() => setEditingProductId(null)}
                  className="w-8 h-8 rounded-full bg-bg flex items-center justify-center shrink-0"
                  aria-label={t('common.cancel')}
                >
                  <X className="h-4 w-4 text-gray" />
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <p className="text-lg font-bold text-dark flex-1 min-w-0 truncate">
                  {formatProductName(selectedItem.product_name)}
                </p>
                <button
                  onClick={() => {
                    setEditingProductId(selectedItem.product_id)
                    const current = selectedItem.product_name || ''
                    setAliasDraft(isUuid(current) ? '' : current)
                  }}
                  className="w-7 h-7 rounded-full bg-bg flex items-center justify-center shrink-0 active:opacity-70"
                  aria-label={t('stock.editProductName')}
                >
                  <Pencil className="h-3.5 w-3.5 text-gray" />
                </button>
              </div>
            )}

            {(() => {
              const unit =
                selectedItem.unit && !isUuid(selectedItem.unit) ? selectedItem.unit : t('common.piecesShort')
              const isOverridden = !!selectedItem.override_at
              const startEdit = (field: 'amount' | 'price', initialValue: number) => {
                setEditingOverride(field)
                setOverrideDraft(String(initialValue || ''))
              }
              const submit = () => {
                const v = parseFloat(overrideDraft.replace(',', '.'))
                if (isNaN(v) || v < 0) return
                if (editingOverride === 'amount') {
                  overrideMutation.mutate({
                    productId: selectedItem.product_id,
                    manual_amount: v,
                  })
                } else if (editingOverride === 'price') {
                  overrideMutation.mutate({
                    productId: selectedItem.product_id,
                    manual_price_per_unit: v,
                  })
                }
              }
              return (
                <>
                  <div className="grid grid-cols-3 gap-2">
                    {/* Quantity */}
                    <EditableMetricCard
                      editing={editingOverride === 'amount'}
                      draft={overrideDraft}
                      setDraft={setOverrideDraft}
                      onSubmit={submit}
                      onCancel={() => setEditingOverride(null)}
                      onStartEdit={() => startEdit('amount', selectedItem.amount)}
                      suffix={unit}
                      value={`${formatMoney(selectedItem.amount || 0)} ${unit}`}
                      label={t('stock.quantity')}
                      isPending={overrideMutation.isPending && editingOverride === 'amount'}
                    />
                    {/* Price per unit */}
                    <EditableMetricCard
                      editing={editingOverride === 'price'}
                      draft={overrideDraft}
                      setDraft={setOverrideDraft}
                      onSubmit={submit}
                      onCancel={() => setEditingOverride(null)}
                      onStartEdit={() => startEdit('price', selectedItem.price_per_unit)}
                      suffix={`${cs}/${unit}`}
                      value={`${formatMoney(selectedItem.price_per_unit || 0)}${cs}/${unit}`}
                      label={t('stock.pricePerUnit')}
                      isPending={overrideMutation.isPending && editingOverride === 'price'}
                    />
                    {/* Total cost — read only */}
                    <div className="bg-bg rounded-[12px] p-2">
                      <p className="text-sm font-bold text-dark truncate">
                        {formatMoney(selectedItem.cost_sum ?? 0)}
                        {cs}
                      </p>
                      <p className="text-[10px] text-gray mt-0.5">{t('stock.costValue')}</p>
                    </div>
                  </div>
                  {isOverridden && (
                    <button
                      onClick={() =>
                        overrideMutation.mutate({
                          productId: selectedItem.product_id,
                          manual_amount: null,
                          manual_price_per_unit: null,
                        })
                      }
                      className="text-[11px] text-primary font-medium underline"
                    >
                      {t('stock.resetOverride')}
                    </button>
                  )}
                </>
              )
            })()}

            <div className="flex items-center justify-between text-sm border-t border-bg-alt pt-3">
              <span className="text-gray">{t('stock.lastSynced')}</span>
              <span className="font-semibold text-dark">
                {new Date(selectedItem.snapshot_at).toLocaleString(locale)}
              </span>
            </div>

            {/* Used in dishes — recipe components reverse lookup */}
            <div className="border-t border-bg-alt pt-3">
              <p className="text-sm text-gray mb-2">{t('stock.usedInDishes')}</p>
              {usedInLoading ? (
                <p className="text-xs text-gray">{t('common.loading')}</p>
              ) : usedIn.length === 0 ? (
                <p className="text-xs text-gray">
                  {t('stock.noRecipesForIngredient')}
                </p>
              ) : (
                <div className="space-y-1.5 max-h-60 overflow-y-auto">
                  {usedIn.map((d) => (
                    <div
                      key={d.dish_iiko_id}
                      className="flex items-center justify-between gap-2 text-sm"
                    >
                      <span className="text-dark truncate flex-1 min-w-0">
                        {formatProductName(d.dish_name)}
                      </span>
                      <span className="text-gray text-xs shrink-0 tabular-nums">
                        {formatRecipeAmount(d.amount)}{' '}
                        {d.unit && !isUuid(d.unit) ? d.unit : (selectedItem.unit && !isUuid(selectedItem.unit) ? selectedItem.unit : t('common.piecesShort'))}
                        {d.dish_unit ? ` / ${d.dish_unit}` : ''}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {(selectedItem.amount || 0) < 0 ? (
              <div className="bg-danger/5 border border-danger/20 rounded-[10px] p-3 flex items-center gap-2">
                <AlertTriangle className="h-4 w-4 text-danger shrink-0" />
                <span className="text-xs font-medium text-danger">
                  {t('stock.negativeStockWarning')}
                </span>
              </div>
            ) : lowStockIds.has(selectedItem.product_id) ? (
              <div className="bg-danger/5 border border-danger/20 rounded-[10px] p-3 flex items-center gap-2">
                <AlertTriangle className="h-4 w-4 text-danger shrink-0" />
                <span className="text-xs font-medium text-danger">{t('stock.lowStockWarning')}</span>
              </div>
            ) : null}

            <button
              onClick={() => setSelectedItem(null)}
              className="w-full text-center text-primary font-semibold py-2"
            >
              {t('common.back')}
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

function EditableMetricCard({
  editing,
  draft,
  setDraft,
  onSubmit,
  onCancel,
  onStartEdit,
  value,
  label,
  suffix,
  isPending,
}: {
  editing: boolean
  draft: string
  setDraft: (v: string) => void
  onSubmit: () => void
  onCancel: () => void
  onStartEdit: () => void
  value: string
  label: string
  suffix: string
  isPending: boolean
}) {
  if (editing) {
    return (
      <div className="bg-bg rounded-[12px] p-2 border border-primary">
        <div className="flex items-center gap-1">
          <input
            autoFocus
            type="number"
            inputMode="decimal"
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') onSubmit()
              if (e.key === 'Escape') onCancel()
            }}
            disabled={isPending}
            className="w-full min-w-0 bg-white rounded-md px-1.5 py-1 text-sm font-bold text-dark outline-none"
          />
          <button
            onClick={onSubmit}
            disabled={isPending}
            aria-label={t('common.save')}
            className="w-6 h-6 rounded-full bg-primary flex items-center justify-center shrink-0"
          >
            <Check className="h-3.5 w-3.5 text-dark" />
          </button>
        </div>
        <p className="text-[10px] text-gray mt-1 truncate">{suffix}</p>
      </div>
    )
  }
  return (
    <button
      onClick={onStartEdit}
      className="bg-bg rounded-[12px] p-2 text-left active:opacity-70 relative group"
    >
      <p className="text-sm font-bold text-dark truncate pr-3">{value}</p>
      <p className="text-[10px] text-gray mt-0.5 truncate">{label}</p>
      <Pencil className="h-3 w-3 text-gray absolute top-2 right-2 opacity-60" />
    </button>
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
