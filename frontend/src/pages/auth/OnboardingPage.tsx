import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { ProgressBar } from '@/components/ui/progress-bar'
import { Bell, Fingerprint } from 'lucide-react'
import { useT } from '@/i18n'

export function OnboardingPage() {
  const navigate = useNavigate()
  const t = useT()
  const [step, setStep] = useState(0)

  // Step defs use the current locale via t() — recomputed on each render when locale changes.
  const steps = [
    {
      icon: Bell,
      title: t('auth.enableNotifications'),
      description: t('auth.enableNotificationsDesc'),
      action: t('auth.enableAction'),
    },
    {
      icon: Fingerprint,
      title: t('auth.enableFaceId'),
      description: t('auth.enableFaceIdDesc'),
      action: t('auth.enableAction'),
    },
  ]

  const current = steps[step]
  const progress = ((step + 1) / steps.length) * 100

  const handleAction = () => {
    // In production: request notification/biometric permissions
    advance()
  }

  const advance = () => {
    if (step < steps.length - 1) {
      setStep(step + 1)
    } else {
      navigate('/')
    }
  }

  const Icon = current.icon

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <div className="px-4 pt-4">
        <ProgressBar value={progress} />
      </div>

      <div className="flex flex-col flex-1 items-center justify-center px-6 text-center">
        <div className="w-24 h-24 rounded-full bg-primary-lighter flex items-center justify-center mb-8">
          <Icon className="h-12 w-12 text-primary" />
        </div>
        <h1 className="text-2xl font-bold text-dark">{current.title}</h1>
        <p className="mt-3 text-sm text-gray leading-relaxed max-w-[280px]">{current.description}</p>
      </div>

      <div className="px-4 pb-8 space-y-3">
        <Button fullWidth onClick={handleAction}>{current.action}</Button>
        <Button variant="ghost" fullWidth onClick={advance}>{t('auth.skipAction')}</Button>
      </div>
    </div>
  )
}
