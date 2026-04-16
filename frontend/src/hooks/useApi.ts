import { useQuery } from '@tanstack/react-query'
import api from '@/lib/api'
import { useAppStore } from '@/stores/app'

/** Returns location filter params for API calls.
 *  When locations are selected → location_ids=id1,id2,...
 *  When none selected (all) → no param (backend returns all)
 */
function useLocationParams() {
  const ids = useAppStore((s) => s.selectedLocationIds)
  const param = ids.length > 0 ? ids.join(',') : undefined
  return { ids, param }
}

export function useDashboard(dateFrom?: string, dateTo?: string) {
  const loc = useLocationParams()
  return useQuery({
    queryKey: ['dashboard', loc.ids, dateFrom, dateTo],
    queryFn: () =>
      api.get('/dashboard/summary', { params: { location_ids: loc.param, date_from: dateFrom, date_to: dateTo } }).then((r) => r.data),
  })
}

export function useRevenueTrend(days: number = 7) {
  const loc = useLocationParams()
  return useQuery({
    queryKey: ['revenue-trend', loc.ids, days],
    queryFn: () =>
      api.get('/dashboard/revenue-trend', { params: { location_ids: loc.param, days } }).then((r) => r.data),
  })
}

export function useOrders(params: Record<string, string> = {}) {
  const loc = useLocationParams()
  return useQuery({
    queryKey: ['orders', loc.ids, params],
    queryFn: () =>
      api.get('/revenue/orders', { params: { location_ids: loc.param, ...params } }).then((r) => r.data),
  })
}

export function useProducts(params: Record<string, string> = {}) {
  const loc = useLocationParams()
  return useQuery({
    queryKey: ['products', loc.ids, params],
    queryFn: () =>
      api.get('/revenue/products', { params: { location_ids: loc.param, ...params } }).then((r) => r.data),
  })
}

export function usePurchases(params: Record<string, string> = {}) {
  const loc = useLocationParams()
  return useQuery({
    queryKey: ['purchases', loc.ids, params],
    queryFn: () =>
      api.get('/purchases', { params: { location_ids: loc.param, ...params } }).then((r) => r.data),
  })
}

export function useSuppliers() {
  return useQuery({
    queryKey: ['suppliers'],
    queryFn: () => api.get('/purchases/suppliers').then((r) => r.data),
  })
}

export function useRevenueStats(dateFrom?: string, dateTo?: string) {
  const loc = useLocationParams()
  return useQuery({
    queryKey: ['stats-revenue', loc.ids, dateFrom, dateTo],
    queryFn: () =>
      api.get('/statistics/revenue', { params: { location_ids: loc.param, date_from: dateFrom, date_to: dateTo } }).then((r) => r.data),
  })
}

export function useUnreadNotificationCount() {
  return useQuery({
    queryKey: ['notifications-unread-count'],
    queryFn: () => api.get('/notifications/unread-count').then((r) => r.data?.count ?? 0),
    refetchInterval: 30_000,
  })
}

export function useProfitStats(dateFrom?: string, dateTo?: string) {
  const loc = useLocationParams()
  return useQuery({
    queryKey: ['stats-profit', loc.ids, dateFrom, dateTo],
    queryFn: () =>
      api.get('/statistics/profit', { params: { location_ids: loc.param, date_from: dateFrom, date_to: dateTo } }).then((r) => r.data),
  })
}
