import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import { Plus, User, Shield, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { useT } from '@/i18n'
import { EMPLOYEE_ROLES, findRoleLabel } from '@/lib/employeeRoles'

export function EmployeesPage() {
  const t = useT()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState({ first_name: '', last_name: '', email: '', phone: '', password: '', role: EMPLOYEE_ROLES[0].id })

  const { data: employees = [], isLoading } = useQuery({
    queryKey: ['employees'],
    queryFn: () => api.get('/employees').then((r) => r.data),
  })

  const addMutation = useMutation({
    mutationFn: (data: typeof form) => api.post('/employees', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employees'] })
      setShowAdd(false)
      setForm({ first_name: '', last_name: '', email: '', phone: '', password: '', role: EMPLOYEE_ROLES[0].id })
    },
  })

  const update = (field: string, value: string) => setForm((f) => ({ ...f, [field]: value }))

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={t('employees.title')} showBack showNotification badgeCount={unreadCount} />

      <div className="px-4 pt-3 pb-3 flex items-center justify-between">
        <span className="text-xs text-gray">{t('employees.countLabel', { count: employees.length })}</span>
        <button onClick={() => navigate('/employees/new')} className="flex items-center gap-1 text-sm font-medium text-primary">
          <Plus className="h-4 w-4" /> {t('common.add')}
        </button>
      </div>

      <main className="flex-1 px-4 pb-20 space-y-2">
        {isLoading ? (
          <>
            <ListItemSkeleton />
            <ListItemSkeleton />
            <ListItemSkeleton />
          </>
        ) : (
        <>
        {employees.map((emp: any) => (
          <button key={emp.id} className="w-full text-left bg-white rounded-[12px] p-4 shadow-sm" onClick={() => navigate(`/employees/${emp.id}`)}>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={cn(
                  'w-10 h-10 rounded-full flex items-center justify-center',
                  emp.role === 'owner' ? 'bg-warning/10' : 'bg-primary-lighter'
                )}>
                  {emp.role === 'owner' ? <Shield className="h-5 w-5 text-warning" /> : <User className="h-5 w-5 text-primary" />}
                </div>
                <div>
                  <p className="text-sm font-semibold text-dark">{emp.first_name} {emp.last_name}</p>
                  <p className="text-xs text-gray mt-0.5">{emp.email}</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className={cn(
                  'text-xs px-2 py-0.5 rounded-full font-medium',
                  emp.role === 'owner' ? 'bg-warning/10 text-warning' : 'bg-primary-lighter text-primary'
                )}>
                  {findRoleLabel(emp.role)}
                </span>
                <ChevronRight className="h-4 w-4 text-gray-light" />
              </div>
            </div>
          </button>
        ))}

        {employees.length === 0 && (
          <div className="text-center py-12">
            <User className="h-12 w-12 text-gray-light mx-auto mb-3" />
            <p className="text-sm text-gray">{t('employees.noEmployeesYet')}</p>
          </div>
        )}
        </>
        )}
      </main>

      <Tabbar />

      <BottomSheet isOpen={showAdd} onClose={() => setShowAdd(false)} title={t('employees.addEmployee')}>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <Input label={t('common.firstName')} value={form.first_name} onChange={(e) => update('first_name', e.target.value)} />
            <Input label={t('common.lastName')} value={form.last_name} onChange={(e) => update('last_name', e.target.value)} />
          </div>
          <Input label={t('common.email')} type="email" value={form.email} onChange={(e) => update('email', e.target.value)} />
          <Input label={t('common.phone')} value={form.phone} onChange={(e) => update('phone', e.target.value)} />
          <Input label={t('common.password')} type="password" value={form.password} onChange={(e) => update('password', e.target.value)} />
          <div>
            <label className="text-sm font-medium text-gray">{t('employees.role')}</label>
            <select className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4 bg-white"
              value={form.role} onChange={(e) => update('role', e.target.value)}>
              {EMPLOYEE_ROLES.map((r) => (
                <option key={r.id} value={r.id}>{r.label}</option>
              ))}
            </select>
          </div>
          <Button fullWidth onClick={() => addMutation.mutate(form)} disabled={addMutation.isPending || !form.email || !form.first_name}>
            {addMutation.isPending ? t('common.adding') : t('employees.addEmployee')}
          </Button>
        </div>
      </BottomSheet>
    </div>
  )
}
