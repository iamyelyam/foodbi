import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { useAuthStore } from '@/stores/auth'

export function DashboardPage() {
  const { user, logout } = useAuthStore()

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="FoodBI" showNotification />

      <main className="flex-1 px-4 pt-4 pb-20">
        {/* Revenue summary card */}
        <div className="bg-white rounded-[16px] p-4 shadow-sm">
          <div className="flex items-center justify-between mb-3">
            <span className="text-sm text-gray">Total Revenue</span>
            <span className="text-xs text-primary font-medium bg-primary-lighter px-2 py-0.5 rounded-full">
              Today
            </span>
          </div>
          <p className="text-3xl font-bold text-dark">$0.00</p>
          <p className="mt-1 text-sm text-gray">Connect iiko to see real data</p>
        </div>

        {/* Purchase summary card */}
        <div className="bg-white rounded-[16px] p-4 shadow-sm mt-3">
          <div className="flex items-center justify-between mb-3">
            <span className="text-sm text-gray">Total Purchases</span>
            <span className="text-xs text-warning font-medium bg-warning/10 px-2 py-0.5 rounded-full">
              Today
            </span>
          </div>
          <p className="text-3xl font-bold text-dark">$0.00</p>
        </div>

        {/* Quick actions */}
        <div className="mt-6">
          <h2 className="text-base font-semibold text-dark mb-3">Quick Actions</h2>
          <div className="grid grid-cols-2 gap-3">
            {['Revenue', 'Purchases', 'Stock', 'Transfers'].map((action) => (
              <div
                key={action}
                className="bg-white rounded-[12px] p-4 flex items-center gap-3 shadow-sm"
              >
                <div className="w-10 h-10 rounded-full bg-primary-lighter flex items-center justify-center">
                  <div className="w-5 h-5 rounded-full bg-primary" />
                </div>
                <span className="text-sm font-medium text-dark">{action}</span>
              </div>
            ))}
          </div>
        </div>

        {user && (
          <div className="mt-6 text-center">
            <p className="text-xs text-gray">
              Logged in as {user.email} ({user.role})
            </p>
            <button onClick={logout} className="mt-2 text-sm text-danger font-medium">
              Sign out
            </button>
          </div>
        )}
      </main>

      <Tabbar />
    </div>
  )
}
