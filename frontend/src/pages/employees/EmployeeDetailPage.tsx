import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { User, Mail, Phone, Shield, MapPin } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'

export function EmployeeDetailPage() {
  const { id } = useParams()

  const { data: emp, isLoading } = useQuery({
    queryKey: ['employee', id],
    queryFn: () => api.get(`/employees/${id}`).then((r) => r.data),
    enabled: !!id,
  })

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={emp ? `${emp.first_name} ${emp.last_name}` : 'Employee'} showBack />
      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {isLoading ? <><ListItemSkeleton /><ListItemSkeleton /></> : emp ? (
          <>
            <div className="bg-white rounded-[16px] p-6 shadow-sm flex flex-col items-center">
              <div className={cn('w-20 h-20 rounded-full flex items-center justify-center',
                emp.role === 'owner' ? 'bg-warning/10' : 'bg-primary-lighter')}>
                <User className={cn('h-10 w-10', emp.role === 'owner' ? 'text-warning' : 'text-primary')} />
              </div>
              <p className="mt-3 text-lg font-bold text-dark">{emp.first_name} {emp.last_name}</p>
              <span className={cn('text-xs px-3 py-1 rounded-full font-medium capitalize mt-1',
                emp.role === 'owner' ? 'bg-warning/10 text-warning' : 'bg-primary-lighter text-primary')}>
                {emp.role}
              </span>
            </div>

            <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-4">
              <div className="flex items-center gap-3">
                <Mail className="h-5 w-5 text-gray" />
                <div><p className="text-xs text-gray">Email</p><p className="text-sm text-dark">{emp.email}</p></div>
              </div>
              <div className="flex items-center gap-3">
                <Phone className="h-5 w-5 text-gray" />
                <div><p className="text-xs text-gray">Phone</p><p className="text-sm text-dark">{emp.phone || 'Not set'}</p></div>
              </div>
              <div className="flex items-center gap-3">
                <Shield className="h-5 w-5 text-gray" />
                <div><p className="text-xs text-gray">Status</p><p className="text-sm text-dark">{emp.is_active ? 'Active' : 'Inactive'}</p></div>
              </div>
            </div>

            {emp.locations && emp.locations.length > 0 && (
              <div className="bg-white rounded-[16px] p-4 shadow-sm">
                <p className="text-sm font-semibold text-dark mb-3">Assigned Locations</p>
                <div className="space-y-2">
                  {emp.locations.map((loc: string, i: number) => (
                    <div key={i} className="flex items-center gap-2">
                      <MapPin className="h-4 w-4 text-primary" />
                      <span className="text-sm text-dark">{loc}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </>
        ) : (
          <p className="text-center text-sm text-gray py-12">Employee not found</p>
        )}
      </main>
      <Tabbar />
    </div>
  )
}
