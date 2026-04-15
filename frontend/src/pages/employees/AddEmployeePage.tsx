import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Checkbox } from '@/components/ui/checkbox'
import { Snackbar } from '@/components/ui/snackbar'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { EMPLOYEE_ROLES, findRoleLabel } from '@/lib/employeeRoles'
import { useT } from '@/i18n'

const schema = z.object({
  first_name: z.string().min(1, 'Required'),
  last_name: z.string().min(1, 'Required'),
  email: z.string().email('Invalid email'),
  phone: z.string().optional(),
  password: z.string().min(8, 'Min 8 characters'),
})

type Form = z.infer<typeof schema>

export function AddEmployeePage() {
  const t = useT()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [role, setRole] = useState<string>(EMPLOYEE_ROLES[0].id)
  const [showRoles, setShowRoles] = useState(false)
  const [showLocations, setShowLocations] = useState(false)
  const [selectedLocs, setSelectedLocs] = useState<string[]>([])
  const [showSuccess, setShowSuccess] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<Form>({ resolver: zodResolver(schema) })

  const { data: locations = [] } = useQuery({
    queryKey: ['locations'],
    queryFn: () => api.get('/locations').then((r) => r.data),
  })

  const mutation = useMutation({
    mutationFn: async (data: Form & { role: string }) => {
      const res = await api.post('/employees', data)
      if (selectedLocs.length > 0) {
        await api.put(`/employees/${res.data.id}/locations`, { location_ids: selectedLocs })
      }
      return res
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employees'] })
      setShowSuccess(true)
      setTimeout(() => navigate('/employees'), 1500)
    },
  })

  const toggleLoc = (id: string) =>
    setSelectedLocs((prev) => prev.includes(id) ? prev.filter((l) => l !== id) : [...prev, id])

  const onSubmit = (data: Form) => mutation.mutate({ ...data, phone: data.phone || '', role })

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <Header title={t('employees.addingAnEmployee')} showBack />
      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col flex-1 px-4 pt-4 gap-4">
        <div className="grid grid-cols-2 gap-3">
          <Input label={t('common.firstName')} error={errors.first_name?.message} {...register('first_name')} />
          <Input label={t('common.lastName')} error={errors.last_name?.message} {...register('last_name')} />
        </div>
        <Input label={t('common.email')} type="email" error={errors.email?.message} {...register('email')} />
        <Input label={t('common.phone')} {...register('phone')} />
        <Input label={t('common.password')} type="password" error={errors.password?.message} {...register('password')} />

        {/* Role selector */}
        <div>
          <label className="text-sm font-medium text-gray">{t('employees.role')}</label>
          <button type="button" onClick={() => setShowRoles(true)}
            className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4 text-left text-sm text-dark bg-white">
            {findRoleLabel(role)}
          </button>
        </div>

        {/* Location selector */}
        <div>
          <label className="text-sm font-medium text-gray">{t('employees.locations')}</label>
          <button type="button" onClick={() => setShowLocations(true)}
            className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4 text-left text-sm text-dark bg-white">
            {selectedLocs.length > 0
              ? t('employees.nSelected', { count: selectedLocs.length })
              : t('employees.selectLocations')}
          </button>
        </div>

        {mutation.isError && <p className="text-sm text-danger text-center">{t('employees.addFailed')}</p>}

        <div className="mt-auto pb-8">
          <Button type="submit" fullWidth disabled={mutation.isPending}>
            {mutation.isPending ? t('common.adding') : t('employees.addEmployee')}
          </Button>
        </div>
      </form>

      {/* Role BottomSheet — radio-style picker, single selection */}
      <BottomSheet isOpen={showRoles} onClose={() => setShowRoles(false)} title={t('employees.chooseRole')}>
        <div className="divide-y divide-bg-alt">
          {EMPLOYEE_ROLES.map((r) => {
            const active = role === r.id
            return (
              <button
                key={r.id}
                onClick={() => { setRole(r.id); setShowRoles(false) }}
                className="w-full flex items-center justify-between py-4 text-left"
              >
                <span className="text-base text-dark">{r.label}</span>
                <span
                  className={cn(
                    'w-6 h-6 rounded-full border-2 flex items-center justify-center',
                    active ? 'border-primary bg-primary' : 'border-gray-light bg-white'
                  )}
                >
                  {active && <span className="w-2 h-2 rounded-full bg-white" />}
                </span>
              </button>
            )
          })}
        </div>
      </BottomSheet>

      {/* Location BottomSheet */}
      <BottomSheet isOpen={showLocations} onClose={() => setShowLocations(false)} title={t('employees.selectLocations')}>
        <div className="space-y-2">
          {locations.map((loc: any) => (
            <Checkbox key={loc.id} label={loc.name}
              checked={selectedLocs.includes(loc.id)}
              onChange={() => toggleLoc(loc.id)} />
          ))}
        </div>
        <Button fullWidth className="mt-4" onClick={() => setShowLocations(false)}>{t('common.done')}</Button>
      </BottomSheet>

      <Snackbar
        isOpen={showSuccess}
        onClose={() => setShowSuccess(false)}
        message={t('employees.addedSuccess')}
        type="success"
      />
    </div>
  )
}
