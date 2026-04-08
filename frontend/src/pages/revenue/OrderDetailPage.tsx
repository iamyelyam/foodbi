import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { useCurrency } from '@/stores/app'

const STATUS_STYLES: Record<string, string> = {
  closed: 'bg-success/10 text-success',
  open: 'bg-warning/10 text-warning',
  review: 'bg-primary/10 text-primary',
  approved: 'bg-success/10 text-success',
  rejected: 'bg-danger/10 text-danger',
}

export function OrderDetailPage() {
  const { id } = useParams()
  const cs = useCurrency()
  const queryClient = useQueryClient()
  const { user } = useAuthStore()
  const canChangeStatus = user?.role === 'owner'
  const [showStatusSheet, setShowStatusSheet] = useState(false)

  const { data: order, isLoading } = useQuery({
    queryKey: ['order', id],
    queryFn: () => api.get(`/revenue/orders/${id}`).then((r) => r.data),
    enabled: !!id,
  })

  const statusMutation = useMutation({
    mutationFn: (status: 'approved' | 'rejected') =>
      api.post(`/revenue/orders/${id}/status`, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', id] })
      setShowStatusSheet(false)
    },
  })

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={`Order #${id?.slice(0, 8) || ''}`} showBack />
      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {isLoading ? (
          <>
            <ListItemSkeleton />
            <ListItemSkeleton />
            <ListItemSkeleton />
          </>
        ) : order ? (
          <>
            {/* Order header card */}
            <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray">Order #</span>
                <span className="text-sm font-semibold text-dark">{order.order_number || id?.slice(0, 8)}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray">Date</span>
                <span className="text-sm text-dark">{new Date(order.order_date).toLocaleString()}</span>
              </div>
              {order.waiter_name && (
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray">Waiter</span>
                  <span className="text-sm text-dark">{order.waiter_name}</span>
                </div>
              )}
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray">Type</span>
                <span className="text-sm text-dark capitalize">{order.order_type}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray">Total</span>
                <span className="text-xl font-bold text-dark">{order.revenue?.toFixed(2)}{cs}</span>
              </div>
              {order.discount > 0 && (
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray">Discount</span>
                  <span className="text-sm text-danger">-{order.discount?.toFixed(2)}{cs}</span>
                </div>
              )}
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray">Status</span>
                {canChangeStatus ? (
                  <button
                    onClick={() => setShowStatusSheet(true)}
                    className={cn(
                      'text-xs px-3 py-1 rounded-full font-medium transition-opacity active:opacity-70',
                      STATUS_STYLES[order.status] || 'bg-gray/10 text-gray'
                    )}
                  >
                    {order.status}
                  </button>
                ) : (
                  <span className={cn(
                    'text-xs px-3 py-1 rounded-full font-medium',
                    STATUS_STYLES[order.status] || 'bg-gray/10 text-gray'
                  )}>
                    {order.status}
                  </span>
                )}
              </div>
            </div>

            {/* Line items */}
            {order.items && order.items.length > 0 && (
              <div className="space-y-1">
                <h3 className="text-sm font-semibold text-dark px-1">Items ({order.items.length})</h3>
                <div className="bg-white rounded-[16px] shadow-sm divide-y divide-bg-alt">
                  {order.items.map((item: any, idx: number) => (
                    <div key={idx} className="flex items-center justify-between px-4 py-3">
                      <div className="flex-1 min-w-0 mr-3">
                        <p className="text-sm font-medium text-dark truncate">{item.product_name || item.name}</p>
                        <p className="text-xs text-gray">
                          {item.quantity} × {item.price?.toFixed(2)}{cs}
                        </p>
                      </div>
                      <span className="text-sm font-semibold text-dark shrink-0">
                        {(item.quantity * item.price)?.toFixed(2)}{cs}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Fallback item count if no detailed items */}
            {(!order.items || order.items.length === 0) && order.item_count > 0 && (
              <div className="bg-white rounded-[16px] p-4 shadow-sm">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray">Total items</span>
                  <span className="text-sm font-semibold text-dark">{order.item_count}</span>
                </div>
              </div>
            )}
          </>
        ) : (
          <p className="text-center text-sm text-gray py-12">Order not found</p>
        )}
      </main>
      <Tabbar />

      {/* Status change BottomSheet */}
      <BottomSheet
        isOpen={showStatusSheet}
        onClose={() => setShowStatusSheet(false)}
        title="Change Order Status"
      >
        <div className="space-y-3 pb-4">
          <p className="text-sm text-gray text-center">
            Current status: <span className="font-medium text-dark capitalize">{order?.status}</span>
          </p>
          <button
            onClick={() => statusMutation.mutate('approved')}
            disabled={statusMutation.isPending}
            className="w-full py-3 rounded-[12px] bg-success text-white font-semibold text-sm transition-opacity active:opacity-80 disabled:opacity-50"
          >
            {statusMutation.isPending ? 'Updating...' : 'Approve'}
          </button>
          <button
            onClick={() => statusMutation.mutate('rejected')}
            disabled={statusMutation.isPending}
            className="w-full py-3 rounded-[12px] border border-danger text-danger font-semibold text-sm transition-opacity active:opacity-80 disabled:opacity-50"
          >
            {statusMutation.isPending ? 'Updating...' : 'Reject'}
          </button>
          {statusMutation.isError && (
            <p className="text-xs text-danger text-center">Failed to update status. Try again.</p>
          )}
        </div>
      </BottomSheet>
    </div>
  )
}
