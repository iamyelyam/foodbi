import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CardSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { DateRangeSheet } from '@/components/layout/DateRangeSheet'
import { usePurchases, useSuppliers, useUnreadNotificationCount } from '@/hooks/useApi'
import { Filter, ChevronRight, Calendar, Coins, Receipt, ScanLine, Pencil, Check, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { formatProductName, formatSupplierName } from '@/lib/format'
import { useAppStore, useCurrency } from '@/stores/app'
import { useT, useI18nStore } from '@/i18n'

function formatDay(dateStr: string, locale: string = 'en'): string {
  const d = new Date(dateStr + 'T00:00:00')
  return d.toLocaleDateString(locale, { month: 'long', day: 'numeric' })
}
// todayIso / isoDaysAgo removed — dates now from global useAppStore
const formatMoney = (v: number) => v.toLocaleString('ru-KZ', { maximumFractionDigits: 0 })

export function PurchasesPage() {
  const cs = useCurrency()
  const t = useT()
  const locale = useI18nStore((s) => s.locale)
  const navigate = useNavigate()

  // Purchases happen less often than orders — default to last 30 days.
  const dateFrom = useAppStore((s) => s.dateFrom)
  const dateTo = useAppStore((s) => s.dateTo)
  const setDateRange = useAppStore((s) => s.setDateRange)
  const [showRangePicker, setShowRangePicker] = useState(false)
  const [showFilters, setShowFilters] = useState(false)
  const [suppliersFilter, setSuppliersFilter] = useState<Set<string>>(new Set())
  const [selectedPurchaseId, setSelectedPurchaseId] = useState<string | null>(null)
  const [editingSupplier, setEditingSupplier] = useState<string | null>(null)
  const [aliasDraft, setAliasDraft] = useState<string>('')
  const queryClient = useQueryClient()

  const aliasMutation = useMutation({
    mutationFn: ({ supplierId, displayName }: { supplierId: string; displayName: string }) =>
      api.put(`/purchases/suppliers/${supplierId}/alias`, { display_name: displayName }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['purchases'] })
      queryClient.invalidateQueries({ queryKey: ['purchase-detail'] })
      queryClient.invalidateQueries({ queryKey: ['suppliers'] })
      setEditingSupplier(null)
    },
  })

  const { data: purchasesData, isLoading: purchasesLoading } = usePurchases({
    date_from: dateFrom,
    date_to: dateTo,
  })
  const { data: suppliers = [] } = useSuppliers()
  const { data: unreadCount = 0 } = useUnreadNotificationCount()

  const rawPurchases = purchasesData?.purchases ?? []

  const purchases = useMemo(() => {
    if (suppliersFilter.size === 0) return rawPurchases
    return (rawPurchases as any[]).filter((p: any) =>
      suppliersFilter.has(String(p.supplier_name || ''))
    )
  }, [rawPurchases, suppliersFilter])

  const totals = useMemo(() => {
    const sum = (purchases as any[]).reduce((s: number, p: any) => s + (p.total_sum || 0), 0)
    return { sum, count: purchases.length }
  }, [purchases])

  const { data: purchaseDetail } = useQuery({
    queryKey: ['purchase-detail', selectedPurchaseId],
    queryFn: () => api.get(`/purchases/${selectedPurchaseId}`).then((r) => r.data),
    enabled: !!selectedPurchaseId,
  })

  const rangeLabel = `${formatDay(dateFrom, locale)} - ${formatDay(dateTo, locale)}`

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <Header title={t('purchases.title')} showBack showNotification badgeCount={unreadCount} />

      {/* Date range */}
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

      {/* 2 metric cards */}
      <div className="px-4 pb-3">
        <div className="grid grid-cols-2 gap-2">
          <MetricCard
            icon={<Coins className="h-4 w-4 text-primary" />}
            value={formatMoney(totals.sum) + cs}
            label={t('purchases.title')}
          />
          <MetricCard
            icon={<Receipt className="h-4 w-4 text-primary" />}
            value={String(totals.count)}
            label={t('purchases.invoices')}
          />
        </div>
      </div>

      {/* Filters + Scan a file action */}
      <div className="px-4 pb-3 flex items-center gap-3">
        <button
          onClick={() => setShowFilters(true)}
          className="flex flex-col items-center gap-0.5 shrink-0"
          aria-label={t('common.filter')}
        >
          <Filter className="h-5 w-5 text-dark" />
          <span className="text-[10px] text-gray">{t('common.filter')}</span>
        </button>
        <button
          onClick={() => navigate('/file-upload')}
          className="flex-1 flex items-center justify-center gap-2 bg-primary text-dark font-semibold py-3 rounded-full"
        >
          <ScanLine className="h-4 w-4" />
          {t('purchases.scanFile')}
        </button>
      </div>

      <main className="flex-1 px-4 pb-28">
        {purchasesLoading ? (
          <div className="space-y-2">
            <CardSkeleton />
            <CardSkeleton />
            <CardSkeleton />
          </div>
        ) : (
          <div className="divide-y divide-bg-alt">
            {(purchases as any[]).map((p: any) => (
              <button
                key={p.id}
                onClick={() => setSelectedPurchaseId(p.id)}
                className="w-full text-left py-3"
              >
                <div className="flex items-center justify-between">
                  <div className="flex-1 min-w-0 mr-3">
                    <p className="text-sm font-semibold text-dark truncate">
                      {formatSupplierName(p.supplier_name)}
                    </p>
                    <p className="text-xs text-gray mt-0.5">
                      {new Date(p.incoming_date).toLocaleDateString(locale)}
                    </p>
                  </div>
                  <p className="text-sm font-bold text-dark shrink-0">
                    {formatMoney(p.total_sum)}{cs}
                  </p>
                </div>
              </button>
            ))}
            {purchases.length === 0 && (
              <div className="text-center py-12">
                <Receipt className="h-12 w-12 text-gray-light mx-auto mb-3" />
                <p className="text-sm text-gray">{t('purchases.noInvoicesForPeriod')}</p>
              </div>
            )}
          </div>
        )}
      </main>

      <Tabbar />

      {/* Purchase detail sheet */}
      <BottomSheet
        isOpen={!!selectedPurchaseId}
        onClose={() => setSelectedPurchaseId(null)}
      >
        {purchaseDetail && (
          <div className="space-y-4">
            <div className="flex items-start justify-between gap-2">
              <div className="flex-1 min-w-0">
                {editingSupplier === purchaseDetail.supplier_id ? (
                  <div className="flex items-center gap-2">
                    <input
                      autoFocus
                      value={aliasDraft}
                      onChange={(e) => setAliasDraft(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          aliasMutation.mutate({
                            supplierId: purchaseDetail.supplier_id,
                            displayName: aliasDraft,
                          })
                        } else if (e.key === 'Escape') {
                          setEditingSupplier(null)
                        }
                      }}
                      placeholder={t('purchases.supplierNamePlaceholder')}
                      className="flex-1 min-w-0 bg-bg rounded-[10px] px-3 py-2 text-base font-bold text-dark outline-none border border-primary"
                    />
                    <button
                      onClick={() =>
                        aliasMutation.mutate({
                          supplierId: purchaseDetail.supplier_id,
                          displayName: aliasDraft,
                        })
                      }
                      disabled={aliasMutation.isPending}
                      className="w-8 h-8 rounded-full bg-primary flex items-center justify-center shrink-0"
                    >
                      <Check className="h-4 w-4 text-dark" />
                    </button>
                    <button
                      onClick={() => setEditingSupplier(null)}
                      className="w-8 h-8 rounded-full bg-bg flex items-center justify-center shrink-0"
                    >
                      <X className="h-4 w-4 text-gray" />
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <p className="text-lg font-bold text-dark truncate">
                      {formatSupplierName(purchaseDetail.supplier_name)}
                    </p>
                    {purchaseDetail.supplier_id && (
                      <button
                        onClick={() => {
                          setEditingSupplier(purchaseDetail.supplier_id)
                          // Pre-fill with current name if it's not a UUID
                          const current = purchaseDetail.supplier_name || ''
                          const isUuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(current)
                          setAliasDraft(isUuid ? '' : current)
                        }}
                        className="w-7 h-7 rounded-full bg-bg flex items-center justify-center shrink-0 active:opacity-70"
                        aria-label={t('purchases.editSupplier')}
                      >
                        <Pencil className="h-3.5 w-3.5 text-gray" />
                      </button>
                    )}
                  </div>
                )}
              </div>
              <p className="text-sm text-gray shrink-0 pt-2">
                {new Date(purchaseDetail.incoming_date).toLocaleDateString(locale)}
              </p>
            </div>

            <div className="grid grid-cols-2 gap-2">
              <div className="bg-bg rounded-[12px] p-3">
                <p className="text-sm font-bold text-dark">
                  {formatMoney(purchaseDetail.total_sum || 0)}{cs}
                </p>
                <p className="text-[10px] text-gray mt-0.5">{t('revenue.totalLabel')}</p>
              </div>
              <div className="bg-bg rounded-[12px] p-3">
                <p className="text-sm font-bold text-dark">
                  #{purchaseDetail.document_number || '—'}
                </p>
                <p className="text-[10px] text-gray mt-0.5">{t('purchases.document')}</p>
              </div>
            </div>

            {purchaseDetail.line_items && purchaseDetail.line_items.length > 0 ? (
              <div className="divide-y divide-bg-alt">
                {purchaseDetail.line_items.map((item: any, idx: number) => (
                  <div key={idx} className="flex items-center justify-between py-3">
                    <div className="flex-1 min-w-0 mr-3">
                      <p className="text-sm font-semibold text-dark truncate">
                        {formatProductName(item.product_name)}
                      </p>
                      <p className="text-xs text-gray mt-0.5">
                        {item.quantity} {item.unit} × {formatMoney(item.price || 0)}{cs}
                      </p>
                    </div>
                    <p className="text-sm font-bold text-dark shrink-0">
                      {formatMoney(item.subtotal || 0)}{cs}
                    </p>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray text-center py-4">{t('purchases.noLineItems')}</p>
            )}

            <button
              onClick={() => setSelectedPurchaseId(null)}
              className="w-full text-center text-primary font-semibold py-2"
            >
              {t('common.back')}
            </button>
          </div>
        )}
      </BottomSheet>

      {/* Filters sheet */}
      <BottomSheet isOpen={showFilters} onClose={() => setShowFilters(false)}>
        <div className="space-y-5">
          <div>
            <p className="text-base font-bold text-dark mb-3">{t('purchases.supplierFilter')}</p>
            {suppliers.length === 0 ? (
              <p className="text-xs text-gray">{t('purchases.noSuppliersLoaded')}</p>
            ) : (
              <div className="flex flex-wrap gap-2 max-h-60 overflow-y-auto">
                {(suppliers as any[]).map((s: any) => {
                  const name = s.supplier_name as string
                  const active = suppliersFilter.has(name)
                  return (
                    <button
                      key={s.supplier_id}
                      onClick={() => {
                        setSuppliersFilter((prev) => {
                          const next = new Set(prev)
                          if (next.has(name)) next.delete(name)
                          else next.add(name)
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
                      {formatProductName(name)}
                    </button>
                  )
                })}
              </div>
            )}
          </div>

          <button
            onClick={() => setShowFilters(false)}
            className="w-full bg-primary text-dark font-bold py-3 rounded-full"
          >
            {t('common.showResults', { count: purchases.length })}
          </button>
          <button
            onClick={() => setShowFilters(false)}
            className="w-full text-center text-primary font-semibold"
          >
            {t('common.back')}
          </button>
        </div>
      </BottomSheet>

      {/* Date range picker — unified BottomSheet shared across pages */}
      <DateRangeSheet
        isOpen={showRangePicker}
        onClose={() => setShowRangePicker(false)}
        from={dateFrom}
        to={dateTo}
        onChange={(f, t) => setDateRange(f, t)}
        resultsCount={purchases.length}
      />
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
