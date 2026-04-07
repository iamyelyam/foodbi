import { useState } from 'react'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { usePurchases, useSuppliers } from '@/hooks/useApi'
import { Filter, ChevronRight, Building2 } from 'lucide-react'
import { cn } from '@/lib/utils'

type Tab = 'invoices' | 'suppliers'

export function PurchasesPage() {
  const [tab, setTab] = useState<Tab>('invoices')
  const [showFilters, setShowFilters] = useState(false)
  const [filters, setFilters] = useState<Record<string, string>>({})
  const [selectedSupplier, setSelectedSupplier] = useState<any>(null)

  const { data: purchasesData } = usePurchases(filters)
  const { data: suppliers = [] } = useSuppliers()

  const purchases = purchasesData?.purchases ?? []

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="Purchases" showBack showNotification />

      <div className="px-4 pt-2 pb-3">
        <div className="flex bg-bg-alt rounded-[12px] p-1">
          {(['invoices', 'suppliers'] as Tab[]).map((t) => (
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

      <div className="px-4 pb-3 flex items-center justify-between">
        <span className="text-xs text-gray">
          {tab === 'invoices' ? `${purchasesData?.total ?? 0} invoices` : `${suppliers.length} suppliers`}
        </span>
        {tab === 'invoices' && (
          <button onClick={() => setShowFilters(true)} className="flex items-center gap-1 text-xs font-medium text-primary">
            <Filter className="h-3.5 w-3.5" /> Filters
          </button>
        )}
      </div>

      <main className="flex-1 px-4 pb-20 space-y-2">
        {tab === 'invoices' &&
          purchases.map((p: any) => (
            <div key={p.id} className="bg-white rounded-[12px] p-4 shadow-sm">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-semibold text-dark">{p.supplier_name || 'Unknown'}</p>
                  <p className="text-xs text-gray mt-0.5">
                    #{p.document_number} - {new Date(p.incoming_date).toLocaleDateString()}
                  </p>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold text-dark">${p.total_sum.toFixed(2)}</p>
                  <span className={cn(
                    'text-xs px-2 py-0.5 rounded-full',
                    p.status === 'processed' ? 'bg-success/10 text-success' : 'bg-bg-alt text-gray'
                  )}>
                    {p.status || 'pending'}
                  </span>
                </div>
              </div>
            </div>
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
                  <p className="text-sm font-semibold text-dark">${s.total_sum.toFixed(2)}</p>
                  <ChevronRight className="h-4 w-4 text-gray-light" />
                </div>
              </div>
            </button>
          ))}

        {((tab === 'invoices' && purchases.length === 0) || (tab === 'suppliers' && suppliers.length === 0)) && (
          <div className="text-center py-12">
            <p className="text-sm text-gray">No data yet.</p>
          </div>
        )}
      </main>

      <Tabbar />

      {/* Supplier detail sheet */}
      <BottomSheet isOpen={!!selectedSupplier} onClose={() => setSelectedSupplier(null)} title={selectedSupplier?.supplier_name}>
        {selectedSupplier && (
          <div className="space-y-3">
            <div className="flex justify-between text-sm">
              <span className="text-gray">Total purchases</span>
              <span className="font-semibold">${selectedSupplier.total_sum.toFixed(2)}</span>
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
      <BottomSheet isOpen={showFilters} onClose={() => setShowFilters(false)} title="Filters">
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium text-gray">Date from</label>
            <input type="date" className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4"
              value={filters.date_from || ''} onChange={(e) => setFilters(f => ({ ...f, date_from: e.target.value }))} />
          </div>
          <div>
            <label className="text-sm font-medium text-gray">Date to</label>
            <input type="date" className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4"
              value={filters.date_to || ''} onChange={(e) => setFilters(f => ({ ...f, date_to: e.target.value }))} />
          </div>
          <div className="flex gap-3">
            <Button variant="secondary" fullWidth onClick={() => { setFilters({}); setShowFilters(false) }}>Clear</Button>
            <Button fullWidth onClick={() => setShowFilters(false)}>Apply</Button>
          </div>
        </div>
      </BottomSheet>
    </div>
  )
}
