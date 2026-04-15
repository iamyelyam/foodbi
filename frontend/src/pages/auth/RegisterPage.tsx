import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import api from '@/lib/api'
import { useT } from '@/i18n'

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
  const t = useT()
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
      setError(err.response?.data?.error || t('auth.registrationFailed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <div className="px-4 pt-16 pb-6">
        <h1 className="text-2xl font-bold text-dark">{t('auth.signUp')}</h1>
        <p className="mt-2 text-sm text-gray">{t('auth.createFoodBIAccount')}</p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col flex-1 px-4 gap-4">
        <Input
          label={t('auth.companyName')}
          placeholder={t('auth.companyNamePh')}
          error={errors.company_name?.message}
          {...register('company_name')}
        />
        <div className="grid grid-cols-2 gap-3">
          <Input
            label={t('common.firstName')}
            placeholder={t('auth.johnPh')}
            error={errors.first_name?.message}
            {...register('first_name')}
          />
          <Input
            label={t('common.lastName')}
            placeholder={t('auth.doePh')}
            error={errors.last_name?.message}
            {...register('last_name')}
          />
        </div>
        <Input
          label={t('common.email')}
          type="email"
          placeholder={t('auth.emailPlaceholder')}
          error={errors.email?.message}
          {...register('email')}
        />
        <Input
          label={t('common.password')}
          type="password"
          placeholder={t('auth.min8Placeholder')}
          error={errors.password?.message}
          {...register('password')}
        />

        {error && <p className="text-sm text-danger text-center">{error}</p>}

        <div className="mt-auto pb-8">
          <Button type="submit" fullWidth disabled={loading}>
            {loading ? t('auth.creatingAccount') : t('auth.createAccount')}
          </Button>
          <button
            type="button"
            onClick={() => navigate('/login')}
            className="mt-4 w-full text-center text-sm text-primary font-medium"
          >
            {t('auth.alreadyHaveAccount')}
          </button>
        </div>
      </form>
    </div>
  )
}
