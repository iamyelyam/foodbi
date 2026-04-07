import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import api from '@/lib/api'

export function OrderDetailPage() {
  const { id } = useParams()

  const { data: order, isLoading } = useQuery({
    queryKey: ['order', id],
    queryFn: () => api.get(`/revenue/orders/${id}`).then((r) => r.data),
    enabled: !!id,
  })

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={`Order ${id?.slice(0, 8) || ''}`} showBack />
      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {isLoading ? <><ListItemSkeleton /><ListItemSkeleton /></> : order ? (
          <>
            <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray">Revenue</span>
                <span className="text-xl font-bold text-dark">${order.revenue?.toFixed(2)}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray">Discount</span>
                <span className="text-sm text-danger">${order.discount?.toFixed(2)}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray">Items</span>
                <span className="text-sm text-dark">{order.item_count}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray">Status</span>
                <span className={cn('text-xs px-2 py-0.5 rounded-full font-medium',
                  order.status === 'closed' ? 'bg-success/10 text-success' : 'bg-warning/10 text-warning'
                )}>{order.status}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray">Type</span>
                <span className="text-sm text-dark capitalize">{order.order_type}</span>
              </div>
              {order.waiter_name && (
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray">Waiter</span>
                  <span className="text-sm text-dark">{order.waiter_name}</span>
                </div>
              )}
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray">Date</span>
                <span className="text-sm text-dark">{new Date(order.order_date).toLocaleString()}</span>
              </div>
            </div>
          </>
        ) : (
          <p className="text-center text-sm text-gray py-12">Order not found</p>
        )}
      </main>
      <Tabbar />
    </div>
  )
}
