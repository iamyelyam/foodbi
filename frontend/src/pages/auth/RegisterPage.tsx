import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import api from '@/lib/api'

const registerSchema = z.object({
  first_name: z.string().min(1, 'Required'),
  last_name: z.string().min(1, 'Required'),
  email: z.string().email('Invalid email'),
  password: z.string().min(8, 'Minimum 8 characters'),
  company_name: z.string().min(1, 'Required'),
})

type RegisterForm = z.infer<typeof registerSchema>

export function RegisterPage() {
  const navigate = useNavigate()
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<RegisterForm>({
    resolver: zodResolver(registerSchema),
  })

  const onSubmit = async (data: RegisterForm) => {
    setLoading(true)
    setError('')
    try {
      await api.post('/auth/register', { ...data, role: 'owner' })
      navigate('/verify-otp', { state: { email: data.email } })
    } catch (err: any) {
      setError(err.response?.data?.error || 'Registration failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <div className="px-4 pt-16 pb-6">
        <h1 className="text-2xl font-bold text-dark">Sign up</h1>
        <p className="mt-2 text-sm text-gray">Create your FoodBI account</p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col flex-1 px-4 gap-4">
        <Input
          label="Company name"
          placeholder="Your restaurant name"
          error={errors.company_name?.message}
          {...register('company_name')}
        />
        <div className="grid grid-cols-2 gap-3">
          <Input
            label="First name"
            placeholder="John"
            error={errors.first_name?.message}
            {...register('first_name')}
          />
          <Input
            label="Last name"
            placeholder="Doe"
            error={errors.last_name?.message}
            {...register('last_name')}
          />
        </div>
        <Input
          label="Email"
          type="email"
          placeholder="your@email.com"
          error={errors.email?.message}
          {...register('email')}
        />
        <Input
          label="Password"
          type="password"
          placeholder="Minimum 8 characters"
          error={errors.password?.message}
          {...register('password')}
        />

        {error && <p className="text-sm text-danger text-center">{error}</p>}

        <div className="mt-auto pb-8">
          <Button type="submit" fullWidth disabled={loading}>
            {loading ? 'Creating account...' : 'Create account'}
          </Button>
          <button
            type="button"
            onClick={() => navigate('/login')}
            className="mt-4 w-full text-center text-sm text-primary font-medium"
          >
            Already have an account? Sign in
          </button>
        </div>
      </form>
    </div>
  )
}
