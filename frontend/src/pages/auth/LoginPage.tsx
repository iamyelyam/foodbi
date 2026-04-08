import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ChevronLeft } from 'lucide-react'
import api from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { useT } from '@/i18n'

const emailSchema = z.object({ email: z.string().email('Invalid email') })
const passwordSchema = z.object({ password: z.string().min(8, 'Minimum 8 characters') })

type Step = 'email' | 'password'

export function LoginPage() {
  const navigate = useNavigate()
  const t = useT()
  const { setTokens } = useAuthStore()
  const [step, setStep] = useState<Step>('email')
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const emailForm = useForm<{ email: string }>({ resolver: zodResolver(emailSchema) })
  const passForm = useForm<{ password: string }>({ resolver: zodResolver(passwordSchema) })

  const handleEmailNext = (data: { email: string }) => {
    setEmail(data.email)
    setStep('password')
    setError('')
  }

  const handleLogin = async (data: { password: string }) => {
    setLoading(true)
    setError('')
    try {
      const res = await api.post('/auth/login', { email, password: data.password })
      setTokens(res.data.access_token, res.data.refresh_token)
      navigate('/')
    } catch (err: any) {
      setError(err.response?.data?.error || 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      {/* Header */}
      {step === 'password' && (
        <button onClick={() => setStep('email')} className="px-4 pt-4">
          <ChevronLeft className="h-6 w-6 text-dark" />
        </button>
      )}

      <div className="px-4 pt-12 pb-6">
        <h1 className="text-2xl font-bold text-dark">
          {step === 'email' ? t('auth.enterEmail') : t('auth.enterPassword')}
        </h1>
        <p className="mt-2 text-sm text-gray">
          {step === 'email' ? t('auth.signIn') : email}
        </p>
      </div>

      {step === 'email' ? (
        <form onSubmit={emailForm.handleSubmit(handleEmailNext)} className="flex flex-col flex-1 px-4 gap-4">
          <Input
            label="Email"
            type="email"
            placeholder="your@email.com"
            autoFocus
            error={emailForm.formState.errors.email?.message}
            {...emailForm.register('email')}
          />
          <div className="mt-auto pb-8">
            <Button type="submit" fullWidth>{t('auth.continue')}</Button>
            <button type="button" onClick={() => navigate('/register')}
              className="mt-4 w-full text-center text-sm text-primary font-medium">
              {t('auth.noAccount')}
            </button>
          </div>
        </form>
      ) : (
        <form onSubmit={passForm.handleSubmit(handleLogin)} className="flex flex-col flex-1 px-4 gap-4">
          <Input
            label="Password"
            type="password"
            placeholder="Enter password"
            autoFocus
            error={passForm.formState.errors.password?.message}
            {...passForm.register('password')}
          />
          {error && <p className="text-sm text-danger text-center">{error}</p>}
          <div className="mt-auto pb-8">
            <Button type="submit" fullWidth disabled={loading}>
              {loading ? t('auth.signingIn') : t('auth.signInBtn')}
            </Button>
            <button type="button" onClick={() => navigate('/forgot-password')}
              className="mt-4 w-full text-center text-sm text-gray">
              {t('auth.forgotPassword')}
            </button>
          </div>
        </form>
      )}
    </div>
  )
}
