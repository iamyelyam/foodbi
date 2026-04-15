import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ChevronDown } from 'lucide-react'
import { Header } from '@/components/layout/Header'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { POS_SYSTEMS, findPosLabel } from '@/lib/posSystems'
import { useT } from '@/i18n'

export function AddLocationPage() {
  const t = useT()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const [name, setName] = useState('')
  const [city, setCity] = useState('')
  const [address, setAddress] = useState('')
  const [posSystem, setPosSystem] = useState<string>('')
  const [showPosSheet, setShowPosSheet] = useState(false)

  const isValid = name.trim().length > 0 && city.trim().length > 0 && address.trim().length > 0 && !!posSystem

  const mutation = useMutation({
    mutationFn: (data: { name: string; city: string; address: string; pos_system: string }) =>
      api.post('/locations', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      navigate('/locations')
    },
  })

  const handleSubmit = () => {
    if (!isValid) return
    mutation.mutate({
      name: name.trim(),
      city: city.trim(),
      address: address.trim(),
      pos_system: posSystem as string,
    })
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <Header title={t('locations.addLocation')} showBack showNotification badgeCount={unreadCount} />

      <main className="flex-1 px-4 pt-4 space-y-3">
        <FilledInput placeholder={t('locations.restaurantNamePh')} value={name} onChange={setName} />
        <FilledInput placeholder={t('locations.cityPlaceholder')} value={city} onChange={setCity} />
        <FilledInput placeholder={t('locations.addressLabel')} value={address} onChange={setAddress} />
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

        {mutation.isError && (
          <p className="text-sm text-danger text-center pt-2">{t('locations.addFailed')}</p>
        )}
      </main>

      <div className="px-4 pb-8 pt-4">
        <button
          onClick={handleSubmit}
          disabled={!isValid || mutation.isPending}
          className={cn(
            'w-full rounded-full py-4 text-base font-semibold transition-colors',
            isValid && !mutation.isPending
              ? 'bg-primary text-dark active:opacity-80'
              : 'bg-primary-lighter text-gray cursor-not-allowed'
          )}
        >
          {mutation.isPending ? t('common.adding') : t('locations.addLocation')}
        </button>
      </div>

      {/* POS System bottom sheet — chosen on tap, no extra confirm */}
      {showPosSheet && (
        <>
          <div
            className="fixed inset-0 bg-black/40 z-40"
            onClick={() => setShowPosSheet(false)}
          />
          <div className="fixed bottom-0 inset-x-0 bg-white rounded-t-[24px] z-50 p-4 pb-8 space-y-2">
            <p className="text-base font-bold text-dark mb-2 text-center">{t('locations.posSystemLabel')}</p>
            {POS_SYSTEMS.map((opt) => (
              <button
                key={opt.id}
                disabled={!opt.enabled}
                onClick={() => {
                  if (!opt.enabled) return
                  setPosSystem(opt.id)
                  setShowPosSheet(false)
                }}
                className={cn(
                  'w-full py-3 rounded-[12px] text-sm font-medium transition-colors flex items-center justify-center gap-2',
                  !opt.enabled
                    ? 'bg-bg text-gray-light cursor-not-allowed'
                    : posSystem === opt.id
                      ? 'bg-primary-lighter text-dark border-2 border-primary'
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
    </div>
  )
}

function FilledInput({
  placeholder,
  value,
  onChange,
}: {
  placeholder: string
  value: string
  onChange: (v: string) => void
}) {
  return (
    <input
      placeholder={placeholder}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="w-full bg-bg rounded-[14px] px-4 py-4 text-base text-dark placeholder:text-gray-light outline-none focus:ring-1 focus:ring-primary"
    />
  )
}
