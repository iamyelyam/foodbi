import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CardSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Snackbar } from '@/components/ui/snackbar'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import { useAuthStore } from '@/stores/auth'
import { Plus, Clock, CheckCircle, XCircle, Truck } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { useCurrency } from '@/stores/app'

type Tab = 'requests' | 'history'

const statusIcon = {
  pending: Clock,
  approved: CheckCircle,
  rejected: XCircle,
}
const statusColor = {
  pending: 'text-warning bg-warning/10',
  approved: 'text-success bg-success/10',
  rejected: 'text-danger bg-danger/10',
}

export function SupplyingPage() {
  const queryClient = useQueryClient()
  const cs = useCurrency()
  const { user } = useAuthStore()
  const isOwner = user?.role === 'owner'
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const [tab, setTab] = useState<Tab>('requests')
  const [showCreate, setShowCreate] = useState(false)
  const [supplierName, setSupplierName] = useState('')
  const [items, setItems] = useState([{ product_name: '', quantity: '', unit: 'kg', price_per_unit: '', category: '' }])
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; type: 'success' | 'error' }>({ open: false, message: '', type: 'success' })

  const { data: requests = [], isLoading } = useQuery({
    queryKey: ['supply-requests'],
    queryFn: () => api.get('/supplying').then((r) => r.data),
  })

  const pendingRequests = requests.filter((r: any) => r.status === 'pending')
  const historyRequests = requests.filter((r: any) => r.status !== 'pending')

  const createMutation = useMutation({
    mutationFn: (data: any) => api.post('/supplying', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['supply-requests'] })
      setShowCreate(false)
      setSupplierName('')
      setItems([{ product_name: '', quantity: '', unit: 'kg', price_per_unit: '', category: '' }])
      setSnackbar({ open: true, message: 'Supply request created', type: 'success' })
    },
  })

  const approveMutation = useMutation({
    mutationFn: (id: string) => api.post(`/supplying/${id}/approve`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['supply-requests'] })
      setSnackbar({ open: true, message: 'Request approved', type: 'success' })
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Failed to approve', type: 'error' })
    },
  })

  const rejectMutation = useMutation({
    mutationFn: (id: string) => api.post(`/supplying/${id}/reject`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['supply-requests'] })
      setSnackbar({ open: true, message: 'Request rejected', type: 'success' })
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Failed to reject', type: 'error' })
    },
  })

  const addItem = () => setItems([...items, { product_name: '', quantity: '', unit: 'kg', price_per_unit: '', category: '' }])

  const updateItem = (idx: number, field: string, value: string) => {
    const updated = [...items]
    updated[idx] = { ...updated[idx], [field]: value }
    setItems(updated)
  }

  const handleCreate = () => {
    const parsed = items
      .filter((i) => i.product_name && i.quantity)
      .map((i) => ({ ...i, quantity: Number(i.quantity), price_per_unit: Number(i.price_per_unit) || 0 }))
    if (!supplierName || parsed.length === 0) return
    createMutation.mutate({ supplier_name: supplierName, location_id: '', items: parsed })
  }

  const displayedRequests = tab === 'requests' ? pendingRequests : historyRequests

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="Supplying" showBack showNotification badgeCount={unreadCount} />

      {/* Segmented Control */}
      <div className="px-4 pt-2 pb-3">
        <div className="flex bg-bg-alt rounded-[12px] p-1">
          {(['requests', 'history'] as Tab[]).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={cn(
                'flex-1 py-2 text-sm font-medium rounded-[10px] transition-colors capitalize',
                tab === t ? 'bg-white text-dark shadow-sm' : 'text-gray'
              )}
            >
              {t}
            </button>
          ))}
        </div>
      </div>

      <div className="px-4 pb-3 flex items-center justify-between">
        <span className="text-xs text-gray">{displayedRequests.length} {tab === 'requests' ? 'pending' : 'completed'}</span>
        {tab === 'requests' && (
          <button onClick={() => setShowCreate(true)} className="flex items-center gap-1 text-sm font-medium text-primary">
            <Plus className="h-4 w-4" /> New Request
          </button>
        )}
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
        {displayedRequests.map((req: any) => {
          const Icon = statusIcon[req.status as keyof typeof statusIcon] || Clock
          const color = statusColor[req.status as keyof typeof statusColor] || statusColor.pending
          return (
            <div key={req.id} className="bg-white rounded-[12px] p-4 shadow-sm">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-semibold text-dark">{req.supplier_name}</p>
                  <p className="text-xs text-gray mt-0.5">{new Date(req.created_at).toLocaleDateString()}</p>
                </div>
                <div className="flex items-center gap-2">
                  <p className="text-sm font-semibold text-dark">{req.total_sum.toLocaleString('ru-KZ', { maximumFractionDigits: 0 })}{cs}</p>
                  <span className={cn('flex items-center gap-1 text-xs px-2 py-0.5 rounded-full font-medium', color)}>
                    <Icon className="h-3 w-3" /> {req.status}
                  </span>
                </div>
              </div>
              {tab === 'requests' && isOwner && (
                <div className="flex gap-2 mt-3">
                  <Button
                    variant="secondary"
                    fullWidth
                    onClick={() => rejectMutation.mutate(req.id)}
                    disabled={rejectMutation.isPending}
                  >
                    <XCircle className="h-4 w-4 mr-1" /> Reject
                  </Button>
                  <Button
                    fullWidth
                    onClick={() => approveMutation.mutate(req.id)}
                    disabled={approveMutation.isPending}
                  >
                    <CheckCircle className="h-4 w-4 mr-1" /> Approve
                  </Button>
                </div>
              )}
            </div>
          )
        })}

        {displayedRequests.length === 0 && (
          <div className="text-center py-12">
            <Truck className="h-12 w-12 text-gray-light mx-auto mb-3" />
            <p className="text-sm text-gray">
              {tab === 'requests' ? 'No pending requests' : 'No history yet'}
            </p>
          </div>
        )}
        </>
        )}
      </main>

      <Tabbar />

      <BottomSheet isOpen={showCreate} onClose={() => setShowCreate(false)} title="New Supply Request">
        <div className="space-y-4">
          <Input label="Supplier" placeholder="Supplier name" value={supplierName} onChange={(e) => setSupplierName(e.target.value)} />

          {items.map((item, idx) => (
            <div key={idx} className="bg-bg rounded-[12px] p-3 space-y-2">
              <p className="text-xs font-medium text-gray">Item {idx + 1}</p>
              <Input placeholder="Product name" value={item.product_name} onChange={(e) => updateItem(idx, 'product_name', e.target.value)} />
              <div className="grid grid-cols-3 gap-2">
                <Input placeholder="Qty" type="number" value={item.quantity} onChange={(e) => updateItem(idx, 'quantity', e.target.value)} />
                <Input placeholder="Unit" value={item.unit} onChange={(e) => updateItem(idx, 'unit', e.target.value)} />
                <Input placeholder="Price" type="number" value={item.price_per_unit} onChange={(e) => updateItem(idx, 'price_per_unit', e.target.value)} />
              </div>
            </div>
          ))}

          <button onClick={addItem} className="w-full text-sm text-primary font-medium py-2">+ Add item</button>

          <Button fullWidth onClick={handleCreate} disabled={createMutation.isPending}>
            {createMutation.isPending ? 'Creating...' : 'Create Request'}
          </Button>
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
