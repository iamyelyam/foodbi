import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import api from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

const schema = z.object({
  first_name: z.string().min(1, 'Required'),
  last_name: z.string().min(1, 'Required'),
  password: z.string().min(8, 'Minimum 8 characters'),
})

type Form = z.infer<typeof schema>

export function AcceptInvitePage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') || ''
  const { setTokens } = useAuthStore()
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<Form>({
    resolver: zodResolver(schema),
  })

  const onSubmit = async (data: Form) => {
    if (!token) { setError('Invalid invite link'); return }
    setLoading(true)
    setError('')
    try {
      const res = await api.post('/auth/accept-invite', { token, ...data })
      setTokens(res.data.access_token, res.data.refresh_token)
      navigate('/onboarding')
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to accept invite')
    } finally {
      setLoading(false)
    }
  }

  if (!token) {
    return (
      <div className="flex flex-col min-h-dvh bg-white items-center justify-center px-6 text-center">
        <h1 className="text-xl font-bold text-dark">Invalid Invite</h1>
        <p className="mt-2 text-sm text-gray">This invite link is invalid or has expired.</p>
        <Button className="mt-6" onClick={() => navigate('/login')}>Go to Login</Button>
      </div>
    )
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <div className="px-4 pt-16 pb-6">
        <h1 className="text-2xl font-bold text-dark">Join your team</h1>
        <p className="mt-2 text-sm text-gray">Set up your account to get started</p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col flex-1 px-4 gap-4">
        <div className="grid grid-cols-2 gap-3">
          <Input label="First name" error={errors.first_name?.message} {...register('first_name')} />
          <Input label="Last name" error={errors.last_name?.message} {...register('last_name')} />
        </div>
        <Input label="Password" type="password" placeholder="Minimum 8 characters"
          error={errors.password?.message} {...register('password')} />

        {error && <p className="text-sm text-danger text-center">{error}</p>}

        <div className="mt-auto pb-8">
          <Button type="submit" fullWidth disabled={loading}>
            {loading ? 'Setting up...' : 'Create Account'}
          </Button>
        </div>
      </form>
    </div>
  )
}
