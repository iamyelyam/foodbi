import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ChevronLeft, Mail } from 'lucide-react'
import api from '@/lib/api'

export function ForgotPasswordPage() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!email) return
    setLoading(true)
    try {
      await api.post('/auth/forgot-password', { email })
    } catch {
      // Always show success to prevent email enumeration
    }
    setSent(true)
    setLoading(false)
  }

  if (sent) {
    return (
      <div className="flex flex-col min-h-dvh bg-white items-center justify-center px-6 text-center">
        <div className="w-20 h-20 rounded-full bg-primary-lighter flex items-center justify-center mb-6">
          <Mail className="h-10 w-10 text-primary" />
        </div>
        <h1 className="text-xl font-bold text-dark">Check your email</h1>
        <p className="mt-2 text-sm text-gray max-w-[280px]">
          If an account exists for {email}, we've sent a password reset link.
        </p>
        <Button className="mt-8" onClick={() => navigate('/login')}>Back to Login</Button>
      </div>
    )
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <button onClick={() => navigate('/login')} className="px-4 pt-4">
        <ChevronLeft className="h-6 w-6 text-dark" />
      </button>

      <div className="px-4 pt-8 pb-6">
        <h1 className="text-2xl font-bold text-dark">Forgot password?</h1>
        <p className="mt-2 text-sm text-gray">Enter your email and we'll send a reset link</p>
      </div>

      <form onSubmit={handleSubmit} className="flex flex-col flex-1 px-4 gap-4">
        <Input label="Email" type="email" placeholder="your@email.com"
          value={email} onChange={(e) => setEmail(e.target.value)} />

        <div className="mt-auto pb-8">
          <Button type="submit" fullWidth disabled={loading || !email}>
            {loading ? 'Sending...' : 'Send reset link'}
          </Button>
        </div>
      </form>
    </div>
  )
}
