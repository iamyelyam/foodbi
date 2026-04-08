import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { useAppStore, useCurrency } from '@/stores/app'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import { Package, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { useT } from '@/i18n'

export function StockPage() {
  const cs = useCurrency()
  const t = useT()
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const locationId = useAppStore((s) => s.activeLocationId)
  const [selectedItem, setSelectedItem] = useState<any>(null)

  const { data: stock = [], isLoading } = useQuery({
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
      <Header title={t('stock.title')} showBack showNotification badgeCount={unreadCount} />

      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {lowStock.length > 0 && (
          <div className="bg-danger/5 border border-danger/20 rounded-[12px] p-3 flex items-center gap-3">
            <AlertTriangle className="h-5 w-5 text-danger shrink-0" />
            <div>
              <p className="text-sm font-semibold text-danger">{lowStock.length} {t('stock.lowStock')}</p>
              <p className="text-xs text-gray mt-0.5">{t('stock.needsRestocking')}</p>
            </div>
          </div>
        )}

        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-dark">{stock.length} products</h2>
        </div>

        {isLoading ? (
          <>
            <ListItemSkeleton />
            <ListItemSkeleton />
            <ListItemSkeleton />
          </>
        ) : (
        <>
        {stock.map((item: any) => {
          const isLow = lowStockIds.has(item.product_id)
          return (
            <button
              key={item.product_id}
              onClick={() => setSelectedItem(item)}
              className={cn(
                'w-full text-left bg-white rounded-[12px] p-4 shadow-sm',
                isLow && 'border border-danger/20'
              )}
            >
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
                  <p className="text-xs text-gray">{(item.cost_sum ?? 0).toLocaleString('ru-KZ', { maximumFractionDigits: 0 })}{cs}</p>
                </div>
              </div>
            </button>
          )
        })}

        {stock.length === 0 && (
          <div className="text-center py-12">
            <Package className="h-12 w-12 text-gray-light mx-auto mb-3" />
            <p className="text-sm text-gray">No stock data. Sync with iiko first.</p>
          </div>
        )}
        </>
        )}
      </main>

      <Tabbar />

      <BottomSheet isOpen={!!selectedItem} onClose={() => setSelectedItem(null)} title={selectedItem?.product_name}>
        {selectedItem && (
          <div className="space-y-3">
            <div className="flex justify-between text-sm">
              <span className="text-gray">Quantity</span>
              <span className="font-semibold text-dark">{selectedItem.amount} {selectedItem.unit}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray">Cost value</span>
              <span className="font-semibold text-dark">{(selectedItem.cost_sum ?? 0).toLocaleString('ru-KZ', { maximumFractionDigits: 0 })}{cs}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray">Last synced</span>
              <span className="font-semibold text-dark">{new Date(selectedItem.snapshot_at).toLocaleString()}</span>
            </div>
            {lowStockIds.has(selectedItem.product_id) && (
              <div className="bg-danger/5 border border-danger/20 rounded-[10px] p-3 flex items-center gap-2 mt-2">
                <AlertTriangle className="h-4 w-4 text-danger" />
                <span className="text-xs font-medium text-danger">Low stock — consider restocking</span>
              </div>
            )}
          </div>
        )}
      </BottomSheet>
    </div>
  )
}
