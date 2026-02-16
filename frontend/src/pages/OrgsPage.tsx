import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import { createOrg, getTierInfo, listOrgs, type Org, type TierInfo } from '../lib/api'

export function OrgsPage() {
  const nav = useNavigate()
  const [orgs, setOrgs] = useState<Org[] | null>(null)
  const [tierInfo, setTierInfo] = useState<TierInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [createName, setCreateName] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const maxOrgs = tierInfo?.limits.max_orgs ?? 1
  const ownedOrgs = tierInfo?.usage.owned_orgs ?? 0
  const orgLimitReached = maxOrgs !== -1 && ownedOrgs >= maxOrgs

  const load = useCallback(() => {
    listOrgs().then(setOrgs).catch((e) => setError((e as Error).message))
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
      <div className="flex items-center justify-center py-16">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-slate-300 border-t-slate-600" />
        <span className="ml-2 text-sm text-slate-500">Loading...</span>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Organizations</h1>
          <p className="mt-1 text-sm text-slate-500">
            Manage your organizations, projects, and team.
          </p>
        </div>
      </div>

      {orgLimitReached && (
        <div className="rounded-lg bg-amber-50 border border-amber-200 px-4 py-3 text-sm text-amber-800 flex items-center justify-between">
          <span>You've reached the organization limit for your plan.</span>
          <Link to="/settings" className="text-xs font-medium text-amber-900 underline underline-offset-2 hover:no-underline">
            Upgrade plan
          </Link>
        </div>
      )}

      {orgs.length === 0 && !showCreate && (
        <Card>
          <div className="py-10 text-center">
            <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-slate-100">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="text-slate-400">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                <path d="M16 3.13a4 4 0 0 1 0 7.75" />
              </svg>
            </div>
            <h2 className="text-lg font-semibold text-slate-900">Welcome to Envo</h2>
            <p className="mt-1 text-sm text-slate-500">
              Create your first organization to start managing secrets securely.
            </p>
            <div className="mt-4">
              <Button onClick={() => setShowCreate(true)}>Create organization</Button>
            </div>
          </div>
        </Card>
      )}

      {orgs.length > 0 && (
        <div className="flex items-center justify-between">
          <span className="text-sm text-slate-500">
            {orgs.length} organization{orgs.length !== 1 ? 's' : ''}
          </span>
          {!orgLimitReached ? (
            <Button onClick={() => setShowCreate(!showCreate)}>
              {showCreate ? 'Cancel' : 'Create organization'}
            </Button>
          ) : (
            <span className="rounded-md bg-slate-100 px-3 py-1.5 text-xs text-slate-500">
              Org limit reached ({maxOrgs} max)
            </span>
          )}
        </div>
      )}

      {showCreate && (
        <Card>
          <form onSubmit={handleCreate} className="flex items-end gap-3">
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-slate-600">Organization name</span>
              <input
                type="text"
                value={createName}
                onChange={(e) => setCreateName(e.target.value)}
                className="rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
                placeholder="My Team"
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

      <div className="grid gap-3 sm:grid-cols-2">
        {orgs.map((o) => (
          <Link key={o.id} to={`/orgs/${o.id}`} className="block">
            <Card className="transition-all hover:border-slate-300 hover:shadow-sm">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-violet-100 text-sm font-semibold text-violet-600">
                    {o.name[0]?.toUpperCase()}
                  </div>
                  <div>
                    <div className="text-sm font-semibold text-slate-900">{o.name}</div>
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
    </div>
  )
}
