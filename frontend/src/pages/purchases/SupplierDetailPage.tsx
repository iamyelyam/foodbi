import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { Building2, Phone, Mail, MapPin } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { useCurrency } from '@/stores/app'

export function SupplierDetailPage() {
  const { id } = useParams()
  const cs = useCurrency()

  const { data: supplier, isLoading } = useQuery({
    queryKey: ['supplier', id],
    queryFn: () => api.get(`/purchases/suppliers/${id}`).then((r) => r.data),
    enabled: !!id,
  })

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={supplier?.supplier_name || 'Supplier'} showBack />
      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {isLoading ? (
          <>
            <ListItemSkeleton />
            <ListItemSkeleton />
            <ListItemSkeleton />
          </>
        ) : supplier ? (
          <>
            {/* Supplier header card */}
            <div className="bg-white rounded-[16px] p-6 shadow-sm flex flex-col items-center">
              <div className="w-16 h-16 rounded-full bg-primary-lighter flex items-center justify-center">
                <Building2 className="h-8 w-8 text-primary" />
              </div>
              <p className="mt-3 text-lg font-bold text-dark">{supplier.supplier_name}</p>
              <div className="grid grid-cols-2 gap-3 mt-4 w-full">
                <div className="bg-bg rounded-[12px] p-3 text-center">
                  <p className="text-xs text-gray">Total Spend</p>
                  <p className="text-lg font-bold text-dark mt-1">{supplier.total_sum?.toFixed(0)}{cs}</p>
                </div>
                <div className="bg-bg rounded-[12px] p-3 text-center">
                  <p className="text-xs text-gray">Invoices</p>
                  <p className="text-lg font-bold text-dark mt-1">{supplier.invoice_count}</p>
                </div>
              </div>
            </div>

            {/* Contact info section */}
            <div className="bg-white rounded-[16px] shadow-sm divide-y divide-bg-alt">
              <div className="flex items-center gap-3 px-4 py-3">
                <Phone className="h-4 w-4 text-gray shrink-0" />
                <div>
                  <p className="text-xs text-gray">Phone</p>
                  <p className="text-sm text-dark">{supplier.phone || 'Not available'}</p>
                </div>
              </div>
              <div className="flex items-center gap-3 px-4 py-3">
                <Mail className="h-4 w-4 text-gray shrink-0" />
                <div>
                  <p className="text-xs text-gray">Email</p>
                  <p className="text-sm text-dark">{supplier.email || 'Not available'}</p>
                </div>
              </div>
              <div className="flex items-center gap-3 px-4 py-3">
                <MapPin className="h-4 w-4 text-gray shrink-0" />
                <div>
                  <p className="text-xs text-gray">Address</p>
                  <p className="text-sm text-dark">{supplier.address || 'Not available'}</p>
                </div>
              </div>
            </div>

            {/* Purchase history */}
            <div className="space-y-1">
              <h3 className="text-sm font-semibold text-dark px-1">Purchase History</h3>
              <div className="bg-white rounded-[16px] shadow-sm divide-y divide-bg-alt">
                {(supplier.purchases || []).map((p: any) => (
                  <div key={p.id} className="flex items-center justify-between px-4 py-3">
                    <div>
                      <p className="text-sm font-medium text-dark">#{p.document_number || 'N/A'}</p>
                      <p className="text-xs text-gray">{new Date(p.incoming_date).toLocaleDateString()}</p>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-semibold text-dark">{p.total_sum?.toFixed(2)}{cs}</p>
                      <span
                        className={cn(
                          'text-xs font-medium',
                          p.status === 'processed' ? 'text-success' : 'text-gray'
                        )}
                      >
                        {p.status}
                      </span>
                    </div>
                  </div>
                ))}
                {(!supplier.purchases || supplier.purchases.length === 0) && (
                  <p className="text-sm text-gray text-center py-6">No invoices yet</p>
                )}
              </div>
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
