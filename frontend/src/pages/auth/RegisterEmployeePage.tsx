import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ProgressBar } from '@/components/ui/progress-bar'
import api from '@/lib/api'

const schema = z.object({
  first_name: z.string().min(1, 'Required'),
  last_name: z.string().min(1, 'Required'),
  email: z.string().email('Invalid email'),
  password: z.string().min(8, 'Minimum 8 characters'),
  invite_code: z.string().min(1, 'Invite code required'),
})

type Form = z.infer<typeof schema>

export function RegisterEmployeePage() {
  const navigate = useNavigate()
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<Form>({
    resolver: zodResolver(schema),
  })

  const onSubmit = async (data: Form) => {
    setLoading(true)
    setError('')
    try {
      await api.post('/auth/accept-invite', {
        token: data.invite_code,
        first_name: data.first_name,
        last_name: data.last_name,
        password: data.password,
      })
      navigate('/onboarding')
    } catch (err: any) {
      setError(err.response?.data?.error || 'Registration failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <div className="px-4 pt-4">
        <ProgressBar value={33} />
      </div>

      <div className="px-4 pt-8 pb-6">
        <h1 className="text-2xl font-bold text-dark">Sign up as Employee</h1>
        <p className="mt-2 text-sm text-gray">Enter your invite code and create your account</p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col flex-1 px-4 gap-4">
        <Input label="Invite code" placeholder="Paste code from your manager" error={errors.invite_code?.message} {...register('invite_code')} />
        <div className="grid grid-cols-2 gap-3">
          <Input label="First name" placeholder="John" error={errors.first_name?.message} {...register('first_name')} />
          <Input label="Last name" placeholder="Doe" error={errors.last_name?.message} {...register('last_name')} />
        </div>
        <Input label="Email" type="email" placeholder="your@email.com" error={errors.email?.message} {...register('email')} />
        <Input label="Password" type="password" placeholder="Minimum 8 characters" error={errors.password?.message} {...register('password')} />

        {error && <p className="text-sm text-danger text-center">{error}</p>}

        <div className="mt-auto pb-8">
          <Button type="submit" fullWidth disabled={loading}>
            {loading ? 'Creating account...' : 'Create account'}
          </Button>
          <button type="button" onClick={() => navigate('/login')}
            className="mt-4 w-full text-center text-sm text-primary font-medium">
            Already have an account? Sign in
          </button>
        </div>
      </form>
    </div>
  )
}
