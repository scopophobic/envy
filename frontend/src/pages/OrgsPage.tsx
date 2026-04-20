import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import { createOrg, getTierInfo, listOrgs, listOrgProjects, type Org, type TierInfo, type Project } from '../lib/api'

export function OrgsPage() {
  const nav = useNavigate()
  const [orgs, setOrgs] = useState<Org[] | null>(null)
  const [tierInfo, setTierInfo] = useState<TierInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [createName, setCreateName] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  const [vaultProjects, setVaultProjects] = useState<Project[] | null>(null)

  const maxOrgs = tierInfo?.limits.max_orgs ?? 1
  const ownedOrgs = tierInfo?.usage.owned_orgs ?? 0
  const orgLimitReached = maxOrgs !== -1 && ownedOrgs >= maxOrgs

  const personalOrg = orgs?.find(o => o.owner_type === 'personal')
  const teamOrgs = orgs?.filter(o => o.owner_type !== 'personal') ?? []

  const vaultUsage = tierInfo?.usage.orgs?.find(o => o.id === personalOrg?.id)

  const load = useCallback(() => {
    listOrgs().then((loaded) => {
      setOrgs(loaded)
      const personal = loaded.find(o => o.owner_type === 'personal')
      if (personal) {
        listOrgProjects(personal.id).then(setVaultProjects).catch(() => setVaultProjects([]))
      }
    }).catch((e) => setError((e as Error).message))
    getTierInfo().then(setTierInfo).catch(() => {})
  }, [])

  useEffect(() => { load() }, [load])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!createName.trim()) return
    setCreating(true)
    setCreateError(null)
    try {
      const org = await createOrg(createName.trim())
      setCreateName('')
      setShowCreate(false)
      load()
      nav(`/orgs/${org.id}`)
    } catch (err) {
      setCreateError((err as Error).message)
    } finally {
      setCreating(false)
    }
  }

  if (error) return <p className="text-sm text-red-600 p-6">{error}</p>
  if (!orgs) {
    return (
      <div className="space-y-8">
        {/* Vault skeleton */}
        <div className="rounded-2xl border border-slate-200 bg-white p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="h-10 w-10 animate-pulse rounded-xl bg-slate-200" />
            <div>
              <div className="h-5 w-28 animate-pulse rounded bg-slate-200" />
              <div className="mt-1.5 h-3 w-40 animate-pulse rounded bg-slate-100" />
            </div>
          </div>
          <div className="grid gap-3 sm:grid-cols-3 mt-4">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="h-16 animate-pulse rounded-lg bg-slate-100" />
            ))}
          </div>
        </div>
        {/* Team orgs skeleton */}
        <div>
          <div className="h-5 w-32 animate-pulse rounded bg-slate-200 mb-3" />
          <div className="grid gap-3 sm:grid-cols-2">
            {[...Array(2)].map((_, i) => (
              <div key={i} className="rounded-xl border border-slate-200 bg-white p-5">
                <div className="flex items-center gap-3">
                  <div className="h-9 w-9 animate-pulse rounded-lg bg-slate-200" />
                  <div className="h-4 w-28 animate-pulse rounded bg-slate-200" />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      {/* ── My Vault ─────────────────────────────────────────── */}
      {personalOrg && (
        <Link to={`/orgs/${personalOrg.id}`} className="block group">
          <div className="rounded-2xl border border-emerald-200 bg-gradient-to-br from-emerald-50/80 to-white p-6 transition-all group-hover:border-emerald-300 group-hover:shadow-md">
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-emerald-100 text-emerald-600">
                  <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                  </svg>
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900">My Vault</h2>
                  <p className="text-sm text-slate-500">Your personal secrets — encrypted, always available</p>
                </div>
              </div>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-slate-300 mt-1 group-hover:text-emerald-400 transition-colors">
                <path d="M9 18l6-6-6-6" />
              </svg>
            </div>

            {/* Quick stats */}
            <div className="mt-5 grid grid-cols-4 gap-3">
              <div className="rounded-lg bg-white/80 border border-emerald-100 px-3 py-2.5">
                <div className="text-lg font-bold text-slate-900">{vaultProjects?.length ?? '—'}</div>
                <div className="text-[11px] text-slate-500 font-medium">Projects</div>
              </div>
              <div className="rounded-lg bg-white/80 border border-emerald-100 px-3 py-2.5">
                <div className="text-lg font-bold text-slate-900">20</div>
                <div className="text-[11px] text-slate-500 font-medium">Env limit</div>
              </div>
              <div className="rounded-lg bg-white/80 border border-emerald-100 px-3 py-2.5">
                <div className="text-lg font-bold text-slate-900">{vaultUsage?.secrets ?? '—'}</div>
                <div className="text-[11px] text-slate-500 font-medium">Secrets</div>
              </div>
              <div className="rounded-lg bg-white/80 border border-emerald-100 px-3 py-2.5">
                <div className="text-lg font-bold text-slate-900">∞</div>
                <div className="text-[11px] text-slate-500 font-medium">Secrets limit / env</div>
              </div>
            </div>

            {/* Recent projects preview */}
            {vaultProjects && vaultProjects.length > 0 && (
              <div className="mt-4 flex flex-wrap gap-2">
                {vaultProjects.slice(0, 5).map(p => (
                  <span key={p.id} className="inline-flex items-center gap-1.5 rounded-full bg-white border border-emerald-100 px-2.5 py-1 text-xs font-medium text-slate-700">
                    <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                    {p.name}
                  </span>
                ))}
                {vaultProjects.length > 5 && (
                  <span className="inline-flex items-center rounded-full bg-white border border-slate-100 px-2.5 py-1 text-xs text-slate-400">
                    +{vaultProjects.length - 5} more
                  </span>
                )}
              </div>
            )}

            {vaultProjects && vaultProjects.length === 0 && (
              <p className="mt-4 text-xs text-emerald-600/70">
                Click to open your vault and create your first project.
              </p>
            )}
          </div>
        </Link>
      )}

      {/* ── Team Organizations ───────────────────────────────── */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold text-slate-900 uppercase tracking-wide">Team Organizations</h2>
            {teamOrgs.length > 0 && (
              <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-medium text-slate-500">
                {teamOrgs.length}
              </span>
            )}
          </div>
          {!orgLimitReached ? (
            <Button onClick={() => setShowCreate(!showCreate)}>
              {showCreate ? 'Cancel' : 'New team'}
            </Button>
          ) : (
            <span className="rounded-md bg-slate-100 px-3 py-1.5 text-xs text-slate-500">
              Org limit reached ({maxOrgs} max)
            </span>
          )}
        </div>

        {orgLimitReached && (
          <div className="rounded-lg bg-amber-50 border border-amber-200 px-4 py-3 text-sm text-amber-800 flex items-center justify-between mb-3">
            <span>You've reached the organization limit for your plan.</span>
            <Link to="/settings" className="text-xs font-medium text-amber-900 underline underline-offset-2 hover:no-underline">
              Upgrade plan
            </Link>
          </div>
        )}

        {showCreate && (
          <Card className="mb-3">
            <form onSubmit={handleCreate} className="flex items-end gap-3">
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs font-medium text-slate-600">Organization name</span>
                <input
                  type="text"
                  value={createName}
                  onChange={(e) => setCreateName(e.target.value)}
                  className="rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
                  placeholder="Acme Corp"
                  autoFocus
                />
              </label>
              <Button type="submit" disabled={creating || !createName.trim()}>
                {creating ? 'Creating...' : 'Create'}
              </Button>
            </form>
            {createError && <p className="mt-2 text-sm text-red-600">{createError}</p>}
          </Card>
        )}

        {teamOrgs.length === 0 && !showCreate ? (
          <Card>
            <div className="py-6 text-center">
              <div className="mx-auto mb-3 flex h-10 w-10 items-center justify-center rounded-full bg-slate-100">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="text-slate-400">
                  <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                  <circle cx="9" cy="7" r="4" />
                  <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                  <path d="M16 3.13a4 4 0 0 1 0 7.75" />
                </svg>
              </div>
              <p className="text-sm text-slate-500">
                No team organizations yet. Create one to collaborate with others.
              </p>
            </div>
          </Card>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            {teamOrgs.map((o) => (
              <Link key={o.id} to={`/orgs/${o.id}`} className="block">
                <Card className="transition-all hover:border-slate-300 hover:shadow-sm">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-violet-100 text-sm font-semibold text-violet-600">
                        {o.name[0]?.toUpperCase()}
                      </div>
                      <div>
                        <span className="text-sm font-semibold text-slate-900">{o.name}</span>
                        {o.owner && (
                          <div className="mt-0.5 text-xs text-slate-500">{o.owner.name || o.owner.email}</div>
                        )}
                      </div>
                    </div>
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-slate-400">
                      <path d="M9 18l6-6-6-6" />
                    </svg>
                  </div>
                </Card>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
