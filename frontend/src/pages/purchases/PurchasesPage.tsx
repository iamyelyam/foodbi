import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { CardSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { SegmentedControl } from '@/components/ui/segmented-control'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { DatePicker } from '@/components/ui/date-picker'
import { FilterChip } from '@/components/ui/filter-chip'
import { usePurchases, useSuppliers, useUnreadNotificationCount } from '@/hooks/useApi'
import { Filter, ChevronRight, Building2, Check, Receipt } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { useCurrency } from '@/stores/app'
import { useT } from '@/i18n'

type Tab = 'invoices' | 'suppliers'

export function PurchasesPage() {
  const cs = useCurrency()
  const t = useT()
  const [tab, setTab] = useState<Tab>('invoices')
  const [showFilters, setShowFilters] = useState(false)
  const [filters, setFilters] = useState<Record<string, string>>({})
  const [selectedSupplier, setSelectedSupplier] = useState<any>(null)
  const [pickingDate, setPickingDate] = useState<'from' | 'to' | null>(null)
  const [selectedPurchaseId, setSelectedPurchaseId] = useState<string | null>(null)

  const { data: purchasesData, isLoading: purchasesLoading } = usePurchases(filters)
  const { data: suppliers = [], isLoading: suppliersLoading } = useSuppliers()

  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const hasActiveFilters = filters.date_from || filters.date_to || filters.supplier_id
  const selectedSupplierName = filters.supplier_id
    ? suppliers.find((s: any) => String(s.supplier_id) === filters.supplier_id)?.supplier_name
    : null

  const purchases = purchasesData?.purchases ?? []

  const { data: purchaseDetail } = useQuery({
    queryKey: ['purchase-detail', selectedPurchaseId],
    queryFn: () => api.get(`/purchases/${selectedPurchaseId}`).then((r) => r.data),
    enabled: !!selectedPurchaseId,
  })

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={t('purchases.title')} showBack showNotification badgeCount={unreadCount} />

      <div className="px-4 pt-2 pb-3">
        <SegmentedControl
          value={tab}
          onChange={setTab}
          options={[
            { value: 'invoices', label: t('purchases.invoices') },
            { value: 'suppliers', label: t('purchases.suppliers') },
          ]}
        />
      </div>

      <div className="px-4 pb-3 flex items-center justify-between">
        <span className="text-xs text-gray">
          {tab === 'invoices' ? `${purchasesData?.total ?? 0} invoices` : `${suppliers.length} suppliers`}
        </span>
        {tab === 'invoices' && (
          <button onClick={() => setShowFilters(true)} className="flex items-center gap-1 text-xs font-medium text-primary">
            <Filter className="h-3.5 w-3.5" /> {t('common.filter')}
          </button>
        )}
      </div>

      {/* Active filter pills */}
      {tab === 'invoices' && hasActiveFilters && (
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
          {selectedSupplierName && (
            <FilterChip
              label={`Supplier: ${selectedSupplierName}`}
              onRemove={() => setFilters((f) => { const { supplier_id, ...rest } = f; return rest })}
            />
          )}
        </div>
      )}

      <main className="flex-1 px-4 pb-20 space-y-2">
        {(tab === 'invoices' ? purchasesLoading : suppliersLoading) ? (
          <>
            <CardSkeleton />
            <CardSkeleton />
            <CardSkeleton />
          </>
        ) : (
        <>
        {tab === 'invoices' &&
          purchases.map((p: any) => (
            <button key={p.id} onClick={() => setSelectedPurchaseId(p.id)} className="bg-white rounded-[12px] p-4 shadow-sm w-full text-left">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-semibold text-dark">{p.supplier_name || 'Unknown'}</p>
                  <p className="text-xs text-gray mt-0.5">
                    #{p.document_number} - {new Date(p.incoming_date).toLocaleDateString()}
                  </p>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold text-dark">{p.total_sum.toFixed(2)}{cs}</p>
                  <span className={cn(
                    'text-xs px-2 py-0.5 rounded-full',
                    p.status === 'processed' ? 'bg-success/10 text-success' : 'bg-bg-alt text-gray'
                  )}>
                    {p.status || 'pending'}
                  </span>
                </div>
              </div>
            </button>
          ))}

        {tab === 'suppliers' &&
          suppliers.map((s: any) => (
            <button
              key={s.supplier_id}
              onClick={() => setSelectedSupplier(s)}
              className="bg-white rounded-[12px] p-4 shadow-sm w-full text-left"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-primary-lighter flex items-center justify-center">
                    <Building2 className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <p className="text-sm font-semibold text-dark">{s.supplier_name}</p>
                    <p className="text-xs text-gray mt-0.5">{s.invoice_count} invoices</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <p className="text-sm font-semibold text-dark">{s.total_sum.toFixed(2)}{cs}</p>
                  <ChevronRight className="h-4 w-4 text-gray-light" />
                </div>
              </div>
            </button>
          ))}

        {((tab === 'invoices' && purchases.length === 0) || (tab === 'suppliers' && suppliers.length === 0)) && (
          <div className="text-center py-12">
            <Receipt className="h-12 w-12 text-gray-light mx-auto mb-3" />
            <p className="text-sm text-gray">No {tab} yet</p>
          </div>
        )}
        </>
        )}
      </main>

      <Tabbar />

      {/* Purchase detail sheet */}
      <BottomSheet isOpen={!!selectedPurchaseId} onClose={() => setSelectedPurchaseId(null)} title="Purchase Detail">
        {purchaseDetail && (
          <div className="space-y-4">
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-gray">Supplier</span>
                <span className="font-semibold text-dark">{purchaseDetail.supplier_name || 'Unknown'}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray">Document #</span>
                <span className="font-semibold text-dark">#{purchaseDetail.document_number}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray">Date</span>
                <span className="font-semibold text-dark">{new Date(purchaseDetail.incoming_date).toLocaleDateString()}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray">Total</span>
                <span className="font-semibold text-dark">{purchaseDetail.total_sum?.toFixed(2)}{cs}</span>
              </div>
            </div>

            <div className="border-t border-bg-alt pt-3">
              <p className="text-sm font-semibold text-dark mb-2">Line Items</p>
              {purchaseDetail.line_items && purchaseDetail.line_items.length > 0 ? (
                <div className="space-y-2">
                  {purchaseDetail.line_items.map((item: any, idx: number) => (
                    <div key={idx} className="bg-bg rounded-[12px] p-3">
                      <div className="flex items-center justify-between">
                        <p className="text-sm font-medium text-dark">{item.product_name}</p>
                        <p className="text-sm font-semibold text-dark">{item.subtotal?.toFixed(2)}{cs}</p>
                      </div>
                      <p className="text-xs text-gray mt-0.5">
                        {item.quantity} {item.unit} x {item.price?.toFixed(2)}{cs}
                      </p>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray text-center py-4">No line items available</p>
              )}
            </div>
          </div>
        )}
      </BottomSheet>

      {/* Supplier detail sheet */}
      <BottomSheet isOpen={!!selectedSupplier} onClose={() => setSelectedSupplier(null)} title={selectedSupplier?.supplier_name}>
        {selectedSupplier && (
          <div className="space-y-3">
            <div className="flex justify-between text-sm">
              <span className="text-gray">Total purchases</span>
              <span className="font-semibold">{selectedSupplier.total_sum.toFixed(2)}{cs}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray">Invoices</span>
              <span className="font-semibold">{selectedSupplier.invoice_count}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray">Last invoice</span>
              <span className="font-semibold">{selectedSupplier.last_invoice}</span>
            </div>
          </div>
        )}
      </BottomSheet>

      {/* Filters sheet */}
      <BottomSheet isOpen={showFilters} onClose={() => { setShowFilters(false); setPickingDate(null) }} title="Filters">
        <div className="space-y-4">
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

          {/* Supplier filter */}
          <div>
            <label className="text-xs font-medium text-gray mb-2 block">Supplier</label>
            <div className="max-h-40 overflow-y-auto space-y-1 rounded-[12px] border border-bg-alt p-2">
              <button
                onClick={() => setFilters((f) => { const { supplier_id, ...rest } = f; return rest })}
                className={cn(
                  'w-full flex items-center justify-between px-3 py-2 rounded-[8px] text-sm transition-colors',
                  !filters.supplier_id ? 'bg-primary/10 text-primary font-medium' : 'text-dark'
                )}
              >
                All suppliers
                {!filters.supplier_id && <Check className="h-4 w-4" />}
              </button>
              {suppliers.map((s: any) => (
                <button
                  key={s.supplier_id}
                  onClick={() => setFilters((f) => ({ ...f, supplier_id: String(s.supplier_id) }))}
                  className={cn(
                    'w-full flex items-center justify-between px-3 py-2 rounded-[8px] text-sm transition-colors',
                    filters.supplier_id === String(s.supplier_id) ? 'bg-primary/10 text-primary font-medium' : 'text-dark'
                  )}
                >
                  {s.supplier_name}
                  {filters.supplier_id === String(s.supplier_id) && <Check className="h-4 w-4" />}
                </button>
              ))}
            </div>
          </div>

          <div className="flex gap-3">
            <Button variant="secondary" fullWidth onClick={() => { setFilters({}); setShowFilters(false); setPickingDate(null) }}>{t('common.clear')}</Button>
            <Button fullWidth onClick={() => { setShowFilters(false); setPickingDate(null) }}>{t('common.apply')}</Button>
          </div>
        </div>
      </BottomSheet>
    </div>
  )
}
