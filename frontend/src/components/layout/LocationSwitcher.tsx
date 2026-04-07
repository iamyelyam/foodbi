import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { BottomSheet } from './BottomSheet'
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

  const { data: locations = [] } = useQuery({
    queryKey: ['locations'],
    queryFn: () => api.get('/locations').then((r) => r.data),
    enabled: isOpen,
  })

  const handleSelect = (id: string) => {
    setActiveLocation(id)
    onClose()
  }

  return (
    <BottomSheet isOpen={isOpen} onClose={onClose} title="Select Location">
      <div className="space-y-2">
        {locations.map((loc: any) => {
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
              </div>
              {isActive && <Check className="h-5 w-5 text-primary" />}
            </button>
          )
        })}

        <button
          onClick={() => { onClose(); navigate('/locations') }}
          className="w-full flex items-center gap-3 p-3 rounded-[12px] text-left hover:bg-bg-alt"
        >
          <div className="w-10 h-10 rounded-full bg-bg-alt flex items-center justify-center">
            <Plus className="h-5 w-5 text-gray" />
          </div>
          <span className="text-sm font-medium text-primary">Add location</span>
        </button>
      </div>
    </BottomSheet>
  )
}
