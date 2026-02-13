import { useCallback, useEffect, useState } from 'react'
import { Link, NavLink, Outlet, useNavigate, useParams, useLocation } from 'react-router-dom'
import { getAccessToken } from '../lib/auth'
import { getCurrentUser, listOrgs, logout, type Org, type User } from '../lib/api'

function cn(...cls: (string | false | undefined)[]) {
  return cls.filter(Boolean).join(' ')
}

export function Layout() {
  const nav = useNavigate()
  const location = useLocation()
  const params = useParams()
  const [user, setUser] = useState<User | null>(null)
  const [orgs, setOrgs] = useState<Org[]>([])
  const [sidebarOpen, setSidebarOpen] = useState(true)

  const loadData = useCallback(() => {
    if (!getAccessToken()) return
    getCurrentUser().then(setUser).catch(() => {})
    listOrgs().then(setOrgs).catch(() => {})
  }, [])

  useEffect(() => {
    loadData()
  }, [loadData, location.pathname])

  const currentOrgId = params.id && location.pathname.startsWith('/orgs/') ? params.id : undefined

  return (
    <div className="flex h-screen bg-slate-50">
      {/* Sidebar */}
      <aside
        className={cn(
          'flex flex-col border-r border-slate-200 bg-white transition-all duration-200',
          sidebarOpen ? 'w-64' : 'w-0 overflow-hidden',
        )}
      >
        {/* Brand */}
        <div className="flex h-14 items-center justify-between border-b border-slate-200 px-4">
          <Link to="/orgs" className="text-lg font-bold text-slate-900">
            Envo
          </Link>
          <button
            onClick={() => setSidebarOpen(false)}
            className="text-slate-400 hover:text-slate-600"
            title="Collapse sidebar"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M15 18l-6-6 6-6" />
            </svg>
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto p-3">
          <div className="mb-2 text-[10px] font-semibold uppercase tracking-wider text-slate-400">
            Organizations
          </div>
          {orgs.length === 0 ? (
            <p className="px-2 text-xs text-slate-400">No orgs yet</p>
          ) : (
            <ul className="space-y-0.5">
              {orgs.map((o) => (
                <li key={o.id}>
                  <NavLink
                    to={`/orgs/${o.id}`}
                    className={({ isActive }) =>
                      cn(
                        'block rounded-md px-2.5 py-1.5 text-sm',
                        isActive
                          ? 'bg-slate-100 font-medium text-slate-900'
                          : 'text-slate-600 hover:bg-slate-50 hover:text-slate-900',
                      )
                    }
                  >
                    {o.name}
                  </NavLink>
                </li>
              ))}
            </ul>
          )}

          <Link
            to="/orgs"
            className="mt-2 block rounded-md px-2.5 py-1.5 text-sm text-slate-500 hover:bg-slate-50 hover:text-slate-900"
          >
            + Create organization
          </Link>

          {currentOrgId && (
            <div className="mt-4 border-t border-slate-100 pt-3">
              <NavLink
                to={`/orgs/${currentOrgId}`}
                end
                className={({ isActive }) =>
                  cn(
                    'block rounded-md px-2.5 py-1.5 text-xs',
                    isActive
                      ? 'bg-slate-100 font-medium text-slate-900'
                      : 'text-slate-500 hover:bg-slate-50 hover:text-slate-900',
                  )
                }
              >
                Projects
              </NavLink>
              <NavLink
                to={`/orgs/${currentOrgId}/members`}
                className={({ isActive }) =>
                  cn(
                    'block rounded-md px-2.5 py-1.5 text-xs',
                    isActive
                      ? 'bg-slate-100 font-medium text-slate-900'
                      : 'text-slate-500 hover:bg-slate-50 hover:text-slate-900',
                  )
                }
              >
                Team Members
              </NavLink>
            </div>
          )}
        </nav>

        {/* User footer */}
        <div className="border-t border-slate-200 p-3">
          {user && (
            <div className="mb-2">
              <div className="truncate text-sm font-medium text-slate-900">{user.name}</div>
              <div className="truncate text-xs text-slate-500">{user.email}</div>
              <div className="mt-0.5 text-[10px] uppercase text-slate-400">{user.tier} tier</div>
            </div>
          )}
          <button
            className="w-full rounded-md px-2.5 py-1.5 text-left text-sm text-slate-600 hover:bg-slate-50 hover:text-slate-900"
            onClick={async () => {
              await logout()
              nav('/login')
            }}
          >
            Sign out
          </button>
        </div>
      </aside>

      {/* Main */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Top bar (when sidebar is collapsed) */}
        {!sidebarOpen && (
          <div className="flex h-14 items-center border-b border-slate-200 bg-white px-4">
            <button
              onClick={() => setSidebarOpen(true)}
              className="mr-3 text-slate-400 hover:text-slate-600"
              title="Open sidebar"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M3 12h18M3 6h18M3 18h18" />
              </svg>
            </button>
            <Link to="/orgs" className="text-lg font-bold text-slate-900">
              Envo
            </Link>
          </div>
        )}
        <main className="flex-1 overflow-y-auto p-6">
          <div className="mx-auto max-w-5xl">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}
