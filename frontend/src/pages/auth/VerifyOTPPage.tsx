import { useState, useRef, type KeyboardEvent } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import api from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

export function VerifyOTPPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const email = (location.state as { email?: string })?.email || ''
  const { setTokens } = useAuthStore()

  const [code, setCode] = useState(['', '', '', '', '', ''])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const inputs = useRef<(HTMLInputElement | null)[]>([])

  const handleChange = (index: number, value: string) => {
    if (!/^\d*$/.test(value)) return
    const newCode = [...code]
    newCode[index] = value.slice(-1)
    setCode(newCode)

    if (value && index < 5) {
      inputs.current[index + 1]?.focus()
    }
  }

  const handleKeyDown = (index: number, e: KeyboardEvent) => {
    if (e.key === 'Backspace' && !code[index] && index > 0) {
      inputs.current[index - 1]?.focus()
    }
  }

  const handleSubmit = async () => {
    const otpCode = code.join('')
    if (otpCode.length !== 6) return

    setLoading(true)
    setError('')
    try {
      const res = await api.post('/auth/verify-otp', { email, code: otpCode })
      setTokens(res.data.access_token, res.data.refresh_token)
      navigate('/')
    } catch (err: any) {
      setError(err.response?.data?.error || 'Verification failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <div className="px-4 pt-16 pb-8">
        <h1 className="text-2xl font-bold text-dark">Activate your account</h1>
        <p className="mt-2 text-sm text-gray">
          Enter the 6-digit code sent to <span className="font-medium text-dark">{email}</span>
        </p>
      </div>

      <div className="px-4 flex flex-col flex-1">
        <div className="flex gap-3 justify-center">
          {code.map((digit, i) => (
            <input
              key={i}
              ref={(el) => { inputs.current[i] = el }}
              type="text"
              inputMode="numeric"
              maxLength={1}
              value={digit}
              onChange={(e) => handleChange(i, e.target.value)}
              onKeyDown={(e) => handleKeyDown(i, e)}
              className="w-12 h-14 text-center text-xl font-bold rounded-[12px] border border-bg-alt bg-white focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          ))}
        </div>

        {error && <p className="mt-4 text-sm text-danger text-center">{error}</p>}

        <div className="mt-auto pb-8">
          <Button onClick={handleSubmit} fullWidth disabled={loading || code.join('').length !== 6}>
            {loading ? 'Verifying...' : 'Verify'}
          </Button>
          <button className="mt-4 w-full text-center text-sm text-primary font-medium">
            Resend code
          </button>
        </div>
      </div>
    </div>
  )
}
