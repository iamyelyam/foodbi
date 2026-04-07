import { useQuery } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { useAppStore } from '@/stores/app'
import { Package, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'

export function StockPage() {
  const locationId = useAppStore((s) => s.activeLocationId)

  const { data: stock = [] } = useQuery({
    queryKey: ['stock', locationId],
    queryFn: () => api.get('/stock', { params: { location_id: locationId } }).then((r) => r.data),
  })

  const { data: lowStock = [] } = useQuery({
    queryKey: ['low-stock', locationId],
    queryFn: () => api.get('/stock/low-stock', { params: { location_id: locationId } }).then((r) => r.data),
  })

  const lowStockIds = new Set(lowStock.map((i: any) => i.product_id))

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="Stock Management" showBack showNotification />

      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {lowStock.length > 0 && (
          <div className="bg-danger/5 border border-danger/20 rounded-[12px] p-3 flex items-center gap-3">
            <AlertTriangle className="h-5 w-5 text-danger shrink-0" />
            <div>
              <p className="text-sm font-semibold text-danger">{lowStock.length} low stock items</p>
              <p className="text-xs text-gray mt-0.5">Items below threshold need restocking</p>
            </div>
          </div>
        )}

        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-dark">{stock.length} products</h2>
        </div>

        {stock.map((item: any) => {
          const isLow = lowStockIds.has(item.product_id)
          return (
            <div key={item.product_id} className={cn(
              'bg-white rounded-[12px] p-4 shadow-sm',
              isLow && 'border border-danger/20'
            )}>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className={cn(
                    'w-10 h-10 rounded-full flex items-center justify-center',
                    isLow ? 'bg-danger/10' : 'bg-primary-lighter'
                  )}>
                    <Package className={cn('h-5 w-5', isLow ? 'text-danger' : 'text-primary')} />
                  </div>
                  <div>
                    <p className="text-sm font-semibold text-dark">{item.product_name}</p>
                    <p className="text-xs text-gray mt-0.5">
                      Updated {new Date(item.snapshot_at).toLocaleTimeString()}
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <p className={cn('text-sm font-bold', isLow ? 'text-danger' : 'text-dark')}>
                    {item.amount} {item.unit}
                  </p>
                  <p className="text-xs text-gray">€{item.cost_sum.toFixed(2)}</p>
                </div>
              </div>
            </div>
          )
        })}

        {stock.length === 0 && (
          <div className="text-center py-12">
            <Package className="h-12 w-12 text-gray-light mx-auto mb-3" />
            <p className="text-sm text-gray">No stock data. Sync with iiko first.</p>
          </div>
        )}
      </main>

      <Tabbar />
    </div>
  )
}
