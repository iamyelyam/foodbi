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
    <nav className="fixed inset-x-0 bottom-0 z-50 bg-white pb-[env(safe-area-inset-bottom)]">
      <div className="h-[2px] bg-bg-alt" />
      <div className="flex items-center justify-around h-[62px] px-4">
        {tabs.map(({ to, icon, label }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              cn(
                'flex flex-col items-center gap-1 px-3 pt-2',
                isActive ? 'opacity-100' : 'opacity-40'
              )
            }
          >
            <img src={icon} alt={label} className="h-8 w-8" />
            <span className="text-[10px] font-semibold text-dark">{label}</span>
          </NavLink>
        ))}
      </div>
    </nav>
  )
}
