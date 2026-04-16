import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CardSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import { Snackbar } from '@/components/ui/snackbar'
import { MapPin, Plus, RefreshCw, Check, Pencil, Trash2, MoreVertical } from 'lucide-react'
import { useAppStore } from '@/stores/app'
import api from '@/lib/api'
import { findPosLabel } from '@/lib/posSystems'
import { useT, useI18nStore } from '@/i18n'

interface Location {
  id: string
  name: string
  address: string
  pos_system?: string
  iiko_org_id?: string
  currency_symbol?: string
  locale?: string
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

  // Edit state
  const [editingLoc, setEditingLoc] = useState<Location | null>(null)
  const [editName, setEditName] = useState('')
  const [editAddress, setEditAddress] = useState('')
  const [editIikoUrl, setEditIikoUrl] = useState('')
  const [editIikoLogin, setEditIikoLogin] = useState('')
  const [editIikoPassword, setEditIikoPassword] = useState('')

  // NUMIER edit state
  const [editNumierApiKey, setEditNumierApiKey] = useState('')

  // Delete state
  const [deletingLoc, setDeletingLoc] = useState<Location | null>(null)

  // Context menu state
  const [menuLocId, setMenuLocId] = useState<string | null>(null)

  const setLocationCurrencies = useAppStore((s) => s.setLocationCurrencies)

  const { data: locations = [], isLoading } = useQuery<Location[]>({
    queryKey: ['locations'],
    queryFn: () => api.get('/locations').then(r => r.data),
  })

  // Sync per-location currencies to global store
  useEffect(() => {
    if (locations.length > 0) {
      const map: Record<string, string> = {}
      for (const loc of locations) {
        if (loc.id && loc.currency_symbol) map[loc.id] = loc.currency_symbol
      }
      setLocationCurrencies(map)
    }
  }, [locations, setLocationCurrencies])

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

