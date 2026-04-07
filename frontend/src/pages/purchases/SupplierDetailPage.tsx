import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { Building2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'

export function SupplierDetailPage() {
  const { id } = useParams()

  const { data: supplier, isLoading } = useQuery({
    queryKey: ['supplier', id],
    queryFn: () => api.get(`/purchases/suppliers/${id}`).then((r) => r.data),
    enabled: !!id,
  })

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={supplier?.supplier_name || 'Supplier'} showBack />
      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {isLoading ? <><ListItemSkeleton /><ListItemSkeleton /><ListItemSkeleton /></> : supplier ? (
          <>
            {/* Supplier info */}
            <div className="bg-white rounded-[16px] p-6 shadow-sm flex flex-col items-center">
              <div className="w-16 h-16 rounded-full bg-primary-lighter flex items-center justify-center">
                <Building2 className="h-8 w-8 text-primary" />
              </div>
              <p className="mt-3 text-lg font-bold text-dark">{supplier.supplier_name}</p>
            </div>

            {/* Stats */}
            <div className="grid grid-cols-3 gap-2">
              <div className="bg-white rounded-[12px] p-3 shadow-sm text-center">
                <p className="text-xs text-gray">Total Spend</p>
                <p className="text-sm font-bold text-dark mt-1">${supplier.total_sum?.toFixed(0)}</p>
              </div>
              <div className="bg-white rounded-[12px] p-3 shadow-sm text-center">
                <p className="text-xs text-gray">Invoices</p>
                <p className="text-sm font-bold text-dark mt-1">{supplier.invoice_count}</p>
              </div>
              <div className="bg-white rounded-[12px] p-3 shadow-sm text-center">
                <p className="text-xs text-gray">Last Order</p>
                <p className="text-sm font-bold text-dark mt-1">{supplier.last_invoice}</p>
              </div>
            </div>

            {/* Recent purchases */}
            <h3 className="text-sm font-semibold text-dark">Recent Invoices</h3>
            <div className="bg-white rounded-[16px] shadow-sm divide-y divide-bg-alt">
              {(supplier.purchases || []).map((p: any) => (
                <div key={p.id} className="flex items-center justify-between px-4 py-3">
                  <div>
                    <p className="text-sm font-medium text-dark">#{p.document_number || 'N/A'}</p>
                    <p className="text-xs text-gray">{new Date(p.incoming_date).toLocaleDateString()}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-semibold text-dark">${p.total_sum?.toFixed(2)}</p>
                    <span className={cn('text-xs', p.status === 'processed' ? 'text-success' : 'text-gray')}>{p.status}</span>
                  </div>
                </div>
              ))}
              {(!supplier.purchases || supplier.purchases.length === 0) && (
                <p className="text-sm text-gray text-center py-6">No invoices yet</p>
              )}
            </div>
          </>
        ) : (
          <p className="text-center text-sm text-gray py-12">Supplier not found</p>
        )}
      </main>
      <Tabbar />
    </div>
  )
}
