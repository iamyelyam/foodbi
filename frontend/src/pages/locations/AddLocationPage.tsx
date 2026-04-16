import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronDown, Loader2, CheckCircle, XCircle } from 'lucide-react'
import { Header } from '@/components/layout/Header'
import { ProgressBar } from '@/components/ui/progress-bar'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { POS_SYSTEMS, findPosLabel } from '@/lib/posSystems'
import { useT } from '@/i18n'

type Step = 'info' | 'iiko' | 'syncing'

export function AddLocationPage() {
  const t = useT()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const [step, setStep] = useState<Step>('info')

  // Step 1: basic info
  const [name] = useState('')
  const [city] = useState('')
  const [address] = useState('')
  const [posSystem, setPosSystem] = useState<string>('')
  const [showPosSheet, setShowPosSheet] = useState(false)

  // Step 2: iiko credentials
  const [iikoUrl, setIikoUrl] = useState('https://')
  const [iikoLogin, setIikoLogin] = useState('')
  const [iikoPassword, setIikoPassword] = useState('')

  // Step 3: sync progress
  const [locationId, setLocationId] = useState<string>('')
  const [syncError, setSyncError] = useState('')

  const step1Valid = !!posSystem
  const step2Valid = iikoUrl.length > 10 && iikoLogin.trim().length > 0 && iikoPassword.length > 0

  // Create location
  const createMutation = useMutation({
    mutationFn: (data: { name: string; city: string; address: string; pos_system: string }) =>
      api.post('/locations', data),
    onSuccess: (res) => {
      const id = res.data?.id
      setLocationId(id)
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      // Save iiko config then trigger sync
      saveIikoThenSync(id)
    },
  })

  // Save iiko config
  const iikoMutation = useMutation({
    mutationFn: () =>
      api.put('/locations/iiko-config', {
        iiko_server_url: iikoUrl.trim(),
        iiko_login: iikoLogin.trim(),
        iiko_password: iikoPassword,
      }),
  })

  // Trigger sync
  const syncMutation = useMutation({
    mutationFn: (locId: string) => api.post(`/locations/${locId}/sync`),
  })

  const saveIikoThenSync = async (locId: string) => {
    setStep('syncing')
    try {
      await iikoMutation.mutateAsync()
      await syncMutation.mutateAsync(locId)
    } catch (e: any) {
      setSyncError(e.response?.data?.error || 'Failed to configure iiko')
    }
  }

  // Poll sync status while on syncing step
  const { data: syncStatus = [] } = useQuery({
    queryKey: ['sync-status-poll'],
    queryFn: () => api.get('/locations/sync-status').then((r) => r.data),
    refetchInterval: step === 'syncing' ? 3000 : false,
    enabled: step === 'syncing',
  })

  // Derive sync progress from status entries
  const syncTypes = ['revenue', 'product_sales', 'purchases', 'stock']
  const completedTypes = syncStatus.filter(
    (s: any) => s.location_id === locationId && s.status === 'success'
  )
  const failedTypes = syncStatus.filter(
    (s: any) => s.location_id === locationId && s.status === 'failed'
  )
  const progressPercent = Math.round(
    ((completedTypes.length + failedTypes.length) / syncTypes.length) * 100
  )
  const allDone = completedTypes.length + failedTypes.length >= syncTypes.length
  const hasFailures = failedTypes.length > 0

  const handleStep1Next = () => {
    if (posSystem === 'iiko') {
      setStep('iiko')
    } else {
      // Non-iiko POS: create location and go to locations list
      createMutation.mutate({
        name: name.trim(),
        city: city.trim(),
        address: address.trim(),
        pos_system: posSystem,
      })
    }
  }

  const handleStep2Submit = () => {
    createMutation.mutate({
      name: name.trim() || posSystem,
      city: city.trim(),
      address: address.trim(),
      pos_system: posSystem,
    })
  }

  const stepTitles: Record<Step, string> = {
    info: t('locations.addLocation'),
    iiko: 'iiko Server',
    syncing: t('locations.syncing'),
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <Header title={stepTitles[step]} showBack />

      {/* Step 1: Basic info */}
      {step === 'info' && (
        <>
          <main className="flex-1 px-4 pt-4 space-y-3">
            <button
              type="button"
              onClick={() => setShowPosSheet(true)}
              className="w-full bg-bg rounded-[14px] px-4 py-4 flex items-center justify-between text-base text-dark"
            >
              <span className={posSystem ? 'text-dark' : 'text-gray-light'}>
                {posSystem ? findPosLabel(posSystem) : t('locations.posSystemLabel')}
              </span>
              <ChevronDown className="h-5 w-5 text-gray shrink-0" />
            </button>
          </main>

          <div className="px-4 pb-8 pt-4">
            <button
              onClick={handleStep1Next}
              disabled={!step1Valid || createMutation.isPending}
              className={cn(
                'w-full rounded-full py-4 text-base font-semibold transition-colors',
                step1Valid ? 'bg-primary text-dark active:opacity-80' : 'bg-primary-lighter text-gray cursor-not-allowed'
              )}
            >
              {createMutation.isPending ? t('common.adding') : t('common.next')}
            </button>
          </div>

          {/* POS picker sheet */}
          {showPosSheet && (
            <>
              <div className="fixed inset-0 bg-black/40 z-40" onClick={() => setShowPosSheet(false)} />
              <div className="fixed bottom-0 inset-x-0 bg-white rounded-t-[24px] z-50 p-4 pb-8 space-y-2">
                <p className="text-base font-bold text-dark mb-2 text-center">{t('locations.posSystemLabel')}</p>
                {POS_SYSTEMS.map((opt) => (
                  <button
                    key={opt.id}
                    disabled={!opt.enabled}
                    onClick={() => { if (opt.enabled) { setPosSystem(opt.id); setShowPosSheet(false) } }}
                    className={cn(
                      'w-full py-3 rounded-[12px] text-sm font-medium transition-colors flex items-center justify-center gap-2',
                      !opt.enabled ? 'bg-bg text-gray-light cursor-not-allowed'
                        : posSystem === opt.id ? 'bg-primary-lighter text-dark border-2 border-primary'
                        : 'bg-bg text-dark'
                    )}
                  >
                    {opt.label}
                    {!opt.enabled && (
                      <span className="text-[10px] uppercase tracking-wide bg-gray-light/30 text-gray px-2 py-0.5 rounded-full">
                        {t('common.comingSoon')}
                      </span>
                    )}
                  </button>
                ))}
              </div>
            </>
          )}
        </>
      )}

      {/* Step 2: iiko credentials */}
      {step === 'iiko' && (
        <>
          <main className="flex-1 px-4 pt-4 space-y-3">
            <p className="text-sm text-gray mb-2">
              Введите данные вашего iiko Server для синхронизации данных
            </p>
            <FilledInput
              placeholder="iiko Server URL (https://...)"
              value={iikoUrl}
              onChange={setIikoUrl}
            />
            <FilledInput
              placeholder="Login"
              value={iikoLogin}
              onChange={setIikoLogin}
            />
            <FilledInput
              placeholder="Password"
              value={iikoPassword}
              onChange={setIikoPassword}
              type="password"
            />

            {(createMutation.isError || iikoMutation.isError) && (
              <p className="text-sm text-danger text-center pt-2">
                {t('locations.addFailed')}
              </p>
            )}
          </main>

          <div className="px-4 pb-8 pt-4 flex gap-3">
            <button
              onClick={() => setStep('info')}
              className="flex-1 rounded-full py-4 text-base font-semibold bg-bg text-dark"
            >
              {t('common.back')}
            </button>
            <button
              onClick={handleStep2Submit}
              disabled={!step2Valid || createMutation.isPending}
              className={cn(
                'flex-1 rounded-full py-4 text-base font-semibold transition-colors',
                step2Valid ? 'bg-primary text-dark active:opacity-80' : 'bg-primary-lighter text-gray cursor-not-allowed'
              )}
            >
              {createMutation.isPending ? t('common.adding') : t('common.next')}
            </button>
          </div>
        </>
      )}

      {/* Step 3: Sync progress */}
      {step === 'syncing' && (
        <main className="flex-1 px-4 pt-12 flex flex-col items-center">
          {!allDone && !syncError ? (
            <>
              <Loader2 className="h-16 w-16 text-primary animate-spin mb-6" />
              <h2 className="text-xl font-bold text-dark mb-2">Синхронизация с iiko</h2>
              <p className="text-sm text-gray text-center mb-6 max-w-[280px]">
                Загружаем выручку, закупки, склад и продукты из вашего iiko Server
              </p>
              <div className="w-full max-w-[300px]">
                <ProgressBar value={progressPercent} />
                <p className="text-xs text-gray text-center mt-2">{progressPercent}%</p>
              </div>

              {/* Per-type status */}
              <div className="w-full max-w-[300px] mt-6 space-y-2">
                {syncTypes.map((type) => {
                  const done = completedTypes.some((s: any) => s.sync_type === type)
                  const failed = failedTypes.some((s: any) => s.sync_type === type)
                  return (
                    <div key={type} className="flex items-center justify-between">
                      <span className="text-sm text-dark capitalize">{type.replace('_', ' ')}</span>
                      {done ? (
                        <CheckCircle className="h-5 w-5 text-success" />
                      ) : failed ? (
                        <XCircle className="h-5 w-5 text-danger" />
                      ) : (
                        <Loader2 className="h-4 w-4 text-gray animate-spin" />
                      )}
                    </div>
                  )
                })}
              </div>
            </>
          ) : syncError ? (
            <>
              <XCircle className="h-16 w-16 text-danger mb-6" />
              <h2 className="text-xl font-bold text-dark mb-2">Ошибка</h2>
              <p className="text-sm text-danger text-center mb-6">{syncError}</p>
              <button
                onClick={() => { setSyncError(''); setStep('iiko') }}
                className="bg-primary text-dark font-semibold rounded-full px-8 py-3"
              >
                {t('common.retry')}
              </button>
            </>
          ) : (
            <>
              <CheckCircle className="h-16 w-16 text-success mb-6" />
              <h2 className="text-xl font-bold text-dark mb-2">Готово!</h2>
              <p className="text-sm text-gray text-center mb-6">
                {hasFailures
                  ? 'Часть данных загружена. Некоторые этапы завершились с ошибкой — повторите позже.'
                  : 'Все данные синхронизированы из iiko. Можно пользоваться!'}
              </p>
              <button
                onClick={() => navigate('/locations')}
                className="bg-primary text-dark font-semibold rounded-full px-8 py-3"
              >
                {t('common.done')}
              </button>
            </>
          )}
        </main>
      )}
    </div>
  )
}

function FilledInput({
  placeholder,
  value,
  onChange,
  type = 'text',
}: {
  placeholder: string
  value: string
  onChange: (v: string) => void
  type?: string
}) {
  return (
    <input
      placeholder={placeholder}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      type={type}
      className="w-full bg-bg rounded-[14px] px-4 py-4 text-base text-dark placeholder:text-gray-light outline-none focus:ring-1 focus:ring-primary"
    />
  )
}