  const updateMutation = useMutation({
    mutationFn: async (data: { id: string; name: string; address: string; posSystem?: string; iikoUrl: string; iikoLogin: string; iikoPassword: string; numierApiKey?: string }) => {
      await api.put(`/locations/${data.id}`, { name: data.name, address: data.address })
      if (data.posSystem === 'numier') {
        if (data.numierApiKey) {
          await api.put('/locations/numier-config', { numier_api_key: data.numierApiKey })
        }
      } else {
        // Only update iiko config if all three fields are filled (password required for security)
        if (data.iikoUrl && data.iikoLogin && data.iikoPassword) {
          await api.put('/locations/iiko-config', {
            iiko_server_url: data.iikoUrl,
            iiko_login: data.iikoLogin,
            iiko_password: data.iikoPassword,
          })
        }
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      setEditingLoc(null)
    },
  })

  const [deleteError, setDeleteError] = useState('')
  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/locations/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      setDeletingLoc(null)
      setDeleteError('')
    },
    onError: (err: any) => {
      setDeleteError(err.response?.data?.error || 'Failed to delete')
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

  const openEdit = async (loc: Location) => {
    setEditName(loc.name)
    setEditAddress(loc.address || '')
    setEditIikoUrl('')
    setEditIikoLogin('')
    setEditIikoPassword('')
    setEditNumierApiKey('')
    setEditingLoc(loc)
    setMenuLocId(null)
    if (loc.pos_system === 'numier') {
      // NUMIER config: API key is masked, nothing to prefill
    } else {
      try {
        const { data } = await api.get('/locations/iiko-config')
        setEditIikoUrl(data.iiko_server_url || '')
        setEditIikoLogin(data.iiko_login || '')
      } catch { /* ignore */ }
    }
  }

  const openDelete = (loc: Location) => {
    setDeletingLoc(loc)
    setMenuLocId(null)
  }

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={t('locations.title')} showBack showNotification badgeCount={unreadCount} />

      <main className="flex-1 px-4 pt-4 pb-28">
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
            const showMenu = menuLocId === loc.id

            return (
              <div key={loc.id} className="relative">
                <button
                  onClick={() => setActiveLocation(loc.id)}
                  className={`w-full bg-white rounded-[16px] p-4 text-left transition-all ${
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
                        <div className="flex items-center gap-2">
                          <p className="text-sm font-semibold text-dark">{loc.name}</p>
                          {loc.pos_system && (
                            <span className="text-[10px] font-medium text-gray bg-bg px-1.5 py-0.5 rounded">
                              {findPosLabel(loc.pos_system)}
                            </span>
                          )}
                        </div>
                        {loc.address && (
                          <p className="text-xs text-gray mt-0.5">{loc.address}</p>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-1">
                      {isActive && <Check className="h-5 w-5 text-primary" />}
                      <button
                        onClick={(e) => { e.stopPropagation(); setMenuLocId(showMenu ? null : loc.id) }}
                        className="w-8 h-8 flex items-center justify-center rounded-full active:bg-bg"
                      >
                        <MoreVertical className="h-4 w-4 text-gray" />
                      </button>
                    </div>
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

                {/* Context menu dropdown */}
                {showMenu && (
                  <>
                    <div className="fixed inset-0 z-30" onClick={() => setMenuLocId(null)} />
                    <div className="absolute right-4 top-12 z-40 bg-white rounded-[12px] shadow-lg border border-bg-alt py-1 min-w-[180px]">
                      <button
                        onClick={() => openEdit(loc)}
                        className="w-full flex items-center gap-3 px-4 py-3 text-sm text-dark active:bg-bg"
                      >
                        <Pencil className="h-4 w-4 text-gray" />
                        {t('locations.editLocation')}
                      </button>
                      <button
                        onClick={() => openDelete(loc)}
                        className="w-full flex items-center gap-3 px-4 py-3 text-sm text-danger active:bg-bg"
                      >
                        <Trash2 className="h-4 w-4 text-danger" />
                        {t('locations.deleteLocation')}
                      </button>
                    </div>
                  </>
                )}
              </div>
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

      {/* Add location sheet */}
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

      {/* Edit location sheet */}
      <BottomSheet isOpen={!!editingLoc} onClose={() => setEditingLoc(null)} title={t('locations.editLocation')}>
        <div className="flex flex-col gap-4">
          <Input
            label={t('locations.restaurantName')}
            placeholder={t('locations.streetExample')}
            value={editName}
            onChange={(e) => setEditName(e.target.value)}
          />
          <Input
            label={t('locations.addressLabel')}
            placeholder={t('locations.streetAddressExample')}
            value={editAddress}
            onChange={(e) => setEditAddress(e.target.value)}
          />

          <div className="border-t border-bg-alt pt-4 mt-1">
            <p className="text-xs font-semibold text-gray mb-3">{t('locations.posConfig')}</p>
            <div className="flex flex-col gap-3">
              {editingLoc?.pos_system === 'numier' ? (
                <Input
                  label="API Key"
                  placeholder={t('locations.credentialsPh')}
                  value={editNumierApiKey}
                  onChange={(e) => setEditNumierApiKey(e.target.value)}
                />
              ) : (
                <>
                  <Input
                    label="Server URL"
                    placeholder="https://..."
                    value={editIikoUrl}
                    onChange={(e) => setEditIikoUrl(e.target.value)}
                  />
                  <Input
                    label={t('locations.login')}
                    placeholder="Login"
                    value={editIikoLogin}
                    onChange={(e) => setEditIikoLogin(e.target.value)}
                  />
                  <Input
                    label={t('locations.password')}
                    placeholder={t('locations.credentialsPh')}
                    type="password"
                    value={editIikoPassword}
                    onChange={(e) => setEditIikoPassword(e.target.value)}
                  />
                </>
              )}
            </div>
          </div>

          <Button
            fullWidth
            onClick={() => editingLoc && updateMutation.mutate({
              id: editingLoc.id,
              name: editName,
              address: editAddress,
              posSystem: editingLoc.pos_system,
              iikoUrl: editIikoUrl,
              iikoLogin: editIikoLogin,
              iikoPassword: editIikoPassword,
              numierApiKey: editNumierApiKey || undefined,
            })}
            disabled={!editName || updateMutation.isPending}
          >
            {updateMutation.isPending ? t('common.saving') : t('common.save')}
          </Button>
        </div>
      </BottomSheet>

      {/* Delete confirmation sheet */}
      <BottomSheet isOpen={!!deletingLoc} onClose={() => { setDeletingLoc(null); setDeleteError('') }} title={t('locations.deleteLocation')}>
        <div className="flex flex-col gap-4">
          <p className="text-sm text-gray text-center">
            {t('locations.deleteConfirm', { name: deletingLoc?.name || '' })}
          </p>
          {deleteError && (
            <p className="text-sm text-danger text-center">{deleteError}</p>
          )}
          <div className="flex gap-3">
            <Button variant="secondary" fullWidth onClick={() => { setDeletingLoc(null); setDeleteError('') }}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="danger"
              fullWidth
              onClick={() => deletingLoc && deleteMutation.mutate(deletingLoc.id)}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? t('locations.deleting') : t('common.delete')}
            </Button>
          </div>
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
