import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ProgressBar } from '@/components/ui/progress-bar'
import { SearchBar } from '@/components/ui/search-bar'
import { Minus, Plus } from 'lucide-react'
import api from '@/lib/api'

interface Item { product_name: string; category: string; quantity: number; unit: string; price_per_unit: number }

export function CreateSupplyPage() {
  const navigate = useNavigate()
  const [step, setStep] = useState(0)
  const [supplier, setSupplier] = useState('')
  const [search, setSearch] = useState('')
  const [items, setItems] = useState<Item[]>([])
  const [newItem, setNewItem] = useState({ product_name: '', category: '', unit: 'kg', price_per_unit: 0 })

  const steps = ['Supplier', 'Add Products', 'Quantities', 'Review']
  const progress = ((step + 1) / steps.length) * 100

  const mutation = useMutation({
    mutationFn: (data: any) => api.post('/supplying', data),
    onSuccess: () => navigate('/supplying'),
  })

  const addItem = () => {
    if (!newItem.product_name) return
    setItems([...items, { ...newItem, quantity: 1 }])
    setNewItem({ product_name: '', category: '', unit: 'kg', price_per_unit: 0 })
  }

  const updateQty = (idx: number, delta: number) => {
    const updated = [...items]
    updated[idx].quantity = Math.max(0.5, updated[idx].quantity + delta)
    setItems(updated)
  }

  const updatePrice = (idx: number, price: number) => {
    const updated = [...items]
    updated[idx].price_per_unit = price
    setItems(updated)
  }

  const removeItem = (idx: number) => setItems(items.filter((_, i) => i !== idx))

  const total = items.reduce((s, i) => s + i.quantity * i.price_per_unit, 0)

  const handleSubmit = () => {
    mutation.mutate({ supplier_name: supplier, location_id: '', items })
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <Header title={steps[step]} showBack />
      <div className="px-4 pt-2 pb-4">
        <ProgressBar value={progress} />
        <p className="text-xs text-gray mt-1">Step {step + 1} of {steps.length}</p>
      </div>

      <div className="flex-1 px-4 pb-4 overflow-y-auto">
        {step === 0 && (
          <div className="space-y-4">
            <Input label="Supplier name" placeholder="Enter supplier" value={supplier} onChange={(e) => setSupplier(e.target.value)} />
          </div>
        )}

        {step === 1 && (
          <div className="space-y-3">
            <SearchBar placeholder="Search products..." value={search} onChange={(e) => setSearch(e.target.value)} onClear={() => setSearch('')} />
            <div className="bg-bg rounded-[12px] p-3 space-y-3">
              <Input placeholder="Product name" value={newItem.product_name} onChange={(e) => setNewItem({ ...newItem, product_name: e.target.value })} />
              <div className="grid grid-cols-2 gap-2">
                <Input placeholder="Category" value={newItem.category} onChange={(e) => setNewItem({ ...newItem, category: e.target.value })} />
                <Input placeholder="Unit" value={newItem.unit} onChange={(e) => setNewItem({ ...newItem, unit: e.target.value })} />
              </div>
              <Button size="sm" onClick={addItem} disabled={!newItem.product_name}>Add Product</Button>
            </div>
            {items.map((item, idx) => (
              <div key={idx} className="flex items-center justify-between bg-bg-alt rounded-[10px] px-3 py-2">
                <span className="text-sm text-dark">{item.product_name}</span>
                <button onClick={() => removeItem(idx)} className="text-xs text-danger">Remove</button>
              </div>
            ))}
          </div>
        )}

        {step === 2 && (
          <div className="space-y-3">
            {items.map((item, idx) => (
              <div key={idx} className="bg-bg rounded-[12px] p-3">
                <p className="text-sm font-semibold text-dark mb-2">{item.product_name}</p>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <button onClick={() => updateQty(idx, -0.5)} className="w-8 h-8 rounded-full bg-bg-alt flex items-center justify-center">
                      <Minus className="h-4 w-4 text-dark" />
                    </button>
                    <span className="text-base font-semibold w-12 text-center">{item.quantity}</span>
                    <button onClick={() => updateQty(idx, 0.5)} className="w-8 h-8 rounded-full bg-primary flex items-center justify-center">
                      <Plus className="h-4 w-4 text-white" />
                    </button>
                    <span className="text-xs text-gray">{item.unit}</span>
                  </div>
                  <div className="w-24">
                    <Input placeholder="Price" type="number" value={item.price_per_unit || ''} onChange={(e) => updatePrice(idx, Number(e.target.value))} />
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {step === 3 && (
          <div className="space-y-3">
            <div className="bg-bg rounded-[12px] p-4">
              <p className="text-sm text-gray">Supplier</p>
              <p className="text-base font-semibold text-dark">{supplier}</p>
            </div>
            {items.map((item, idx) => (
              <div key={idx} className="flex items-center justify-between bg-bg rounded-[12px] px-4 py-3">
                <div>
                  <p className="text-sm font-medium text-dark">{item.product_name}</p>
                  <p className="text-xs text-gray">{item.quantity} {item.unit}</p>
                </div>
                <p className="text-sm font-semibold text-dark">€{(item.quantity * item.price_per_unit).toFixed(2)}</p>
              </div>
            ))}
            <div className="flex items-center justify-between bg-primary/5 rounded-[12px] px-4 py-3">
              <span className="text-sm font-semibold text-dark">Total</span>
              <span className="text-lg font-bold text-primary">€{total.toFixed(2)}</span>
            </div>
          </div>
        )}
      </div>

      <div className="px-4 pb-8 flex gap-3">
        {step > 0 && <Button variant="secondary" fullWidth onClick={() => setStep(step - 1)}>Back</Button>}
        {step < 3 ? (
          <Button fullWidth onClick={() => setStep(step + 1)}
            disabled={(step === 0 && !supplier) || (step === 1 && items.length === 0)}>
            Next
          </Button>
        ) : (
          <Button fullWidth onClick={handleSubmit} disabled={mutation.isPending}>
            {mutation.isPending ? 'Submitting...' : 'Confirm Request'}
          </Button>
        )}
      </div>
    </div>
  )
}
