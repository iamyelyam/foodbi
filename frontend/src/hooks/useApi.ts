import { useQuery } from '@tanstack/react-query'
import api from '@/lib/api'
import { useAppStore } from '@/stores/app'

export function useDashboard() {
  const locationId = useAppStore((s) => s.activeLocationId)
  return useQuery({
    queryKey: ['dashboard', locationId],
    queryFn: () =>
      api.get('/dashboard/summary', { params: { location_id: locationId } }).then((r) => r.data),
  })
}

export function useRevenueTrend(days: number = 7) {
  const locationId = useAppStore((s) => s.activeLocationId)
  return useQuery({
    queryKey: ['revenue-trend', locationId, days],
    queryFn: () =>
      api.get('/dashboard/revenue-trend', { params: { location_id: locationId, days } }).then((r) => r.data),
  })
}

export function useOrders(params: Record<string, string> = {}) {
  const locationId = useAppStore((s) => s.activeLocationId)
  return useQuery({
    queryKey: ['orders', locationId, params],
    queryFn: () =>
      api.get('/revenue/orders', { params: { location_id: locationId, ...params } }).then((r) => r.data),
  })
}

export function useProducts(params: Record<string, string> = {}) {
  const locationId = useAppStore((s) => s.activeLocationId)
  return useQuery({
    queryKey: ['products', locationId, params],
    queryFn: () =>
      api.get('/revenue/products', { params: { location_id: locationId, ...params } }).then((r) => r.data),
  })
}

export function usePurchases(params: Record<string, string> = {}) {
  const locationId = useAppStore((s) => s.activeLocationId)
  return useQuery({
    queryKey: ['purchases', locationId, params],
    queryFn: () =>
      api.get('/purchases', { params: { location_id: locationId, ...params } }).then((r) => r.data),
  })
}

export function useSuppliers() {
  return useQuery({
    queryKey: ['suppliers'],
    queryFn: () => api.get('/purchases/suppliers').then((r) => r.data),
  })
}

export function useRevenueStats(dateFrom?: string, dateTo?: string) {
  const locationId = useAppStore((s) => s.activeLocationId)
  return useQuery({
    queryKey: ['stats-revenue', locationId, dateFrom, dateTo],
    queryFn: () =>
      api.get('/statistics/revenue', { params: { location_id: locationId, date_from: dateFrom, date_to: dateTo } }).then((r) => r.data),
  })
}

export function useProfitStats(dateFrom?: string, dateTo?: string) {
  const locationId = useAppStore((s) => s.activeLocationId)
  return useQuery({
    queryKey: ['stats-profit', locationId, dateFrom, dateTo],
    queryFn: () =>
      api.get('/statistics/profit', { params: { location_id: locationId, date_from: dateFrom, date_to: dateTo } }).then((r) => r.data),
  })
}
