import { Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { getAccessToken } from './lib/auth'
import { AuthCallbackPage } from './pages/AuthCallbackPage'
import { EnvironmentDetailPage } from './pages/EnvironmentDetailPage'
import { LandingPage } from './pages/LandingPage'
import { LoginPage } from './pages/LoginPage'
import { MembersPage } from './pages/MembersPage'
import { OrgDetailPage } from './pages/OrgDetailPage'
import { OrgsPage } from './pages/OrgsPage'
import { ProjectDetailPage } from './pages/ProjectDetailPage'
import { SettingsPage } from './pages/SettingsPage'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const t = getAccessToken()
  if (!t) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<LandingPage />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/auth/callback" element={<AuthCallbackPage />} />
      <Route
        element={
          <RequireAuth>
            <Layout />
          </RequireAuth>
        }
      >
        <Route path="/orgs" element={<OrgsPage />} />
        <Route path="/orgs/:id" element={<OrgDetailPage />} />
        <Route path="/orgs/:id/members" element={<MembersPage />} />
        <Route path="/projects/:id" element={<ProjectDetailPage />} />
        <Route path="/environments/:id" element={<EnvironmentDetailPage />} />
        <Route path="/settings" element={<SettingsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
