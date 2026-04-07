import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { MapPin, Plus, RefreshCw, Check } from 'lucide-react'
import { useAppStore } from '@/stores/app'
import api from '@/lib/api'

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
  const queryClient = useQueryClient()
  const { activeLocationId, setActiveLocation } = useAppStore()
  const [showAdd, setShowAdd] = useState(false)
  const [name, setName] = useState('')
  const [address, setAddress] = useState('')
  const [iikoOrgId, setIikoOrgId] = useState('')

  const { data: locations = [] } = useQuery<Location[]>({
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

  const getLastSync = (locationId: string) => {
    const entries = syncStatus.filter(s => s.location_id === locationId && s.status === 'success')
    if (entries.length === 0) return null
    return entries.sort((a, b) => b.started_at.localeCompare(a.started_at))[0]
  }

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="Locations" showBack showNotification />

      <main className="flex-1 px-4 pt-4 pb-20">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-semibold text-dark">
            {locations.length} location{locations.length !== 1 ? 's' : ''}
          </h2>
          <button
            onClick={() => setShowAdd(true)}
            className="flex items-center gap-1.5 text-sm font-medium text-primary"
          >
            <Plus className="h-4 w-4" /> Add
          </button>
        </div>

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

                {lastSync && (
                  <div className="mt-3 flex items-center gap-1.5 text-xs text-gray">
                    <RefreshCw className="h-3 w-3" />
                    <span>
                      Last sync: {new Date(lastSync.started_at).toLocaleTimeString()} ({lastSync.records_synced} records)
                    </span>
                  </div>
                )}
              </button>
            )
          })}

          {locations.length === 0 && (
            <div className="text-center py-12">
              <MapPin className="h-12 w-12 text-gray-light mx-auto mb-3" />
              <p className="text-sm text-gray">No locations yet</p>
              <Button variant="primary" size="sm" className="mt-4" onClick={() => setShowAdd(true)}>
                Add your first location
              </Button>
            </div>
          )}
        </div>
      </main>

      <Tabbar />

      <BottomSheet isOpen={showAdd} onClose={() => setShowAdd(false)} title="Add Location">
        <div className="flex flex-col gap-4">
          <Input
            label="Restaurant name"
            placeholder="e.g. Main Street Branch"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <Input
            label="Address"
            placeholder="123 Main St"
            value={address}
            onChange={(e) => setAddress(e.target.value)}
          />
          <Input
            label="iiko Organization ID"
            placeholder="From iiko Cloud settings"
            value={iikoOrgId}
            onChange={(e) => setIikoOrgId(e.target.value)}
          />
          <Button
            fullWidth
            onClick={() => addMutation.mutate({ name, address, iiko_org_id: iikoOrgId })}
            disabled={!name || addMutation.isPending}
          >
            {addMutation.isPending ? 'Adding...' : 'Add Location'}
          </Button>
        </div>
      </BottomSheet>
    </div>
  )
}
