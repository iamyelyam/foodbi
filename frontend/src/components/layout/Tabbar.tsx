import { NavLink } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { useT } from '@/i18n'

export function Tabbar() {
  const t = useT()

  const tabs = [
    { to: '/', icon: '/illustrations/tabbar-home.png', label: t('nav.main') },
    { to: '/file-upload', icon: '/illustrations/tabbar-upload.png', label: t('nav.upload') },
    { to: '/employees', icon: '/illustrations/tabbar-employees.png', label: t('nav.employees') },
    { to: '/profile', icon: '/illustrations/tabbar-profile.png', label: t('nav.profile') },
  ]

  return (
    <nav className="fixed inset-x-0 bottom-0 z-50 bg-white">
      {/* Top divider */}
      <div className="h-[1px] bg-bg-alt" />

      {/* Tab icons + labels — minimal padding to push as low as possible */}
      <div className="flex items-center justify-around px-4 pt-1.5 pb-0.5">
        {tabs.map(({ to, icon, label }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              cn(
                'flex flex-col items-center gap-1 px-3',
                isActive ? 'opacity-100' : 'opacity-40'
              )
            }
          >
            <img src={icon} alt={label} className="h-7 w-7" />
            <span className="text-[10px] font-semibold text-dark">{label}</span>
          </NavLink>
        ))}
      </div>

      {/* No safe-area spacer — Capacitor contentInset:'always' handles it natively */}
    </nav>
  )
}
