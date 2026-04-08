import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Toggle } from '@/components/ui/toggle'
import { useAuthStore } from '@/stores/auth'
import { User, Mail, Phone, Building2, LogOut, Users, MapPin } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { useI18nStore, LOCALE_NAMES, type Locale } from '@/i18n'
import { useT } from '@/i18n'

export function ProfilePage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { logout } = useAuthStore()
  const t = useT()
  const { locale, setLocale } = useI18nStore()
  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState({ first_name: '', last_name: '', phone: '' })
  const [formErrors, setFormErrors] = useState<{ first_name?: string; last_name?: string }>({})
  const [notifications, setNotifications] = useState(true)
  const [faceId, setFaceId] = useState(false)

  const { data: profile } = useQuery({
    queryKey: ['profile'],
    queryFn: () => api.get('/profile/me').then((r) => r.data),
  })

  useEffect(() => {
    if (profile && !editing) {
      setForm({ first_name: profile.first_name, last_name: profile.last_name, phone: profile.phone || '' })
    }
  }, [profile, editing])

  const updateMutation = useMutation({
    mutationFn: (data: typeof form) => api.put('/profile/me', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['profile'] })
      setEditing(false)
    },
  })

  const handleLogout = () => {
    api.post('/auth/logout').catch(() => {})
    logout()
    navigate('/login')
  }

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={t('profile.title')} showBack />

      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {/* Avatar + name + role badge */}
        <div className="bg-white rounded-[16px] p-6 shadow-sm flex flex-col items-center">
          <div className="w-20 h-20 rounded-full bg-primary-lighter flex items-center justify-center">
            <User className="h-10 w-10 text-primary" />
          </div>
          <p className="mt-3 text-lg font-bold text-dark">
            {profile?.first_name} {profile?.last_name}
          </p>
          <span className={cn(
            'text-xs px-3 py-1 rounded-full font-medium capitalize mt-1',
            profile?.role === 'owner' ? 'bg-warning/10 text-warning' : 'bg-primary-lighter text-primary'
          )}>
            {profile?.role}
          </span>
        </div>

        {/* Info or edit form */}
        {!editing ? (
          <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-4">
            <div className="flex items-center gap-3">
              <Mail className="h-5 w-5 text-gray" />
              <div>
                <p className="text-xs text-gray">Email</p>
                <p className="text-sm text-dark">{profile?.email}</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Phone className="h-5 w-5 text-gray" />
              <div>
                <p className="text-xs text-gray">Phone</p>
                <p className="text-sm text-dark">{profile?.phone || 'Not set'}</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Building2 className="h-5 w-5 text-gray" />
              <div>
                <p className="text-xs text-gray">Company</p>
                <p className="text-sm text-dark">{profile?.company_name}</p>
              </div>
            </div>
            <Button variant="secondary" fullWidth onClick={() => setEditing(true)}>
              {t('common.edit')} {t('profile.title')}
            </Button>
          </div>
        ) : (
          <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-4">
            <Input label="First name" value={form.first_name} onChange={(e) => { setForm((f) => ({ ...f, first_name: e.target.value })); setFormErrors((e) => ({ ...e, first_name: undefined })) }} error={formErrors.first_name} />
            <Input label="Last name" value={form.last_name} onChange={(e) => { setForm((f) => ({ ...f, last_name: e.target.value })); setFormErrors((e) => ({ ...e, last_name: undefined })) }} error={formErrors.last_name} />
            <Input label="Phone" value={form.phone} onChange={(e) => setForm((f) => ({ ...f, phone: e.target.value }))} />
            {updateMutation.isError && (
              <p className="text-sm text-danger text-center">Failed to update profile. Please try again.</p>
            )}
            <div className="flex gap-3">
              <Button variant="secondary" fullWidth onClick={() => { setEditing(false); setFormErrors({}) }}>Cancel</Button>
              <Button fullWidth onClick={() => {
                const errors: typeof formErrors = {}
                if (!form.first_name.trim()) errors.first_name = 'First name is required'
                if (!form.last_name.trim()) errors.last_name = 'Last name is required'
                if (Object.keys(errors).length > 0) { setFormErrors(errors); return }
                setFormErrors({})
                updateMutation.mutate(form)
              }} disabled={updateMutation.isPending}>
                {updateMutation.isPending ? 'Saving...' : 'Save'}
              </Button>
            </div>
          </div>
        )}

        {/* Settings */}
        <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-1">
          <p className="text-sm font-semibold text-dark mb-2">{t('profile.settings')}</p>
          <div className="flex items-center justify-between py-3">
            <span className="text-sm text-dark">{t('profile.language')}</span>
            <select
              value={locale}
              onChange={(e) => setLocale(e.target.value as Locale)}
              className="text-sm text-primary bg-transparent font-medium"
            >
              {Object.entries(LOCALE_NAMES).map(([code, name]) => (
                <option key={code} value={code}>{name}</option>
              ))}
            </select>
          </div>
          <Toggle label={t('profile.notifications')} checked={notifications} onChange={setNotifications} className="py-2" />
          <Toggle label={t('profile.faceId')} checked={faceId} onChange={setFaceId} className="py-2" />
        </div>

        {/* Quick links */}
        <div className="bg-white rounded-[16px] shadow-sm divide-y divide-bg-alt">
          {[
            { icon: Users, label: t('employees.title'), to: '/employees' },
            { icon: MapPin, label: t('locations.title'), to: '/locations' },
          ].map(({ icon: Icon, label, to }) => (
            <button key={to} onClick={() => navigate(to)} className="w-full flex items-center gap-3 px-4 py-3.5">
              <Icon className="h-5 w-5 text-gray" />
              <span className="text-sm text-dark">{label}</span>
            </button>
          ))}
        </div>

        {/* Danger zone */}
        <Button variant="danger" fullWidth onClick={handleLogout}>
          <LogOut className="h-4 w-4 mr-2" /> {t('profile.signOut')}
        </Button>

        {/* App version */}
        <p className="text-xs text-gray text-center pb-4">FoodBI v1.0.0</p>
      </main>

      <Tabbar />
    </div>
  )
}
