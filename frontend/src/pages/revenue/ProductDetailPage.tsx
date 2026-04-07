import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { RevenueChart } from '@/components/charts/RevenueChart'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import api from '@/lib/api'

export function ProductDetailPage() {
  const { id } = useParams()

  const { data: sales = [], isLoading } = useQuery({
    queryKey: ['product-detail', id],
    queryFn: () => api.get(`/revenue/products/${id}`).then((r) => r.data),
    enabled: !!id,
  })

  const totalRev = sales.reduce((s: number, d: any) => s + d.revenue, 0)
  const totalQty = sales.reduce((s: number, d: any) => s + d.quantity, 0)

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="Product Details" showBack />
      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {isLoading ? <><ListItemSkeleton /><ListItemSkeleton /></> : (
          <>
            <div className="grid grid-cols-2 gap-3">
              <div className="bg-white rounded-[12px] p-3 shadow-sm">
                <p className="text-xs text-gray">Total Revenue</p>
                <p className="text-lg font-bold text-dark mt-1">${totalRev.toFixed(2)}</p>
              </div>
              <div className="bg-white rounded-[12px] p-3 shadow-sm">
                <p className="text-xs text-gray">Total Sold</p>
                <p className="text-lg font-bold text-dark mt-1">{totalQty.toFixed(1)}</p>
              </div>
            </div>

            {sales.length > 0 && (
              <div className="bg-white rounded-[16px] p-4 shadow-sm">
                <h3 className="text-sm font-semibold text-dark mb-3">Sales Trend (30 days)</h3>
                <RevenueChart data={sales.map((s: any) => ({ date: s.date, revenue: s.revenue }))} height={200} />
              </div>
            )}

            <div className="bg-white rounded-[16px] shadow-sm divide-y divide-bg-alt">
              {sales.map((s: any) => (
                <div key={s.date} className="flex items-center justify-between px-4 py-3">
                  <span className="text-sm text-dark">{new Date(s.date).toLocaleDateString()}</span>
                  <div className="text-right">
                    <p className="text-sm font-semibold text-dark">${s.revenue.toFixed(2)}</p>
                    <p className="text-xs text-gray">{s.quantity} sold</p>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </main>
      <Tabbar />
    </div>
  )
}
