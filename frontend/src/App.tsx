import { lazy, Suspense } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { getAccessToken } from './lib/auth'

const LandingPage = lazy(() => import('./pages/LandingPage').then(m => ({ default: m.LandingPage })))
const LoginPage = lazy(() => import('./pages/LoginPage').then(m => ({ default: m.LoginPage })))
const AuthCallbackPage = lazy(() => import('./pages/AuthCallbackPage').then(m => ({ default: m.AuthCallbackPage })))
const OrgsPage = lazy(() => import('./pages/OrgsPage').then(m => ({ default: m.OrgsPage })))
const OrgDetailPage = lazy(() => import('./pages/OrgDetailPage').then(m => ({ default: m.OrgDetailPage })))
const MembersPage = lazy(() => import('./pages/MembersPage').then(m => ({ default: m.MembersPage })))
const ProjectDetailPage = lazy(() => import('./pages/ProjectDetailPage').then(m => ({ default: m.ProjectDetailPage })))
const EnvironmentDetailPage = lazy(() => import('./pages/EnvironmentDetailPage').then(m => ({ default: m.EnvironmentDetailPage })))
const SettingsPage = lazy(() => import('./pages/SettingsPage').then(m => ({ default: m.SettingsPage })))
const PricingPage = lazy(() => import('./pages/PricingPage').then(m => ({ default: m.PricingPage })))

function PageLoader() {
  return (
    <div className="flex items-center justify-center py-20">
      <div className="h-5 w-5 animate-spin rounded-full border-2 border-slate-300 border-t-slate-600" />
    </div>
  )
}

function RequireAuth({ children }: { children: React.ReactNode }) {
  const t = getAccessToken()
  if (!t) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Suspense fallback={<PageLoader />}>
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/pricing" element={<PricingPage />} />
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
    </Suspense>
  )
}
