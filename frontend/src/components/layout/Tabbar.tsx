import { NavLink } from 'react-router-dom'
import { Home, TrendingUp, Package, ShoppingCart, User } from 'lucide-react'
import { cn } from '@/lib/utils'

const tabs = [
  { to: '/', icon: Home, label: 'Home' },
  { to: '/revenue', icon: TrendingUp, label: 'Revenue' },
  { to: '/stock', icon: Package, label: 'Stock' },
  { to: '/purchases', icon: ShoppingCart, label: 'Purchases' },
  { to: '/profile', icon: User, label: 'Profile' },
]

export function Tabbar() {
  return (
    <nav className="fixed bottom-0 left-1/2 -translate-x-1/2 w-full max-w-[375px] bg-white border-t border-bg-alt">
      <div className="flex items-center justify-around h-16 px-2">
        {tabs.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              cn(
                'flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-lg transition-colors',
                isActive ? 'text-primary' : 'text-gray-light'
              )
            }
          >
            <Icon className="h-5 w-5" />
            <span className="text-[10px] font-medium">{label}</span>
          </NavLink>
        ))}
      </div>
    </nav>
  )
}
