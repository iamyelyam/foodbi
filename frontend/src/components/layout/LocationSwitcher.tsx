import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { BottomSheet } from './BottomSheet'
import { SearchBar } from '@/components/ui/search-bar'
import { useAppStore } from '@/stores/app'
import { MapPin, Check, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'

interface LocationSwitcherProps {
  isOpen: boolean
  onClose: () => void
}

export function LocationSwitcher({ isOpen, onClose }: LocationSwitcherProps) {
  const navigate = useNavigate()
  const { activeLocationId, setActiveLocation } = useAppStore()
  const [search, setSearch] = useState('')

  const { data: locations = [] } = useQuery({
    queryKey: ['locations'],
    queryFn: () => api.get('/locations').then((r) => r.data),
    enabled: isOpen,
  })

  const filtered = useMemo(() => {
    if (!search.trim()) return locations
    const q = search.toLowerCase()
    return locations.filter(
      (loc: any) =>
        loc.name?.toLowerCase().includes(q) ||
        loc.address?.toLowerCase().includes(q)
    )
  }, [locations, search])

  const handleSelect = (id: string) => {
    setActiveLocation(id)
    onClose()
  }

  return (
    <BottomSheet isOpen={isOpen} onClose={onClose} title="Select Location">
      <div className="mb-3">
        <SearchBar
          placeholder="Search locations..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onClear={() => setSearch('')}
        />
      </div>

      <div className="space-y-2">
        {/* All locations option */}
        <button
          onClick={() => handleSelect('')}
          className={cn(
            'w-full flex items-center gap-3 p-3 rounded-[12px] text-left transition-colors',
            !activeLocationId ? 'bg-primary/5 ring-1 ring-primary' : 'hover:bg-bg-alt'
          )}
        >
          <div className={cn(
            'w-10 h-10 rounded-full flex items-center justify-center',
            !activeLocationId ? 'bg-primary' : 'bg-primary-lighter'
          )}>
            <MapPin className={cn('h-5 w-5', !activeLocationId ? 'text-white' : 'text-primary')} />
          </div>
          <div className="flex-1">
            <p className="text-sm font-semibold text-dark">All locations</p>
            <p className="text-xs text-gray mt-0.5">View combined data</p>
          </div>
          {!activeLocationId && <Check className="h-5 w-5 text-primary" />}
        </button>

        {filtered.map((loc: any) => {
          const isActive = activeLocationId === loc.id
          return (
            <button
              key={loc.id}
              onClick={() => handleSelect(loc.id)}
              className={cn(
                'w-full flex items-center gap-3 p-3 rounded-[12px] text-left transition-colors',
                isActive ? 'bg-primary/5 ring-1 ring-primary' : 'hover:bg-bg-alt'
              )}
            >
              <div className={cn(
                'w-10 h-10 rounded-full flex items-center justify-center',
                isActive ? 'bg-primary' : 'bg-primary-lighter'
              )}>
                <MapPin className={cn('h-5 w-5', isActive ? 'text-white' : 'text-primary')} />
              </div>
              <div className="flex-1">
                <p className="text-sm font-semibold text-dark">{loc.name}</p>
                {loc.address && <p className="text-xs text-gray mt-0.5">{loc.address}</p>}
                {loc.last_synced_at && (
                  <p className="text-xs text-gray mt-0.5">
                    Synced {new Date(loc.last_synced_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                  </p>
                )}
              </div>
              {isActive && <Check className="h-5 w-5 text-primary" />}
            </button>
          )
        })}

        {filtered.length === 0 && search.trim() && (
          <p className="text-sm text-gray text-center py-4">No locations found</p>
        )}

        <button
          onClick={() => { onClose(); navigate('/locations/new') }}
          className="w-full flex items-center gap-3 p-3 rounded-[12px] text-left hover:bg-bg-alt"
        >
          <div className="w-10 h-10 rounded-full bg-bg-alt flex items-center justify-center">
            <Plus className="h-5 w-5 text-gray" />
          </div>
          <span className="text-sm font-medium text-primary">Add Location</span>
        </button>
      </div>
    </BottomSheet>
  )
}
