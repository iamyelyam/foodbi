import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CardSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import { Snackbar } from '@/components/ui/snackbar'
import { MapPin, Plus, RefreshCw, Check } from 'lucide-react'
import { useAppStore } from '@/stores/app'
import api from '@/lib/api'
import { useT, useI18nStore } from '@/i18n'

interface Location {
  id: string
  name: string
  address: string
  iiko_org_id?: string
}

interface SyncStatus {
  location_id: string
  sync_type: string
  status: string
  records_synced: number
  started_at: string
  completed_at?: string
}

export function LocationsPage() {
  const t = useT()
  const locale = useI18nStore((s) => s.locale)
  const queryClient = useQueryClient()
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const { activeLocationId, setActiveLocation } = useAppStore()
  const [showAdd, setShowAdd] = useState(false)
  const [showSyncSuccess, setShowSyncSuccess] = useState(false)
  const [name, setName] = useState('')
  const [address, setAddress] = useState('')
  const [iikoOrgId, setIikoOrgId] = useState('')

  const { data: locations = [], isLoading } = useQuery<Location[]>({
    queryKey: ['locations'],
    queryFn: () => api.get('/locations').then(r => r.data),
  })

  const { data: syncStatus = [] } = useQuery<SyncStatus[]>({
    queryKey: ['sync-status'],
    queryFn: () => api.get('/locations/sync-status').then(r => r.data),
    refetchInterval: 30000,
  })

  const addMutation = useMutation({
    mutationFn: (data: { name: string; address: string; iiko_org_id: string }) =>
      api.post('/locations', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      setShowAdd(false)
      setName('')
      setAddress('')
      setIikoOrgId('')
    },
  })

  const syncMutation = useMutation({
    mutationFn: (locId: string) => api.post(`/locations/${locId}/sync`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sync-status'] })
      setShowSyncSuccess(true)
    },
  })

  const getLastSync = (locationId: string) => {
    const entries = syncStatus.filter(s => s.location_id === locationId && s.status === 'success')
    if (entries.length === 0) return null
    return entries.sort((a, b) => b.started_at.localeCompare(a.started_at))[0]
  }

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={t('locations.title')} showBack showNotification badgeCount={unreadCount} />

      <main className="flex-1 px-4 pt-4 pb-20">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-semibold text-dark">
            {locations.length === 1
              ? t('locations.countSingular', { count: locations.length })
              : t('locations.countLabel', { count: locations.length })}
          </h2>
          <button
            onClick={() => setShowAdd(true)}
            className="flex items-center gap-1.5 text-sm font-medium text-primary"
          >
            <Plus className="h-4 w-4" /> {t('common.add')}
          </button>
        </div>

        {isLoading ? (
          <div className="flex flex-col gap-3">
            <CardSkeleton />
            <CardSkeleton />
          </div>
        ) : (
        <div className="flex flex-col gap-3">
          {locations.map((loc) => {
            const isActive = activeLocationId === loc.id
            const lastSync = getLastSync(loc.id)

            return (
              <button
                key={loc.id}
                onClick={() => setActiveLocation(loc.id)}
                className={`bg-white rounded-[16px] p-4 text-left transition-all ${
                  isActive ? 'ring-2 ring-primary' : ''
                }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-3">
                    <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
                      isActive ? 'bg-primary' : 'bg-primary-lighter'
                    }`}>
                      <MapPin className={`h-5 w-5 ${isActive ? 'text-white' : 'text-primary'}`} />
                    </div>
                    <div>
                      <p className="text-sm font-semibold text-dark">{loc.name}</p>
                      {loc.address && (
                        <p className="text-xs text-gray mt-0.5">{loc.address}</p>
                      )}
                    </div>
                  </div>
                  {isActive && <Check className="h-5 w-5 text-primary" />}
                </div>

                <div className="mt-3 flex items-center justify-between">
                  {lastSync ? (
                    <div className="flex items-center gap-1.5 text-xs text-gray">
                      <RefreshCw className="h-3 w-3" />
                      <span>{t('locations.lastSync', { time: new Date(lastSync.started_at).toLocaleTimeString(locale) })}</span>
                    </div>
                  ) : (
                    <span className="text-xs text-gray">{t('locations.notSyncedYet')}</span>
                  )}
                  <button
                    onClick={(e) => { e.stopPropagation(); syncMutation.mutate(loc.id) }}
                    disabled={syncMutation.isPending}
                    className="text-xs font-medium text-primary bg-primary-lighter px-3 py-1 rounded-full"
                  >
                    {syncMutation.isPending ? t('locations.syncing') : t('locations.syncNow')}
                  </button>
                </div>
              </button>
            )
          })}

          {locations.length === 0 && (
            <div className="text-center py-12">
              <MapPin className="h-12 w-12 text-gray-light mx-auto mb-3" />
              <p className="text-sm text-gray">{t('locations.noLocationsYet')}</p>
              <Button variant="primary" size="sm" className="mt-4" onClick={() => setShowAdd(true)}>
                {t('locations.addFirstLocation')}
              </Button>
            </div>
          )}
        </div>
        )}
      </main>

      <Tabbar />

      <BottomSheet isOpen={showAdd} onClose={() => setShowAdd(false)} title={t('locations.addLocation')}>
        <div className="flex flex-col gap-4">
          <Input
            label={t('locations.restaurantName')}
            placeholder={t('locations.streetExample')}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <Input
            label={t('locations.addressLabel')}
            placeholder={t('locations.streetAddressExample')}
            value={address}
            onChange={(e) => setAddress(e.target.value)}
          />
          <Input
            label={t('locations.iikoOrgId')}
            placeholder={t('locations.iikoOrgIdPh')}
            value={iikoOrgId}
            onChange={(e) => setIikoOrgId(e.target.value)}
          />
          <Button
            fullWidth
            onClick={() => addMutation.mutate({ name, address, iiko_org_id: iikoOrgId })}
            disabled={!name || addMutation.isPending}
          >
            {addMutation.isPending ? t('common.adding') : t('locations.addLocation')}
          </Button>
        </div>
      </BottomSheet>

      <Snackbar
        isOpen={showSyncSuccess}
        onClose={() => setShowSyncSuccess(false)}
        message={t('locations.syncQueued')}
        type="success"
      />
    </div>
  )
}
