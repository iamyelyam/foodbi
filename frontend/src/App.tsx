import { useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import api from '@/lib/api'
import { LoginPage } from '@/pages/auth/LoginPage'
import { RegisterPage } from '@/pages/auth/RegisterPage'
import { RegisterEmployeePage } from '@/pages/auth/RegisterEmployeePage'
import { VerifyOTPPage } from '@/pages/auth/VerifyOTPPage'
import { OnboardingPage } from '@/pages/auth/OnboardingPage'
import { AcceptInvitePage } from '@/pages/auth/AcceptInvitePage'
import { ForgotPasswordPage } from '@/pages/auth/ForgotPasswordPage'
import { DashboardPage } from '@/pages/DashboardPage'
import { EmployeeHomePage } from '@/pages/EmployeeHomePage'
import { LocationsPage } from '@/pages/locations/LocationsPage'
import { AddLocationPage } from '@/pages/locations/AddLocationPage'
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
import { HistoryPage } from '@/pages/transfers/HistoryPage'
import { EmployeesPage } from '@/pages/employees/EmployeesPage'
import { AddEmployeePage } from '@/pages/employees/AddEmployeePage'
import { EmployeeDetailPage } from '@/pages/employees/EmployeeDetailPage'
import { ProfilePage } from '@/pages/profile/ProfilePage'
import { NotificationsPage } from '@/pages/notifications/NotificationsPage'
import { AISuggestionsPage } from '@/pages/ai-suggestions/AISuggestionsPage'
import { AISuggestionDetailPage } from '@/pages/ai-suggestions/AISuggestionDetailPage'
import { FileUploadPage } from '@/pages/file-upload/FileUploadPage'
import { EditInvoicePage } from '@/pages/file-upload/EditInvoicePage'
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

function RoleHome() {
  const { user } = useAuthStore()
  if (user?.role === 'employee') return <EmployeeHomePage />
  return <DashboardPage />
}

function AppSettingsLoader() {
  const { isAuthenticated } = useAuthStore()
  const setCompanySettings = useAppStore((s) => s.setCompanySettings)

  useEffect(() => {
    if (isAuthenticated) {
      api.get('/profile/me').then((r) => {
        if (r.data?.company_settings) {
          setCompanySettings(r.data.company_settings)
        }
      }).catch(() => {})
    }
  }, [isAuthenticated, setCompanySettings])

  return null
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ErrorBoundary>
        <AppSettingsLoader />
        <BrowserRouter>
          <Routes>
            {/* Public */}
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/register/employee" element={<RegisterEmployeePage />} />
            <Route path="/verify-otp" element={<VerifyOTPPage />} />
            <Route path="/onboarding" element={<OnboardingPage />} />
            <Route path="/accept-invite" element={<AcceptInvitePage />} />
            <Route path="/forgot-password" element={<ForgotPasswordPage />} />

            {/* Locations */}
            <Route path="/locations" element={<P><LocationsPage /></P>} />
            <Route path="/locations/new" element={<P><AddLocationPage /></P>} />

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
            <Route path="/transfers/history" element={<P><HistoryPage /></P>} />

            {/* Employees */}
            <Route path="/employees" element={<P><EmployeesPage /></P>} />
            <Route path="/employees/new" element={<P><AddEmployeePage /></P>} />
            <Route path="/employees/:id" element={<P><EmployeeDetailPage /></P>} />

            {/* Profile + Notifications */}
            <Route path="/profile" element={<P><ProfilePage /></P>} />
            <Route path="/notifications" element={<P><NotificationsPage /></P>} />

            {/* Intelligence */}
            <Route path="/ai-suggestions" element={<P><AISuggestionsPage /></P>} />
            <Route path="/ai-suggestions/:id" element={<P><AISuggestionDetailPage /></P>} />
            <Route path="/file-upload" element={<P><FileUploadPage /></P>} />
            <Route path="/file-upload/edit/:id" element={<P><EditInvoicePage /></P>} />

            {/* Default — role-based home */}
            <Route path="/*" element={<P><RoleHome /></P>} />
          </Routes>
        </BrowserRouter>
      </ErrorBoundary>
    </QueryClientProvider>
  )
}
