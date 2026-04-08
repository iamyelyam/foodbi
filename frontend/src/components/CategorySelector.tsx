import { useState } from 'react'
import { SearchBar } from '@/components/ui/search-bar'
import { cn } from '@/lib/utils'
import {
  Wine,
  Beef,
  Fish,
  Apple,
  Wheat,
  Milk,
  Carrot,
  Cookie,
  Package,
  IceCreamCone,
  Flame,
  Citrus,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

interface Category {
  name: string
  icon: LucideIcon
}

const categories: Category[] = [
  { name: 'Liquor', icon: Wine },
  { name: 'Creme', icon: IceCreamCone },
  { name: 'Meat', icon: Beef },
  { name: 'Conserves', icon: Package },
  { name: 'Sause', icon: Flame },
  { name: 'Seasoning', icon: Citrus },
  { name: 'Desserts', icon: Cookie },
  { name: 'Seafood', icon: Fish },
  { name: 'Fruits', icon: Apple },
  { name: 'Bakery', icon: Wheat },
  { name: 'Vegetables', icon: Carrot },
  { name: 'Dairy', icon: Milk },
]

interface CategorySelectorProps {
  selected: string
  onSelect: (category: string) => void
}

export function CategorySelector({ selected, onSelect }: CategorySelectorProps) {
  const [search, setSearch] = useState('')

  const filtered = categories.filter((c) =>
    c.name.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="space-y-4">
      <SearchBar
        placeholder="Search categories..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        onClear={() => setSearch('')}
      />

      <div className="grid grid-cols-3 gap-3">
        {filtered.map((cat) => {
          const Icon = cat.icon
          const isSelected = selected === cat.name
          return (
            <button
              key={cat.name}
              onClick={() => onSelect(cat.name)}
              className={cn(
                'flex flex-col items-center justify-center gap-2 rounded-[16px] bg-bg-alt p-4 transition-colors',
                'w-full aspect-square',
                isSelected && 'ring-2 ring-primary bg-primary-lighter'
              )}
            >
              <div className="w-16 h-16 flex items-center justify-center">
                <Icon className={cn('h-8 w-8', isSelected ? 'text-primary' : 'text-gray')} />
              </div>
              <span
                className={cn(
                  'text-sm font-medium',
                  isSelected ? 'text-primary' : 'text-dark'
                )}
              >
                {cat.name}
              </span>
            </button>
          )
        })}
      </div>

      {filtered.length === 0 && (
        <p className="text-center text-sm text-gray py-8">No categories found</p>
      )}
    </div>
  )
}
