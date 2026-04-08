import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { RevenueChart } from '@/components/charts/RevenueChart'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import api from '@/lib/api'
import { useCurrency } from '@/stores/app'

export function ProductDetailPage() {
  const { id } = useParams()
  const cs = useCurrency()

  const { data: product, isLoading } = useQuery({
    queryKey: ['product-detail', id],
    queryFn: () => api.get(`/revenue/products/${id}`).then((r) => r.data),
    enabled: !!id,
  })

  const { data: trend = [] } = useQuery({
    queryKey: ['product-trend', id],
    queryFn: () => api.get(`/revenue/products/${id}/trend`).then((r) => r.data),
    enabled: !!id,
  })

  const { data: recentOrders = [] } = useQuery({
    queryKey: ['product-recent-orders', id],
    queryFn: () => api.get(`/revenue/products/${id}/orders?limit=10`).then((r) => r.data),
    enabled: !!id,
  })

  // Compute metrics from trend data
  const dailyRevenues = trend.map((d: any) => d.revenue)
  const totalRev = dailyRevenues.reduce((s: number, v: number) => s + v, 0)
  const totalQty = trend.reduce((s: number, d: any) => s + (d.quantity || 0), 0)
  const avgDaily = dailyRevenues.length > 0 ? totalRev / dailyRevenues.length : 0
  const bestDay = dailyRevenues.length > 0 ? Math.max(...dailyRevenues) : 0
  const worstDay = dailyRevenues.length > 0 ? Math.min(...dailyRevenues) : 0

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={product?.name || 'Product Details'} showBack />
      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {isLoading ? (
          <>
            <ListItemSkeleton />
            <ListItemSkeleton />
            <ListItemSkeleton />
          </>
        ) : (
          <>
            {/* Product header card */}
            <div className="bg-white rounded-[16px] p-4 shadow-sm">
              <h2 className="text-lg font-bold text-dark">{product?.name || 'Unknown Product'}</h2>
              {product?.category && (
                <span className="inline-block mt-1 text-xs px-2 py-0.5 rounded-full bg-primary/10 text-primary font-medium">
                  {product.category}
                </span>
              )}
              <div className="grid grid-cols-2 gap-3 mt-4">
                <div className="bg-bg rounded-[12px] p-3">
                  <p className="text-xs text-gray">Total Revenue</p>
                  <p className="text-lg font-bold text-dark mt-1">{totalRev.toFixed(2)}{cs}</p>
                </div>
                <div className="bg-bg rounded-[12px] p-3">
                  <p className="text-xs text-gray">Total Sold</p>
                  <p className="text-lg font-bold text-dark mt-1">{totalQty.toFixed(0)}</p>
                </div>
              </div>
            </div>

            {/* Key metrics row */}
            {trend.length > 0 && (
              <div className="grid grid-cols-3 gap-2">
                <div className="bg-white rounded-[12px] p-3 shadow-sm text-center">
                  <p className="text-xs text-gray">Avg Daily</p>
                  <p className="text-sm font-bold text-dark mt-1">{avgDaily.toFixed(0)}{cs}</p>
                </div>
                <div className="bg-white rounded-[12px] p-3 shadow-sm text-center">
                  <p className="text-xs text-gray">Best Day</p>
                  <p className="text-sm font-bold text-success mt-1">{bestDay.toFixed(0)}{cs}</p>
                </div>
                <div className="bg-white rounded-[12px] p-3 shadow-sm text-center">
                  <p className="text-xs text-gray">Worst Day</p>
                  <p className="text-sm font-bold text-danger mt-1">{worstDay.toFixed(0)}{cs}</p>
                </div>
              </div>
            )}

            {/* Daily sales trend chart */}
            {trend.length > 0 && (
              <div className="bg-white rounded-[16px] p-4 shadow-sm">
                <h3 className="text-sm font-semibold text-dark mb-3">Sales Trend (30 days)</h3>
                <RevenueChart data={trend.map((d: any) => ({ date: d.date, revenue: d.revenue }))} height={200} />
              </div>
            )}

            {/* Recent orders */}
            {recentOrders.length > 0 && (
              <div className="space-y-1">
                <h3 className="text-sm font-semibold text-dark px-1">Recent Orders</h3>
                <div className="bg-white rounded-[16px] shadow-sm divide-y divide-bg-alt">
                  {recentOrders.map((o: any) => (
                    <div key={o.id} className="flex items-center justify-between px-4 py-3">
                      <div>
                        <p className="text-sm font-medium text-dark">#{o.order_number || o.id?.slice(0, 8)}</p>
                        <p className="text-xs text-gray">{new Date(o.order_date).toLocaleDateString()}</p>
                      </div>
                      <div className="text-right">
                        <p className="text-sm font-semibold text-dark">{o.revenue?.toFixed(2)}{cs}</p>
                        <p className="text-xs text-gray">{o.quantity} pcs</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </main>
      <Tabbar />
    </div>
  )
}
