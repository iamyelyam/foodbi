import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ProgressBar } from '@/components/ui/progress-bar'
import { MapPin, Check, Minus, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'

interface Item { product_name: string; category: string; quantity: number; unit: string }

export function CreateTransferPage() {
  const navigate = useNavigate()
  const [step, setStep] = useState(0)
  const [fromLoc, setFromLoc] = useState<any>(null)
  const [toLoc, setToLoc] = useState<any>(null)
  const [items, setItems] = useState<Item[]>([])
  const [newName, setNewName] = useState('')

  const steps = ['From', 'To', 'Products', 'Quantities', 'Review']
  const progress = ((step + 1) / steps.length) * 100

  const { data: locations = [] } = useQuery({
    queryKey: ['locations'],
    queryFn: () => api.get('/locations').then((r) => r.data),
  })

  const mutation = useMutation({
    mutationFn: (data: any) => api.post('/transfers', data),
    onSuccess: () => navigate('/transfers'),
  })

  const addItem = () => {
    if (!newName) return
    setItems([...items, { product_name: newName, category: '', quantity: 1, unit: 'kg' }])
    setNewName('')
  }

  const updateQty = (idx: number, delta: number) => {
    const u = [...items]
    u[idx].quantity = Math.max(0.5, u[idx].quantity + delta)
    setItems(u)
  }

  const LocationList = ({ selected, onSelect, exclude }: { selected: any; onSelect: (l: any) => void; exclude?: string }) => (
    <div className="space-y-2">
      {locations.filter((l: any) => l.id !== exclude).map((loc: any) => (
        <button key={loc.id} onClick={() => onSelect(loc)}
          className={cn('w-full flex items-center gap-3 p-3 rounded-[12px] text-left',
            selected?.id === loc.id ? 'bg-primary/5 ring-1 ring-primary' : 'bg-bg')}>
          <div className={cn('w-10 h-10 rounded-full flex items-center justify-center',
            selected?.id === loc.id ? 'bg-primary' : 'bg-primary-lighter')}>
            <MapPin className={cn('h-5 w-5', selected?.id === loc.id ? 'text-white' : 'text-primary')} />
          </div>
          <div className="flex-1">
            <p className="text-sm font-semibold text-dark">{loc.name}</p>
            {loc.address && <p className="text-xs text-gray">{loc.address}</p>}
          </div>
          {selected?.id === loc.id && <Check className="h-5 w-5 text-primary" />}
        </button>
      ))}
    </div>
  )

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <Header title={steps[step]} showBack />
      <div className="px-4 pt-2 pb-4">
        <ProgressBar value={progress} />
        <p className="text-xs text-gray mt-1">Step {step + 1} of {steps.length}</p>
      </div>

      <div className="flex-1 px-4 pb-4 overflow-y-auto">
        {step === 0 && <LocationList selected={fromLoc} onSelect={setFromLoc} />}
        {step === 1 && <LocationList selected={toLoc} onSelect={setToLoc} exclude={fromLoc?.id} />}

        {step === 2 && (
          <div className="space-y-3">
            <div className="flex gap-2">
              <Input placeholder="Product name" value={newName} onChange={(e) => setNewName(e.target.value)} className="flex-1" />
              <Button size="sm" onClick={addItem} disabled={!newName}>Add</Button>
            </div>
            {items.map((item, idx) => (
              <div key={idx} className="flex items-center justify-between bg-bg rounded-[10px] px-3 py-2">
                <span className="text-sm text-dark">{item.product_name}</span>
                <button onClick={() => setItems(items.filter((_, i) => i !== idx))} className="text-xs text-danger">Remove</button>
              </div>
            ))}
          </div>
        )}

        {step === 3 && (
          <div className="space-y-3">
            {items.map((item, idx) => (
              <div key={idx} className="bg-bg rounded-[12px] p-3">
                <p className="text-sm font-semibold text-dark mb-2">{item.product_name}</p>
                <div className="flex items-center gap-3">
                  <button onClick={() => updateQty(idx, -0.5)} className="w-8 h-8 rounded-full bg-bg-alt flex items-center justify-center">
                    <Minus className="h-4 w-4" />
                  </button>
                  <span className="text-base font-semibold w-12 text-center">{item.quantity}</span>
                  <button onClick={() => updateQty(idx, 0.5)} className="w-8 h-8 rounded-full bg-primary flex items-center justify-center">
                    <Plus className="h-4 w-4 text-white" />
                  </button>
                  <span className="text-xs text-gray">{item.unit}</span>
                </div>
              </div>
            ))}
          </div>
        )}

        {step === 4 && (
          <div className="space-y-3">
            <div className="bg-bg rounded-[12px] p-4 space-y-2">
              <div className="flex justify-between text-sm"><span className="text-gray">From</span><span className="font-semibold text-dark">{fromLoc?.name}</span></div>
              <div className="flex justify-between text-sm"><span className="text-gray">To</span><span className="font-semibold text-dark">{toLoc?.name}</span></div>
              <div className="flex justify-between text-sm"><span className="text-gray">Items</span><span className="font-semibold text-dark">{items.length}</span></div>
            </div>
            {items.map((item, idx) => (
              <div key={idx} className="flex items-center justify-between bg-bg rounded-[10px] px-4 py-2">
                <span className="text-sm text-dark">{item.product_name}</span>
                <span className="text-sm font-semibold text-dark">{item.quantity} {item.unit}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="px-4 pb-8 flex gap-3">
        {step > 0 && <Button variant="secondary" fullWidth onClick={() => setStep(step - 1)}>Back</Button>}
        {step < 4 ? (
          <Button fullWidth onClick={() => setStep(step + 1)}
            disabled={(step === 0 && !fromLoc) || (step === 1 && !toLoc) || (step === 2 && items.length === 0)}>
            Next
          </Button>
        ) : (
          <Button fullWidth onClick={() => mutation.mutate({ from_location_id: fromLoc.id, to_location_id: toLoc.id, items })}
            disabled={mutation.isPending}>
            {mutation.isPending ? 'Creating...' : 'Confirm Transfer'}
          </Button>
        )}
      </div>
    </div>
  )
}
