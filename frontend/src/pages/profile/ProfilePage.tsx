import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Toggle } from '@/components/ui/toggle'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { User, Mail, Phone, Building2, LogOut, Users, MapPin, Pencil } from 'lucide-react'
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
  const showUploadInvoicesBanner = useAppStore((s) => s.uiPrefs.showUploadInvoicesBanner)
  const setUiPref = useAppStore((s) => s.setUiPref)

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

      <main className="flex-1 px-4 pt-4 pb-28 space-y-3">
        {/* Compact profile header — avatar + name + phone + edit */}
        <div className="bg-white rounded-[16px] p-4 shadow-sm">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 rounded-full bg-primary-lighter flex items-center justify-center shrink-0">
              <User className="h-6 w-6 text-primary" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-base font-bold text-dark truncate">
                {profile?.first_name} {profile?.last_name}
              </p>
              <p className="text-sm text-gray truncate">
                {profile?.phone || profile?.email || t('common.notSet')}
              </p>
            </div>
            <button
              onClick={() => setEditing(true)}
              className="w-9 h-9 rounded-full bg-bg flex items-center justify-center shrink-0 active:opacity-70"
              aria-label="Edit"
            >
              <Pencil className="h-4 w-4 text-gray" />
            </button>
          </div>
        </div>

        {/* Info or edit form */}
        {!editing ? (
          <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-4">
            <div className="flex items-center gap-3">
              <Mail className="h-5 w-5 text-gray" />
              <div>
                <p className="text-xs text-gray">{t('common.email')}</p>
                <p className="text-sm text-dark">{profile?.email}</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Phone className="h-5 w-5 text-gray" />
              <div>
                <p className="text-xs text-gray">{t('common.phone')}</p>
                <p className="text-sm text-dark">{profile?.phone || t('common.notSet')}</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Building2 className="h-5 w-5 text-gray" />
              <div>
                <p className="text-xs text-gray">{t('profile.company')}</p>
                <p className="text-sm text-dark">{profile?.company_name}</p>
              </div>
            </div>
          </div>
        ) : (
          <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-4">
            <Input label={t('common.firstName')} value={form.first_name} onChange={(e) => { setForm((f) => ({ ...f, first_name: e.target.value })); setFormErrors((e) => ({ ...e, first_name: undefined })) }} error={formErrors.first_name} />
            <Input label={t('common.lastName')} value={form.last_name} onChange={(e) => { setForm((f) => ({ ...f, last_name: e.target.value })); setFormErrors((e) => ({ ...e, last_name: undefined })) }} error={formErrors.last_name} />
            <Input label={t('common.phone')} value={form.phone} onChange={(e) => setForm((f) => ({ ...f, phone: e.target.value }))} />
            {updateMutation.isError && (
              <p className="text-sm text-danger text-center">{t('profile.updateFailed')}</p>
            )}
            <div className="flex gap-3">
              <Button variant="secondary" fullWidth onClick={() => { setEditing(false); setFormErrors({}) }}>{t('common.cancel')}</Button>
              <Button fullWidth onClick={() => {
                const errors: typeof formErrors = {}
                if (!form.first_name.trim()) errors.first_name = t('profile.firstNameRequired')
                if (!form.last_name.trim()) errors.last_name = t('profile.lastNameRequired')
                if (Object.keys(errors).length > 0) { setFormErrors(errors); return }
                setFormErrors({})
                updateMutation.mutate(form)
              }} disabled={updateMutation.isPending}>
                {updateMutation.isPending ? t('common.saving') : t('common.save')}
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
          <Toggle
            label={t('profile.uploadInvoicesBanner')}
            checked={showUploadInvoicesBanner}
            onChange={(v) => setUiPref('showUploadInvoicesBanner', v)}
            className="py-2"
          />
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
        <p className="text-xs text-gray text-center pb-4">{t('profile.version', { version: '1.0.0' })}</p>
      </main>

      <Tabbar />
    </div>
  )
}
