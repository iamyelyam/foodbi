import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuthStore } from '@/stores/auth'
import { User, Mail, Phone, Building2, LogOut, Users, MapPin } from 'lucide-react'
import api from '@/lib/api'

export function ProfilePage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { logout } = useAuthStore()
  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState({ first_name: '', last_name: '', phone: '' })

  const { data: profile } = useQuery({
    queryKey: ['profile'],
    queryFn: () => api.get('/profile/me').then((r) => r.data),
    select: (data) => {
      if (!editing) setForm({ first_name: data.first_name, last_name: data.last_name, phone: data.phone })
      return data
    },
  })

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
      <Header title="Profile" showBack />

      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {/* Avatar + name */}
        <div className="bg-white rounded-[16px] p-6 shadow-sm flex flex-col items-center">
          <div className="w-20 h-20 rounded-full bg-primary-lighter flex items-center justify-center">
            <User className="h-10 w-10 text-primary" />
          </div>
          <p className="mt-3 text-lg font-bold text-dark">
            {profile?.first_name} {profile?.last_name}
          </p>
          <span className="text-xs text-gray capitalize">{profile?.role}</span>
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
              Edit Profile
            </Button>
          </div>
        ) : (
          <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-4">
            <Input label="First name" value={form.first_name} onChange={(e) => setForm((f) => ({ ...f, first_name: e.target.value }))} />
            <Input label="Last name" value={form.last_name} onChange={(e) => setForm((f) => ({ ...f, last_name: e.target.value }))} />
            <Input label="Phone" value={form.phone} onChange={(e) => setForm((f) => ({ ...f, phone: e.target.value }))} />
            <div className="flex gap-3">
              <Button variant="secondary" fullWidth onClick={() => setEditing(false)}>Cancel</Button>
              <Button fullWidth onClick={() => updateMutation.mutate(form)} disabled={updateMutation.isPending}>Save</Button>
            </div>
          </div>
        )}

        {/* Quick links */}
        <div className="bg-white rounded-[16px] shadow-sm divide-y divide-bg-alt">
          {[
            { icon: Users, label: 'Employees', to: '/employees' },
            { icon: MapPin, label: 'Locations', to: '/locations' },
          ].map(({ icon: Icon, label, to }) => (
            <button key={to} onClick={() => navigate(to)} className="w-full flex items-center gap-3 px-4 py-3.5">
              <Icon className="h-5 w-5 text-gray" />
              <span className="text-sm text-dark">{label}</span>
            </button>
          ))}
        </div>

        <Button variant="danger" fullWidth onClick={handleLogout}>
          <LogOut className="h-4 w-4 mr-2" /> Sign Out
        </Button>
      </main>

      <Tabbar />
    </div>
  )
}
