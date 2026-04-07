import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useAuthStore } from '@/stores/auth'
import { LoginPage } from '@/pages/auth/LoginPage'
import { RegisterPage } from '@/pages/auth/RegisterPage'
import { VerifyOTPPage } from '@/pages/auth/VerifyOTPPage'
import { DashboardPage } from '@/pages/DashboardPage'
import { LocationsPage } from '@/pages/locations/LocationsPage'
import { RevenuePage } from '@/pages/revenue/RevenuePage'
import { PurchasesPage } from '@/pages/purchases/PurchasesPage'
import { StatisticsPage } from '@/pages/statistics/StatisticsPage'
import { StockPage } from '@/pages/stock/StockPage'
import { SupplyingPage } from '@/pages/supplying/SupplyingPage'
import { TransfersPage } from '@/pages/transfers/TransfersPage'
import { EmployeesPage } from '@/pages/employees/EmployeesPage'
import { ProfilePage } from '@/pages/profile/ProfilePage'
import { NotificationsPage } from '@/pages/notifications/NotificationsPage'
import type { ReactNode } from 'react'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
      retry: 1,
    },
  },
})

function ProtectedRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuthStore()
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/verify-otp" element={<VerifyOTPPage />} />
          <Route path="/locations" element={<ProtectedRoute><LocationsPage /></ProtectedRoute>} />
          <Route path="/revenue" element={<ProtectedRoute><RevenuePage /></ProtectedRoute>} />
          <Route path="/purchases" element={<ProtectedRoute><PurchasesPage /></ProtectedRoute>} />
          <Route path="/statistics" element={<ProtectedRoute><StatisticsPage /></ProtectedRoute>} />
          <Route path="/stock" element={<ProtectedRoute><StockPage /></ProtectedRoute>} />
          <Route path="/supplying" element={<ProtectedRoute><SupplyingPage /></ProtectedRoute>} />
          <Route path="/transfers" element={<ProtectedRoute><TransfersPage /></ProtectedRoute>} />
          <Route path="/employees" element={<ProtectedRoute><EmployeesPage /></ProtectedRoute>} />
          <Route path="/profile" element={<ProtectedRoute><ProfilePage /></ProtectedRoute>} />
          <Route path="/notifications" element={<ProtectedRoute><NotificationsPage /></ProtectedRoute>} />
          <Route
            path="/*"
            element={
              <ProtectedRoute>
                <DashboardPage />
              </ProtectedRoute>
            }
          />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
