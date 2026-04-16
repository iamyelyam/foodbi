import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { BottomSheet } from './BottomSheet'
import { SearchBar } from '@/components/ui/search-bar'
import { useAppStore } from '@/stores/app'
import { MapPin, Check, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { useT, useI18nStore } from '@/i18n'

interface LocationSwitcherProps {
  isOpen: boolean
  onClose: () => void
}

export function LocationSwitcher({ isOpen, onClose }: LocationSwitcherProps) {
  const navigate = useNavigate()
  const t = useT()
  const locale = useI18nStore((s) => s.locale)
  const { selectedLocationIds, setSelectedLocations } = useAppStore()
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

  const allIds = useMemo(() => locations.map((l: any) => l.id as string), [locations])
  const isAllSelected = selectedLocationIds.length === 0 || selectedLocationIds.length === allIds.length

  const toggleLocation = (id: string) => {
    const set = new Set(selectedLocationIds)
    if (set.has(id)) set.delete(id)
    else set.add(id)
    const next = Array.from(set)
    setSelectedLocations(next.length === allIds.length ? [] : next)
  }

  const selectAll = () => {
    setSelectedLocations([])
  }

  return (
    <BottomSheet isOpen={isOpen} onClose={onClose} title={t('location.selectLocation')}>
      <div className="mb-3">
        <SearchBar
          placeholder={t('location.searchLocations')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onClear={() => setSearch('')}
        />
      </div>

      <div className="space-y-2">
        {/* All locations option */}
        <button
          onClick={selectAll}
          className={cn(
            'w-full flex items-center gap-3 p-3 rounded-[12px] text-left transition-colors',
            isAllSelected ? 'bg-primary/5 ring-1 ring-primary' : 'hover:bg-bg-alt'
          )}
        >
          <div className={cn(
            'w-10 h-10 rounded-full flex items-center justify-center',
            isAllSelected ? 'bg-primary' : 'bg-primary-lighter'
          )}>
            <MapPin className={cn('h-5 w-5', isAllSelected ? 'text-white' : 'text-primary')} />
          </div>
          <div className="flex-1">
            <p className="text-sm font-semibold text-dark">{t('location.allLocations')}</p>
            <p className="text-xs text-gray mt-0.5">{t('location.viewCombinedData')}</p>
          </div>
          {isAllSelected && <Check className="h-5 w-5 text-primary" />}
        </button>

        {filtered.map((loc: any) => {
          const isActive = !isAllSelected && selectedLocationIds.includes(loc.id)
          return (
            <button
              key={loc.id}
              onClick={() => toggleLocation(loc.id)}
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
                    {t('location.synced', { date: new Date(loc.last_synced_at).toLocaleDateString(locale, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) })}
                  </p>
                )}
              </div>
              {isActive && <Check className="h-5 w-5 text-primary" />}
            </button>
          )
        })}

        {filtered.length === 0 && search.trim() && (
          <p className="text-sm text-gray text-center py-4">{t('location.noLocationsFound')}</p>
        )}

        <button
          onClick={() => { onClose(); navigate('/locations/new') }}
          className="w-full flex items-center gap-3 p-3 rounded-[12px] text-left hover:bg-bg-alt"
        >
          <div className="w-10 h-10 rounded-full bg-bg-alt flex items-center justify-center">
            <Plus className="h-5 w-5 text-gray" />
          </div>
          <span className="text-sm font-medium text-primary">{t('location.addLocation')}</span>
        </button>

        <button
          onClick={onClose}
          className="w-full mt-2 py-3 rounded-full bg-primary text-dark font-semibold"
        >
          {t('common.done')}
        </button>
      </div>
    </BottomSheet>
  )
}
