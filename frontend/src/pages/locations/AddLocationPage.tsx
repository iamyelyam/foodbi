import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import api from '@/lib/api'

export function AddLocationPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [address, setAddress] = useState('')
  const [iikoOrgId, setIikoOrgId] = useState('')
  const [nameError, setNameError] = useState('')

  const mutation = useMutation({
    mutationFn: (data: { name: string; address: string; iiko_org_id: string }) =>
      api.post('/locations', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      navigate('/locations')
    },
  })

  const handleSubmit = () => {
    if (!name.trim()) {
      setNameError('Restaurant name is required')
      return
    }
    setNameError('')
    mutation.mutate({ name, address, iiko_org_id: iikoOrgId })
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <Header title="Add Location" showBack />

      <div className="flex-1 px-4 pt-4 space-y-4">
        <Input
          label="Restaurant name"
          placeholder="e.g. Downtown Branch"
          value={name}
          onChange={(e) => { setName(e.target.value); setNameError('') }}
          error={nameError}
        />
        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium text-gray">Address</label>
          <textarea
            placeholder="Full address"
            value={address}
            onChange={(e) => setAddress(e.target.value)}
            rows={3}
            className="w-full rounded-[12px] border border-bg-alt bg-white px-4 py-3 text-base text-dark placeholder:text-gray-light focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary resize-none"
          />
        </div>
        <Input
          label="iiko Organization ID"
          placeholder="From iiko Cloud settings"
          value={iikoOrgId}
          onChange={(e) => setIikoOrgId(e.target.value)}
        />
        <p className="text-xs text-gray">You can find the Organization ID in your iiko Cloud admin panel under Settings → API.</p>

        {mutation.isError && (
          <p className="text-sm text-danger text-center">Failed to add location. Please try again.</p>
        )}
      </div>

      <div className="px-4 pb-8 pt-4">
        <Button fullWidth onClick={handleSubmit} disabled={mutation.isPending}>
          {mutation.isPending ? 'Adding...' : 'Add Location'}
        </Button>
      </div>
    </div>
  )
}
