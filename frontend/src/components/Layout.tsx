import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, NavLink, Outlet, useNavigate, useLocation } from 'react-router-dom'
import { getAccessToken } from '../lib/auth'
import { getCurrentUser, getEnvironment, getProject, listOrgs, logout, type Org, type User } from '../lib/api'

function cn(...cls: (string | false | undefined)[]) {
  return cls.filter(Boolean).join(' ')
}

export function Layout() {
  const nav = useNavigate()
  const location = useLocation()
  const [user, setUser] = useState<User | null>(null)
  const [orgs, setOrgs] = useState<Org[]>([])
  const [orgDropdownOpen, setOrgDropdownOpen] = useState(false)
  const [userDropdownOpen, setUserDropdownOpen] = useState(false)
  const orgDropRef = useRef<HTMLDivElement>(null)
  const userDropRef = useRef<HTMLDivElement>(null)

  // Derive current org from URL
  const [currentOrgId, setCurrentOrgId] = useState<string | null>(null)

  useEffect(() => {
    const orgMatch = location.pathname.match(/\/orgs\/([^/]+)/)
    if (orgMatch) {
      setCurrentOrgId(orgMatch[1])
      return
    }
    // For project/environment pages, resolve org from API
    const projMatch = location.pathname.match(/\/projects\/([^/]+)/)
    if (projMatch) {
      getProject(projMatch[1]).then(p => {
        setCurrentOrgId(p.org_id)
      }).catch(() => {})
      return
    }
    const envMatch = location.pathname.match(/\/environments\/([^/]+)/)
    if (envMatch) {
      getEnvironment(envMatch[1]).then(e => {
        setCurrentOrgId(e.org_id)
      }).catch(() => {})
      return
    }
  }, [location.pathname])

  const currentOrg = orgs.find(o => o.id === currentOrgId)

  const loadData = useCallback(() => {
    if (!getAccessToken()) return
    getCurrentUser().then(setUser).catch(() => {})
    listOrgs().then(setOrgs).catch(() => {})
  }, [])

  useEffect(() => { loadData() }, [loadData])

  // Close dropdowns on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (orgDropRef.current && !orgDropRef.current.contains(e.target as Node)) setOrgDropdownOpen(false)
      if (userDropRef.current && !userDropRef.current.contains(e.target as Node)) setUserDropdownOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const TIER_COLORS: Record<string, string> = {
    free: 'bg-slate-100 text-slate-600',
    starter: 'bg-violet-50 text-violet-600',
    team: 'bg-purple-50 text-purple-600',
  }

  return (
    <div className="flex h-screen flex-col bg-slate-50">
      {/* Top Nav */}
      <header className="shrink-0 border-b border-slate-200 bg-white">
        <div className="mx-auto flex h-14 max-w-[1400px] items-center justify-between px-4 sm:px-6">
          {/* Left: Logo + Org Switcher */}
          <div className="flex items-center gap-4">
            <Link to="/orgs" className="flex items-center gap-2 shrink-0">
              <div className="flex h-7 w-7 items-center justify-center rounded-md bg-slate-900 text-xs font-bold text-white">E</div>
              <span className="text-lg font-bold text-slate-900 tracking-tight hidden sm:inline">Envo</span>
            </Link>

            {/* Org Switcher */}
            {orgs.length > 0 && (
              <>
                <div className="h-5 w-px bg-slate-200" />
                <div className="relative" ref={orgDropRef}>
                  <button
                    onClick={() => { setOrgDropdownOpen(!orgDropdownOpen); setUserDropdownOpen(false) }}
                    className="flex items-center gap-1.5 rounded-md px-2 py-1 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
                  >
                    {currentOrg ? (
                      <>
                        <span className="flex h-5 w-5 items-center justify-center rounded bg-violet-100 text-[10px] font-semibold text-violet-600">
                          {currentOrg.name[0]?.toUpperCase()}
                        </span>
                        <span className="max-w-[120px] truncate font-medium">{currentOrg.name}</span>
                      </>
                    ) : (
                      <span className="text-slate-500">Select org</span>
                    )}
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-slate-400">
                      <path d="M6 9l6 6 6-6" />
                    </svg>
                  </button>
                  {orgDropdownOpen && (
                    <div className="absolute left-0 top-full mt-1 w-56 rounded-lg border border-slate-200 bg-white py-1 shadow-lg z-50">
                      <div className="px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-slate-400">Organizations</div>
                      {orgs.map(o => (
                        <Link
                          key={o.id}
                          to={`/orgs/${o.id}`}
                          onClick={() => setOrgDropdownOpen(false)}
                          className={cn(
                            'flex items-center gap-2 px-3 py-2 text-sm hover:bg-slate-50 transition-colors',
                            o.id === currentOrgId ? 'bg-slate-50 font-medium text-slate-900' : 'text-slate-600',
                          )}
                        >
                          <span className="flex h-5 w-5 items-center justify-center rounded bg-violet-100 text-[10px] font-semibold text-violet-600">
                            {o.name[0]?.toUpperCase()}
                          </span>
                          <span className="truncate">{o.name}</span>
                          {o.id === currentOrgId && (
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" className="ml-auto text-violet-600 shrink-0">
                              <polyline points="20 6 9 17 4 12" />
                            </svg>
                          )}
                        </Link>
                      ))}
                      <div className="border-t border-slate-100 mt-1 pt-1">
                        <Link
                          to="/orgs"
                          onClick={() => setOrgDropdownOpen(false)}
                          className="flex items-center gap-2 px-3 py-2 text-sm text-slate-500 hover:bg-slate-50 transition-colors"
                        >
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-slate-400">
                            <line x1="12" y1="5" x2="12" y2="19" />
                            <line x1="5" y1="12" x2="19" y2="12" />
                          </svg>
                          Create organization
                        </Link>
                      </div>
                    </div>
                  )}
                </div>
              </>
            )}

            {/* Context Nav Links */}
            {currentOrgId && (
              <>
                <div className="h-5 w-px bg-slate-200 hidden sm:block" />
                <div className="hidden sm:flex items-center gap-1">
                  <NavLink
                    to={`/orgs/${currentOrgId}`}
                    end
                    className={({ isActive }) =>
                      cn(
                        'rounded-md px-2.5 py-1.5 text-sm transition-colors',
                        isActive ? 'bg-slate-100 font-medium text-slate-900' : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50',
                      )
                    }
                  >
                    Projects
                  </NavLink>
                  <NavLink
                    to={`/orgs/${currentOrgId}/members`}
                    className={({ isActive }) =>
                      cn(
                        'rounded-md px-2.5 py-1.5 text-sm transition-colors',
                        isActive ? 'bg-slate-100 font-medium text-slate-900' : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50',
                      )
                    }
                  >
                    Team
                  </NavLink>
                </div>
              </>
            )}
          </div>

          {/* Right: User Menu */}
          <div className="relative" ref={userDropRef}>
            <button
              onClick={() => { setUserDropdownOpen(!userDropdownOpen); setOrgDropdownOpen(false) }}
              className="flex items-center gap-2 rounded-md px-2 py-1 hover:bg-slate-50 transition-colors"
            >
              <div className="flex h-7 w-7 items-center justify-center rounded-full bg-violet-100 text-[11px] font-semibold text-violet-600">
                {(user?.name || user?.email || '?')[0].toUpperCase()}
              </div>
              <span className="hidden sm:inline text-sm text-slate-700 max-w-[100px] truncate">{user?.name}</span>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-slate-400">
                <path d="M6 9l6 6 6-6" />
              </svg>
            </button>
            {userDropdownOpen && (
              <div className="absolute right-0 top-full mt-1 w-56 rounded-lg border border-slate-200 bg-white py-1 shadow-lg z-50">
                {user && (
                  <div className="px-3 py-2 border-b border-slate-100">
                    <div className="text-sm font-medium text-slate-900 truncate">{user.name}</div>
                    <div className="text-xs text-slate-500 truncate">{user.email}</div>
                    <span className={cn('mt-1 inline-flex rounded-full px-2 py-0.5 text-[10px] font-medium', TIER_COLORS[user.tier] || 'bg-slate-100 text-slate-600')}>
                      {user.tier} plan
                    </span>
                  </div>
                )}
                <Link
                  to="/settings"
                  onClick={() => setUserDropdownOpen(false)}
                  className="flex items-center gap-2 px-3 py-2 text-sm text-slate-600 hover:bg-slate-50 transition-colors"
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="text-slate-400">
                    <circle cx="12" cy="12" r="3" />
                    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
                  </svg>
                  Plans & Settings
                </Link>
                <button
                  onClick={async () => {
                    setUserDropdownOpen(false)
                    await logout()
                    nav('/login')
                  }}
                  className="flex w-full items-center gap-2 px-3 py-2 text-sm text-slate-600 hover:bg-slate-50 transition-colors"
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="text-slate-400">
                    <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                    <polyline points="16 17 21 12 16 7" />
                    <line x1="21" y1="12" x2="9" y2="12" />
                  </svg>
                  Sign out
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-5xl px-4 py-6 sm:px-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
