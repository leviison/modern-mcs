import { Navigate, Route, Routes } from 'react-router-dom'
import { NavBar } from './components/NavBar'
import { RequireAuth } from './components/RequireAuth'
import { useAuth } from './context/AuthContext'
import { LoginPage } from './pages/LoginPage'
import { SQLProfilesPage } from './pages/SQLProfilesPage'
import { SessionsPage } from './pages/SessionsPage'
import { MigrationsPage } from './pages/MigrationsPage'
import { ChangePasswordPage } from './pages/ChangePasswordPage'

function ProtectedLayout({ children }: { children: React.ReactNode }) {
  return (
    <RequireAuth>
      <NavBar />
      {children}
    </RequireAuth>
  )
}

export default function App() {
  const auth = useAuth()

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/sql-profiles"
        element={
          <ProtectedLayout>
            <SQLProfilesPage />
          </ProtectedLayout>
        }
      />
      <Route
        path="/sessions"
        element={
          <ProtectedLayout>
            <SessionsPage />
          </ProtectedLayout>
        }
      />
      <Route
        path="/migrations"
        element={
          <ProtectedLayout>
            <MigrationsPage />
          </ProtectedLayout>
        }
      />
      <Route
        path="/change-password"
        element={
          <ProtectedLayout>
            <ChangePasswordPage />
          </ProtectedLayout>
        }
      />
      <Route path="*" element={<Navigate to={auth.isAuthenticated ? '/sql-profiles' : '/login'} replace />} />
    </Routes>
  )
}
