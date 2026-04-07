import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useAuthStore } from '@/stores/auth'
import { LoginPage } from '@/pages/auth/LoginPage'
import { RegisterPage } from '@/pages/auth/RegisterPage'
import { VerifyOTPPage } from '@/pages/auth/VerifyOTPPage'
import { OnboardingPage } from '@/pages/auth/OnboardingPage'
import { AcceptInvitePage } from '@/pages/auth/AcceptInvitePage'
import { ForgotPasswordPage } from '@/pages/auth/ForgotPasswordPage'
import { DashboardPage } from '@/pages/DashboardPage'
import { LocationsPage } from '@/pages/locations/LocationsPage'
import { RevenuePage } from '@/pages/revenue/RevenuePage'
import { OrderDetailPage } from '@/pages/revenue/OrderDetailPage'
import { ProductDetailPage } from '@/pages/revenue/ProductDetailPage'
import { PurchasesPage } from '@/pages/purchases/PurchasesPage'
import { SupplierDetailPage } from '@/pages/purchases/SupplierDetailPage'
import { StatisticsPage } from '@/pages/statistics/StatisticsPage'
import { StockPage } from '@/pages/stock/StockPage'
import { SupplyingPage } from '@/pages/supplying/SupplyingPage'
import { CreateSupplyPage } from '@/pages/supplying/CreateSupplyPage'
import { TransfersPage } from '@/pages/transfers/TransfersPage'
import { CreateTransferPage } from '@/pages/transfers/CreateTransferPage'
import { EmployeesPage } from '@/pages/employees/EmployeesPage'
import { AddEmployeePage } from '@/pages/employees/AddEmployeePage'
import { EmployeeDetailPage } from '@/pages/employees/EmployeeDetailPage'
import { ProfilePage } from '@/pages/profile/ProfilePage'
import { NotificationsPage } from '@/pages/notifications/NotificationsPage'
import { AISuggestionsPage } from '@/pages/ai-suggestions/AISuggestionsPage'
import { FileUploadPage } from '@/pages/file-upload/FileUploadPage'
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

function P({ children }: { children: ReactNode }) {
  return <ProtectedRoute>{children}</ProtectedRoute>
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          {/* Public */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/verify-otp" element={<VerifyOTPPage />} />
          <Route path="/onboarding" element={<OnboardingPage />} />
          <Route path="/accept-invite" element={<AcceptInvitePage />} />
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />

          {/* Dashboard */}
          <Route path="/locations" element={<P><LocationsPage /></P>} />

          {/* Revenue */}
          <Route path="/revenue" element={<P><RevenuePage /></P>} />
          <Route path="/revenue/orders/:id" element={<P><OrderDetailPage /></P>} />
          <Route path="/revenue/products/:id" element={<P><ProductDetailPage /></P>} />

          {/* Purchases */}
          <Route path="/purchases" element={<P><PurchasesPage /></P>} />
          <Route path="/purchases/suppliers/:id" element={<P><SupplierDetailPage /></P>} />

          {/* Statistics */}
          <Route path="/statistics" element={<P><StatisticsPage /></P>} />

          {/* Stock */}
          <Route path="/stock" element={<P><StockPage /></P>} />

          {/* Supplying */}
          <Route path="/supplying" element={<P><SupplyingPage /></P>} />
          <Route path="/supplying/new" element={<P><CreateSupplyPage /></P>} />

          {/* Transfers */}
          <Route path="/transfers" element={<P><TransfersPage /></P>} />
          <Route path="/transfers/new" element={<P><CreateTransferPage /></P>} />

          {/* Employees */}
          <Route path="/employees" element={<P><EmployeesPage /></P>} />
          <Route path="/employees/new" element={<P><AddEmployeePage /></P>} />
          <Route path="/employees/:id" element={<P><EmployeeDetailPage /></P>} />

          {/* Profile + Notifications */}
          <Route path="/profile" element={<P><ProfilePage /></P>} />
          <Route path="/notifications" element={<P><NotificationsPage /></P>} />

          {/* Intelligence */}
          <Route path="/ai-suggestions" element={<P><AISuggestionsPage /></P>} />
          <Route path="/file-upload" element={<P><FileUploadPage /></P>} />

          {/* Default */}
          <Route path="/*" element={<P><DashboardPage /></P>} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
