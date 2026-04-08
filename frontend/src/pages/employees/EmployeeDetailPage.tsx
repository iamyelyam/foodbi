import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { Button } from '@/components/ui/button'
import { Modal } from '@/components/ui/modal'
import { Snackbar } from '@/components/ui/snackbar'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { useAuthStore } from '@/stores/auth'
import { User, Mail, Phone, Shield, MapPin } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'

export function EmployeeDetailPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const isOwner = user?.role === 'owner'
  const [showDeactivate, setShowDeactivate] = useState(false)
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; type: 'success' | 'error' }>({ open: false, message: '', type: 'success' })

  const { data: emp, isLoading } = useQuery({
    queryKey: ['employee', id],
    queryFn: () => api.get(`/employees/${id}`).then((r) => r.data),
    enabled: !!id,
  })

  const deactivateMutation = useMutation({
    mutationFn: () => api.post(`/employees/${id}/deactivate`),
    onSuccess: () => {
      setShowDeactivate(false)
      setSnackbar({ open: true, message: 'Employee deactivated', type: 'success' })
      setTimeout(() => navigate('/employees'), 1500)
    },
    onError: () => {
      setShowDeactivate(false)
      setSnackbar({ open: true, message: 'Failed to deactivate', type: 'error' })
    },
  })

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={emp ? `${emp.first_name} ${emp.last_name}` : 'Employee'} showBack />
      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {isLoading ? <><ListItemSkeleton /><ListItemSkeleton /></> : emp ? (
          <>
            <div className="bg-white rounded-[16px] p-6 shadow-sm flex flex-col items-center">
              <div className={cn('w-20 h-20 rounded-full flex items-center justify-center',
                emp.role === 'owner' ? 'bg-warning/10' : 'bg-primary-lighter')}>
                <User className={cn('h-10 w-10', emp.role === 'owner' ? 'text-warning' : 'text-primary')} />
              </div>
              <p className="mt-3 text-lg font-bold text-dark">{emp.first_name} {emp.last_name}</p>
              <span className={cn('text-xs px-3 py-1 rounded-full font-medium capitalize mt-1',
                emp.role === 'owner' ? 'bg-warning/10 text-warning' : 'bg-primary-lighter text-primary')}>
                {emp.role}
              </span>
            </div>

            <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-4">
              <div className="flex items-center gap-3">
                <Mail className="h-5 w-5 text-gray" />
                <div><p className="text-xs text-gray">Email</p><p className="text-sm text-dark">{emp.email}</p></div>
              </div>
              <div className="flex items-center gap-3">
                <Phone className="h-5 w-5 text-gray" />
                <div><p className="text-xs text-gray">Phone</p><p className="text-sm text-dark">{emp.phone || 'Not set'}</p></div>
              </div>
              <div className="flex items-center gap-3">
                <Shield className="h-5 w-5 text-gray" />
                <div><p className="text-xs text-gray">Status</p><p className="text-sm text-dark">{emp.is_active ? 'Active' : 'Inactive'}</p></div>
              </div>
            </div>

            {emp.locations && emp.locations.length > 0 && (
              <div className="bg-white rounded-[16px] p-4 shadow-sm">
                <p className="text-sm font-semibold text-dark mb-3">Assigned Locations</p>
                <div className="space-y-2">
                  {emp.locations.map((loc: string, i: number) => (
                    <div key={i} className="flex items-center gap-2">
                      <MapPin className="h-4 w-4 text-primary" />
                      <span className="text-sm text-dark">{loc}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {isOwner && emp.is_active && emp.role !== 'owner' && (
              <Button variant="danger" fullWidth onClick={() => setShowDeactivate(true)}>
                Deactivate Employee
              </Button>
            )}
          </>
        ) : (
          <p className="text-center text-sm text-gray py-12">Employee not found</p>
        )}
      </main>
      <Tabbar />

      <Modal isOpen={showDeactivate} onClose={() => setShowDeactivate(false)} title="Deactivate Employee">
        <p className="text-sm text-gray mb-4">
          Are you sure you want to deactivate {emp?.first_name} {emp?.last_name}?
        </p>
        <div className="flex gap-3">
          <Button variant="secondary" fullWidth onClick={() => setShowDeactivate(false)}>Cancel</Button>
          <Button variant="danger" fullWidth onClick={() => deactivateMutation.mutate()} disabled={deactivateMutation.isPending}>
            {deactivateMutation.isPending ? 'Deactivating...' : 'Deactivate'}
          </Button>
        </div>
      </Modal>

      <Snackbar
        message={snackbar.message}
        type={snackbar.type}
        isOpen={snackbar.open}
        onClose={() => setSnackbar((s) => ({ ...s, open: false }))}
      />
    </div>
  )
}
