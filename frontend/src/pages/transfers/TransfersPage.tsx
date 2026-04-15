import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CardSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import { useAuthStore } from '@/stores/auth'
import { Snackbar } from '@/components/ui/snackbar'
import { Plus, ArrowRightLeft, Filter, CheckCircle, XCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { useT, useI18nStore } from '@/i18n'

export function TransfersPage() {
  const queryClient = useQueryClient()
  const t = useT()
  const locale = useI18nStore((s) => s.locale)
  const { user } = useAuthStore()
  const isOwner = user?.role === 'owner'
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const [showCreate, setShowCreate] = useState(false)
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; type: 'success' | 'error' }>({ open: false, message: '', type: 'success' })
  const [showFilters, setShowFilters] = useState(false)
  const [filters, setFilters] = useState<Record<string, string>>({})
  const [fromLoc, setFromLoc] = useState('')
  const [toLoc, setToLoc] = useState('')
  const [items, setItems] = useState([{ product_name: '', quantity: '', unit: 'kg', category: '' }])

  const { data: transfers = [], isLoading } = useQuery({
    queryKey: ['transfers', filters],
    queryFn: () => api.get('/transfers', { params: filters }).then((r) => r.data),
  })

  const { data: locations = [] } = useQuery({
    queryKey: ['locations'],
    queryFn: () => api.get('/locations').then((r) => r.data),
  })

  const createMutation = useMutation({
    mutationFn: (data: any) => api.post('/transfers', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['transfers'] })
      setShowCreate(false)
      setFromLoc('')
      setToLoc('')
      setItems([{ product_name: '', quantity: '', unit: 'kg', category: '' }])
    },
  })

  const completeMutation = useMutation({
    mutationFn: (id: string) => api.post(`/transfers/${id}/complete`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['transfers'] })
      setSnackbar({ open: true, message: t('transfers.completedSuccess'), type: 'success' })
    },
    onError: () => {
      setSnackbar({ open: true, message: t('transfers.completeFailed'), type: 'error' })
    },
  })

  const cancelMutation = useMutation({
    mutationFn: (id: string) => api.post(`/transfers/${id}/cancel`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['transfers'] })
      setSnackbar({ open: true, message: t('transfers.cancelledSuccess'), type: 'success' })
    },
    onError: () => {
      setSnackbar({ open: true, message: t('transfers.cancelFailed'), type: 'error' })
    },
  })

  const addItem = () => setItems([...items, { product_name: '', quantity: '', unit: 'kg', category: '' }])

  const updateItem = (idx: number, field: string, value: string) => {
    const updated = [...items]
    updated[idx] = { ...updated[idx], [field]: value }
    setItems(updated)
  }

  const handleCreate = () => {
    const parsed = items
      .filter((i) => i.product_name && i.quantity)
      .map((i) => ({ ...i, quantity: Number(i.quantity) }))
    if (!fromLoc || !toLoc || parsed.length === 0) return
    createMutation.mutate({ from_location_id: fromLoc, to_location_id: toLoc, items: parsed })
  }

  const getLocName = (id: string) => locations.find((l: any) => l.id === id)?.name || id.slice(0, 8)

  const statusLabel = (s: string) =>
    s === 'completed' ? t('transfers.statusCompleted')
      : s === 'cancelled' ? t('transfers.statusCancelled')
      : t('transfers.statusPending')

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={t('transfers.pageTitle')} showBack showNotification badgeCount={unreadCount} />

      <div className="px-4 pt-3 pb-3 flex items-center justify-between">
        <span className="text-xs text-gray">{t('transfers.countLabel', { count: transfers.length })}</span>
        <div className="flex items-center gap-3">
          <button onClick={() => setShowFilters(true)} className="flex items-center gap-1 text-xs font-medium text-primary">
            <Filter className="h-3.5 w-3.5" /> {t('common.filter')}
          </button>
          <button onClick={() => setShowCreate(true)} className="flex items-center gap-1 text-sm font-medium text-primary">
            <Plus className="h-4 w-4" /> {t('transfers.newBtn')}
          </button>
        </div>
      </div>

      <main className="flex-1 px-4 pb-20 space-y-2">
        {isLoading ? (
          <>
            <CardSkeleton />
            <CardSkeleton />
            <CardSkeleton />
          </>
        ) : (
        <>
        {transfers.map((tr: any) => (
          <div key={tr.id} className="bg-white rounded-[12px] p-4 shadow-sm">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-info-light flex items-center justify-center">
                  <ArrowRightLeft className="h-5 w-5 text-info" />
                </div>
                <div>
                  <p className="text-sm font-semibold text-dark">
                    {getLocName(tr.from_location_id)} → {getLocName(tr.to_location_id)}
                  </p>
                  <p className="text-xs text-gray mt-0.5">{new Date(tr.created_at).toLocaleDateString(locale)}</p>
                </div>
              </div>
              <span className={cn(
                'text-xs px-2 py-0.5 rounded-full font-medium',
                tr.status === 'completed' ? 'bg-success/10 text-success' :
                tr.status === 'cancelled' ? 'bg-danger/10 text-danger' : 'bg-warning/10 text-warning'
              )}>
                {statusLabel(tr.status)}
              </span>
            </div>
            {tr.status === 'pending' && isOwner && (
              <div className="flex gap-2 mt-3">
                <Button
                  variant="secondary"
                  fullWidth
                  onClick={() => cancelMutation.mutate(tr.id)}
                  disabled={cancelMutation.isPending}
                >
                  <XCircle className="h-4 w-4 mr-1" /> {t('transfers.cancelBtn')}
                </Button>
                <Button
                  fullWidth
                  onClick={() => completeMutation.mutate(tr.id)}
                  disabled={completeMutation.isPending}
                >
                  <CheckCircle className="h-4 w-4 mr-1" /> {t('transfers.completeBtn')}
                </Button>
              </div>
            )}
          </div>
        ))}

        {transfers.length === 0 && (
          <div className="text-center py-12">
            <ArrowRightLeft className="h-12 w-12 text-gray-light mx-auto mb-3" />
            <p className="text-sm text-gray">{t('transfers.noTransfersYet')}</p>
          </div>
        )}
        </>
        )}
      </main>

      <Tabbar />

      {/* Create transfer */}
      <BottomSheet isOpen={showCreate} onClose={() => setShowCreate(false)} title={t('transfers.newTransferTitle')}>
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium text-gray">{t('transfers.fromLocation')}</label>
            <select className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4 bg-white"
              value={fromLoc} onChange={(e) => setFromLoc(e.target.value)}>
              <option value="">{t('transfers.selectSource')}</option>
              {locations.map((l: any) => <option key={l.id} value={l.id}>{l.name}</option>)}
            </select>
          </div>
          <div>
            <label className="text-sm font-medium text-gray">{t('transfers.toLocation')}</label>
            <select className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4 bg-white"
              value={toLoc} onChange={(e) => setToLoc(e.target.value)}>
              <option value="">{t('transfers.selectDestination')}</option>
              {locations.filter((l: any) => l.id !== fromLoc).map((l: any) => <option key={l.id} value={l.id}>{l.name}</option>)}
            </select>
          </div>

          {items.map((item, idx) => (
            <div key={idx} className="bg-bg rounded-[12px] p-3 space-y-2">
              <p className="text-xs font-medium text-gray">{t('transfers.itemNum', { num: idx + 1 })}</p>
              <Input placeholder={t('transfers.productNamePh')} value={item.product_name} onChange={(e) => updateItem(idx, 'product_name', e.target.value)} />
              <div className="grid grid-cols-2 gap-2">
                <Input placeholder={t('transfers.qtyPh')} type="number" value={item.quantity} onChange={(e) => updateItem(idx, 'quantity', e.target.value)} />
                <Input placeholder={t('transfers.unitPh')} value={item.unit} onChange={(e) => updateItem(idx, 'unit', e.target.value)} />
              </div>
            </div>
          ))}

          <button onClick={addItem} className="w-full text-sm text-primary font-medium py-2">{t('transfers.addItem')}</button>

          <Button fullWidth onClick={handleCreate} disabled={createMutation.isPending}>
            {createMutation.isPending ? t('common.creating') : t('transfers.createBtn')}
          </Button>
        </div>
      </BottomSheet>

      {/* Filters */}
      <BottomSheet isOpen={showFilters} onClose={() => setShowFilters(false)} title={t('common.filter')}>
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium text-gray">{t('statistics.dateFromLabel')}</label>
            <input type="date" className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4"
              value={filters.date_from || ''} onChange={(e) => setFilters((f) => ({ ...f, date_from: e.target.value }))} />
          </div>
          <div>
            <label className="text-sm font-medium text-gray">{t('statistics.dateToLabel')}</label>
            <input type="date" className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4"
              value={filters.date_to || ''} onChange={(e) => setFilters((f) => ({ ...f, date_to: e.target.value }))} />
          </div>
          <div className="flex gap-3">
            <Button variant="secondary" fullWidth onClick={() => { setFilters({}); setShowFilters(false) }}>{t('common.clear')}</Button>
            <Button fullWidth onClick={() => setShowFilters(false)}>{t('common.apply')}</Button>
          </div>
        </div>
      </BottomSheet>

      <Snackbar
        message={snackbar.message}
        type={snackbar.type}
        isOpen={snackbar.open}
        onClose={() => setSnackbar((s) => ({ ...s, open: false }))}
      />
    </div>
  )
}
