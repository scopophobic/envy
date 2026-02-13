import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import { createOrg, getCurrentUser, listOrgs, type Org, type User } from '../lib/api'

export function OrgsPage() {
  const nav = useNavigate()
  const [orgs, setOrgs] = useState<Org[] | null>(null)
  const [user, setUser] = useState<User | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [createName, setCreateName] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const tierLimits: Record<string, number> = { free: 1, starter: 1, team: -1 }
  const maxOrgs = user ? (tierLimits[user.tier] ?? 1) : 1
  const orgLimitReached = maxOrgs !== -1 && orgs !== null && orgs.length >= maxOrgs

  const load = useCallback(() => {
    listOrgs().then(setOrgs).catch((e) => setError((e as Error).message))
    getCurrentUser().then(setUser).catch(() => {})
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

  if (error) return <p className="text-sm text-red-600">{error}</p>
  if (!orgs) return <p className="text-sm text-slate-500">Loading...</p>

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Organizations</h1>
        <p className="mt-1 text-sm text-slate-500">
          Manage your organizations, projects, and team.
          {user && <span className="ml-1">({user.tier} tier)</span>}
        </p>
      </div>

      {orgs.length === 0 && !showCreate && (
        <Card>
          <div className="py-8 text-center">
            <h2 className="text-lg font-semibold text-slate-900">Welcome to Envo</h2>
            <p className="mt-2 text-sm text-slate-500">
              Create your first organization to start managing secrets.
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
            {orgs.length} org{orgs.length !== 1 ? 's' : ''}
            {maxOrgs !== -1 && ` / ${maxOrgs} max`}
          </span>
          {!orgLimitReached && (
            <Button onClick={() => setShowCreate(!showCreate)}>
              {showCreate ? 'Cancel' : 'Create organization'}
            </Button>
          )}
          {orgLimitReached && (
            <span className="text-xs text-slate-400">Org limit reached for your tier</span>
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
            <Card className="transition-colors hover:border-slate-300">
              <div className="text-sm font-semibold text-slate-900">{o.name}</div>
              {o.owner && (
                <div className="mt-1 text-xs text-slate-500">Owner: {o.owner.name || o.owner.email}</div>
              )}
            </Card>
          </Link>
        ))}
      </div>
    </div>
  )
}
