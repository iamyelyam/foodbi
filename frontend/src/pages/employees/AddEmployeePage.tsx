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
import { Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'

const schema = z.object({
  first_name: z.string().min(1, 'Required'),
  last_name: z.string().min(1, 'Required'),
  email: z.string().email('Invalid email'),
  phone: z.string().optional(),
  password: z.string().min(8, 'Min 8 characters'),
})

type Form = z.infer<typeof schema>

export function AddEmployeePage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [role, setRole] = useState('employee')
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
      <Header title="Adding an employee" showBack />
      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col flex-1 px-4 pt-4 gap-4">
        <div className="grid grid-cols-2 gap-3">
          <Input label="First name" error={errors.first_name?.message} {...register('first_name')} />
          <Input label="Last name" error={errors.last_name?.message} {...register('last_name')} />
        </div>
        <Input label="Email" type="email" error={errors.email?.message} {...register('email')} />
        <Input label="Phone" {...register('phone')} />
        <Input label="Password" type="password" error={errors.password?.message} {...register('password')} />

        {/* Role selector */}
        <div>
          <label className="text-sm font-medium text-gray">Role</label>
          <button type="button" onClick={() => setShowRoles(true)}
            className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4 text-left text-sm text-dark capitalize bg-white">
            {role}
          </button>
        </div>

        {/* Location selector */}
        <div>
          <label className="text-sm font-medium text-gray">Locations</label>
          <button type="button" onClick={() => setShowLocations(true)}
            className="w-full mt-1 h-12 rounded-[12px] border border-bg-alt px-4 text-left text-sm text-dark bg-white">
            {selectedLocs.length > 0 ? `${selectedLocs.length} selected` : 'Select locations'}
          </button>
        </div>

        {mutation.isError && <p className="text-sm text-danger text-center">Failed to add employee</p>}

        <div className="mt-auto pb-8">
          <Button type="submit" fullWidth disabled={mutation.isPending}>
            {mutation.isPending ? 'Adding...' : 'Add Employee'}
          </Button>
        </div>
      </form>

      {/* Role BottomSheet */}
      <BottomSheet isOpen={showRoles} onClose={() => setShowRoles(false)} title="Choose a role">
        {['owner', 'employee'].map((r) => (
          <button key={r} onClick={() => { setRole(r); setShowRoles(false) }}
            className={cn('w-full flex items-center justify-between p-3 rounded-[12px] mb-2',
              role === r ? 'bg-primary/5' : 'hover:bg-bg-alt')}>
            <span className="text-sm font-medium text-dark capitalize">{r}</span>
            {role === r && <Check className="h-5 w-5 text-primary" />}
          </button>
        ))}
      </BottomSheet>

      {/* Location BottomSheet */}
      <BottomSheet isOpen={showLocations} onClose={() => setShowLocations(false)} title="Select locations">
        <div className="space-y-2">
          {locations.map((loc: any) => (
            <Checkbox key={loc.id} label={loc.name}
              checked={selectedLocs.includes(loc.id)}
              onChange={() => toggleLoc(loc.id)} />
          ))}
        </div>
        <Button fullWidth className="mt-4" onClick={() => setShowLocations(false)}>Done</Button>
      </BottomSheet>

      <Snackbar
        isOpen={showSuccess}
        onClose={() => setShowSuccess(false)}
        message="Employee added successfully"
        type="success"
      />
    </div>
  )
}
