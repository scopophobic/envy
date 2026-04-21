import { useCallback, useEffect, useState } from 'react'
import { Card } from '../components/Card'
import {
  createPlatformConnection,
  createCheckoutSession,
  deletePlatformConnection,
  createPortalSession,
  getBillingStatus,
  getCurrentUser,
  getTierInfo,
  listOrgs,
  type Org,
  listPlatformConnections,
  type PlatformConnection,
  type BillingStatus,
  type TierInfo,
  type User,
} from '../lib/api'
import { SUBSCRIPTION_PLANS } from '../lib/pricing'

type Tab = 'plans' | 'deployment' | 'account'

const WORKSPACE_POLICIES = [
  {
    id: 'vault',
    name: 'My Vault (Personal)',
    accent: 'emerald',
    description: 'Solo workspace optimized for personal secret storage.',
    limits: {
      projects: '10',
      members: 'No team members',
      environments: '20',
      secrets: 'Unlimited',
    },
  },
  {
    id: 'org',
    name: 'Organization (Team)',
    accent: 'violet',
    description: 'Team workspace with tighter collaboration limits.',
    limits: {
      projects: '2',
      members: '2',
      environments: '10',
      secrets: 'Unlimited',
    },
  },
]

function Skeleton({ className = '' }: { className?: string }) {
  return <div className={`animate-pulse rounded bg-slate-200 ${className}`} />
}

function UsageBar({ current, max, label }: { current: number; max: number; label: string }) {
  const isUnlimited = max === -1
  const pct = isUnlimited ? 10 : max === 0 ? 0 : Math.min((current / max) * 100, 100)
  const atLimit = !isUnlimited && max > 0 && current >= max
  return (
    <div>
      <div className="flex items-center justify-between text-xs">
        <span className="text-slate-600">{label}</span>
        <span className={atLimit ? 'font-semibold text-red-600' : 'text-slate-500'}>
          {current} / {isUnlimited ? '∞' : max}
        </span>
      </div>
      <div className="mt-1 h-1.5 w-full rounded-full bg-slate-100">
        <div
          className={`h-1.5 rounded-full transition-all ${atLimit ? 'bg-red-500' : 'bg-violet-600'}`}
          style={{ width: `${Math.max(isUnlimited ? 8 : pct, 2)}%` }}
        />
      </div>
    </div>
  )
}

export function SettingsPage() {
  const [tab, setTab] = useState<Tab>('plans')
  const [user, setUser] = useState<User | null>(null)
  const [tierInfo, setTierInfo] = useState<TierInfo | null>(null)
  const [orgs, setOrgs] = useState<Org[]>([])
  const [connections, setConnections] = useState<PlatformConnection[]>([])
  const [upgrading, setUpgrading] = useState<string | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [connectionError, setConnectionError] = useState<string | null>(null)
  const [savingConnection, setSavingConnection] = useState(false)
  const [platform, setPlatform] = useState('vercel')
  const [name, setName] = useState('')
  const [token, setToken] = useState('')
  const [billingStatus, setBillingStatus] = useState<BillingStatus | null>(null)

  const load = useCallback(() => {
    setLoadError(null)
    setLoading(true)
    Promise.allSettled([
      getCurrentUser(),
      getTierInfo(),
      listOrgs(),
      listPlatformConnections(),
      getBillingStatus(),
    ])
      .then((results) => {
        const [u, t, o, c, b] = results
        if (u.status === 'fulfilled') setUser(u.value)
        else setLoadError((u.reason as Error)?.message ?? 'Failed to load profile')
        if (t.status === 'fulfilled') setTierInfo(t.value)
        if (o.status === 'fulfilled') setOrgs(o.value)
        if (c.status === 'fulfilled') setConnections(c.value)
        else setConnections([])
        if (b.status === 'fulfilled') setBillingStatus(b.value)
        else setBillingStatus(null)
      })
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => { load() }, [load])

  const handleUpgrade = async (plan: string) => {
    setUpgrading(plan)
    try {
      const url = await createCheckoutSession(plan)
      window.location.href = url
    } catch (err) {
      alert((err as Error).message)
    } finally {
      setUpgrading(null)
    }
  }

  const handleManageBilling = async () => {
    try {
      const url = await createPortalSession()
      window.location.href = url
    } catch (err) {
      alert((err as Error).message)
    }
  }

  const handleCreateConnection = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token.trim()) return
    setConnectionError(null)
    setSavingConnection(true)
    try {
      await createPlatformConnection({
        platform,
        name: name.trim() || undefined,
        token: token.trim(),
      })
      setToken('')
      setName('')
      setConnections(await listPlatformConnections())
    } catch (err) {
      setConnectionError((err as Error).message)
    } finally {
      setSavingConnection(false)
    }
  }

  const handleDeleteConnection = async (id: string, connName: string) => {
    if (!confirm(`Remove deployment connection "${connName}"?`)) return
    setConnectionError(null)
    try {
      await deletePlatformConnection(id)
      setConnections((prev) => prev.filter((c) => c.id !== id))
    } catch (err) {
      setConnectionError((err as Error).message)
    }
  }

  const currentTier = user?.tier || 'free'
  const checkoutOk = billingStatus?.checkout_enabled === true
  const starterReady = billingStatus?.starter_plan_ready === true
  const teamReady = billingStatus?.team_plan_ready === true

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Settings</h1>
        <p className="mt-1 text-sm text-slate-500">Manage your plan, billing, and account.</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-slate-200">
        <button
          onClick={() => setTab('plans')}
          className={`px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px ${
            tab === 'plans'
              ? 'border-violet-600 text-violet-700'
              : 'border-transparent text-slate-500 hover:text-slate-700'
          }`}
        >
          Plans & Billing
        </button>
        <button
          onClick={() => setTab('deployment')}
          className={`px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px ${
            tab === 'deployment'
              ? 'border-violet-600 text-violet-700'
              : 'border-transparent text-slate-500 hover:text-slate-700'
          }`}
        >
          Deployment
        </button>
        <button
          onClick={() => setTab('account')}
          className={`px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px ${
            tab === 'account'
              ? 'border-violet-600 text-violet-700'
              : 'border-transparent text-slate-500 hover:text-slate-700'
          }`}
        >
          Account
        </button>
      </div>

      {tab === 'plans' && (
        <div className="space-y-8">
          {/* Subscription pricing */}
          <div>
            <h2 className="text-lg font-semibold text-slate-900">Plans &amp; pricing</h2>
            <p className="mt-1 text-xs text-slate-500">
              Prices below are for display; the amount you pay is confirmed on Razorpay. Backend must have{' '}
              <span className="font-mono">RAZORPAY_PLAN_STARTER</span> and{' '}
              <span className="font-mono">RAZORPAY_PLAN_TEAM</span> set to your dashboard plan IDs.
            </p>
            {billingStatus && !checkoutOk && (
              <div className="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
                <strong className="font-medium">Checkout unavailable.</strong>{' '}
                {billingStatus.message ?? 'Configure RAZORPAY_KEY_ID and RAZORPAY_KEY_SECRET on the API server and restart.'}
              </div>
            )}
            {checkoutOk && (!starterReady || !teamReady) && (
              <div className="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
                Missing plan ID{!starterReady && !teamReady ? 's' : ''} in the API env:{' '}
                {!starterReady && <span className="font-mono">RAZORPAY_PLAN_STARTER</span>}
                {!starterReady && !teamReady && ' and '}
                {!teamReady && <span className="font-mono">RAZORPAY_PLAN_TEAM</span>}.
              </div>
            )}
            <div className="mt-4 grid gap-4 lg:grid-cols-3">
              {SUBSCRIPTION_PLANS.map((plan) => (
                <div
                  key={plan.id}
                  className={`rounded-xl border p-5 ${
                    plan.id === 'free'
                      ? 'border-slate-200 bg-white'
                      : plan.id === 'starter'
                        ? 'border-violet-200 bg-violet-50/50'
                        : 'border-purple-200 bg-purple-50/40'
                  }`}
                >
                  <div className="flex items-baseline justify-between gap-2">
                    <h3 className="text-base font-semibold text-slate-900">{plan.name}</h3>
                    <span className="text-sm font-semibold text-slate-800">{plan.priceLine}</span>
                  </div>
                  <p className="mt-2 text-xs text-slate-600">{plan.detail}</p>
                  <ul className="mt-3 space-y-1.5 text-xs text-slate-700">
                    {plan.highlights.map((h) => (
                      <li key={h} className="flex gap-2">
                        <span className="text-emerald-600">✓</span>
                        <span>{h}</span>
                      </li>
                    ))}
                  </ul>
                  {plan.id === 'starter' && currentTier === 'free' && (
                    <button
                      type="button"
                      onClick={() => handleUpgrade('starter')}
                      disabled={upgrading === 'starter' || !checkoutOk || !starterReady}
                      className="mt-4 w-full rounded-lg border border-violet-300 bg-white py-2 text-sm font-medium text-violet-800 hover:bg-violet-50 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {upgrading === 'starter' ? 'Redirecting…' : 'Subscribe to Starter'}
                    </button>
                  )}
                  {plan.id === 'team' && (currentTier === 'free' || currentTier === 'starter') && (
                    <button
                      type="button"
                      onClick={() => handleUpgrade('team')}
                      disabled={upgrading === 'team' || !checkoutOk || !teamReady}
                      className="mt-4 w-full rounded-lg bg-violet-600 py-2 text-sm font-medium text-white hover:bg-violet-700 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {upgrading === 'team' ? 'Redirecting…' : 'Subscribe to Team'}
                    </button>
                  )}
                  {plan.id === 'free' && currentTier === 'free' && (
                    <p className="mt-4 text-center text-xs font-medium text-slate-500">Current plan</p>
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Workspace policy cards */}
          <div className="grid gap-4 sm:grid-cols-2">
            {WORKSPACE_POLICIES.map((policy) => (
              <div
                key={policy.id}
                className={`rounded-xl border p-5 ${
                  policy.accent === 'emerald'
                    ? 'border-emerald-200 bg-emerald-50/40'
                    : 'border-violet-200 bg-violet-50/40'
                }`}
              >
                <h3 className="text-lg font-semibold text-slate-900">{policy.name}</h3>
                <p className="mt-1 text-xs text-slate-500">{policy.description}</p>
                <ul className="mt-4 space-y-2">
                  {Object.entries(policy.limits).map(([key, val]) => (
                    <li key={key} className="flex items-center gap-2 text-xs text-slate-700">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className={policy.accent === 'emerald' ? 'text-emerald-600' : 'text-violet-600'}>
                        <polyline points="20 6 9 17 4 12" />
                      </svg>
                      <span className="capitalize">
                        <span className="font-medium">{val}</span> {key}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>

          <Card>
            <div className="flex flex-col gap-4 sm:flex-row sm:flex-wrap sm:items-start sm:justify-between">
              <div>
                <h3 className="text-sm font-semibold text-slate-900">Billing</h3>
                <p className="mt-1 text-xs text-slate-500">
                  Current subscription tier: <span className="font-medium capitalize">{currentTier}</span>
                </p>
                <p className="mt-2 text-xs text-slate-500">
                  API: <span className="font-mono text-slate-600">{import.meta.env.VITE_API_URL || 'http://localhost:8080 (default)'}</span>
                  {' — '}
                  must be your Go server (e.g. :8080), not the Vite dev port (:5173).
                </p>
                <p className="mt-1 text-xs text-slate-500">Use the plan cards above to start a subscription.</p>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {currentTier !== 'free' && (
                  <button
                    type="button"
                    onClick={() => void handleManageBilling()}
                    disabled={!checkoutOk}
                    className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    Manage billing
                  </button>
                )}
              </div>
            </div>
          </Card>

          {/* Per-org usage */}
          {tierInfo && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold text-slate-900">Usage by organization</h3>
                <UsageBar
                  current={tierInfo.usage.owned_orgs}
                  max={tierInfo.limits.max_orgs}
                  label=""
                />
                <span className="text-xs text-slate-500">
                  {tierInfo.usage.owned_orgs} / {tierInfo.limits.max_orgs === -1 ? '∞' : tierInfo.limits.max_orgs} orgs
                </span>
              </div>

              {tierInfo.usage.orgs && tierInfo.usage.orgs.length > 0 ? (
                tierInfo.usage.orgs.map((org) => (
                  (() => {
                    const ownerType = orgs.find((o) => o.id === org.id)?.owner_type
                    const projectCap = ownerType === 'personal' ? 10 : 2
                    const memberCap = ownerType === 'personal' ? 1 : 2
                    return (
                  <Card key={org.id}>
                    <div className="flex items-center gap-2 mb-3">
                      <div className="flex h-7 w-7 items-center justify-center rounded-md bg-violet-100 text-xs font-semibold text-violet-600">
                        {org.name[0]?.toUpperCase()}
                      </div>
                      <h4 className="text-sm font-semibold text-slate-900">{org.name}</h4>
                    </div>
                    <div className="grid gap-3 sm:grid-cols-3">
                      <UsageBar
                        current={org.projects}
                        max={projectCap}
                        label="Projects"
                      />
                      <UsageBar
                        current={org.members}
                        max={memberCap}
                        label="Team members"
                      />
                      <UsageBar
                        current={org.secrets}
                        max={-1}
                        label="Secrets"
                      />
                    </div>
                  </Card>
                    )
                  })()
                ))
              ) : (
                <Card>
                  <p className="text-sm text-slate-500 text-center py-4">No organizations yet. Create one to get started.</p>
                </Card>
              )}
            </div>
          )}
        </div>
      )}

      {tab === 'deployment' && (
        <div className="space-y-6">
          <div className="rounded-xl border border-cyan-200 bg-cyan-50/50 p-4">
            <h3 className="text-sm font-semibold text-cyan-900">Manual Deploy Sync</h3>
            <p className="mt-1 text-xs leading-relaxed text-cyan-800/90">
              Envo stays your source of truth. Add a deploy platform token once, then trigger sync manually from each environment page.
            </p>
          </div>

          <Card>
            <h3 className="mb-3 text-sm font-semibold text-slate-900">Add deployment platform connection</h3>
            <form onSubmit={handleCreateConnection} className="grid gap-3 sm:grid-cols-4">
              <label className="flex flex-col gap-1">
                <span className="text-xs text-slate-500">Platform</span>
                <select
                  value={platform}
                  onChange={(e) => setPlatform(e.target.value)}
                  className="rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-violet-400 focus:outline-none focus:ring-1 focus:ring-violet-400"
                >
                  <option value="vercel">Vercel</option>
                </select>
              </label>
              <label className="flex flex-col gap-1 sm:col-span-1">
                <span className="text-xs text-slate-500">Connection name</span>
                <input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Vercel Team"
                  className="rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-violet-400 focus:outline-none focus:ring-1 focus:ring-violet-400"
                />
              </label>
              <label className="flex flex-col gap-1 sm:col-span-2">
                <span className="text-xs text-slate-500">Token</span>
                <input
                  type="password"
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  placeholder="Paste platform API token"
                  className="rounded-md border border-slate-300 px-3 py-2 text-sm font-mono focus:border-violet-400 focus:outline-none focus:ring-1 focus:ring-violet-400"
                />
              </label>
              <div className="sm:col-span-4">
                <button
                  type="submit"
                  disabled={savingConnection || !token.trim()}
                  className="rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 disabled:opacity-60"
                >
                  {savingConnection ? 'Validating...' : 'Save connection'}
                </button>
              </div>
            </form>
            {connectionError && <p className="mt-2 text-sm text-red-600">{connectionError}</p>}
          </Card>

          <Card>
            <h3 className="mb-3 text-sm font-semibold text-slate-900">Saved connections</h3>
            {connections.length === 0 ? (
              <p className="text-sm text-slate-500">No deployment platform connections yet.</p>
            ) : (
              <div className="space-y-2">
                {connections.map((conn) => (
                  <div key={conn.id} className="flex items-center justify-between rounded-lg border border-slate-200 bg-white px-3 py-2">
                    <div>
                      <p className="text-sm font-medium text-slate-900">{conn.name}</p>
                      <p className="text-xs text-slate-500">
                        {conn.platform} · token prefix: <span className="font-mono">{conn.token_prefix}</span>
                      </p>
                    </div>
                    <button
                      onClick={() => handleDeleteConnection(conn.id, conn.name)}
                      className="rounded-md px-2.5 py-1.5 text-xs text-red-600 hover:bg-red-50"
                    >
                      Remove
                    </button>
                  </div>
                ))}
              </div>
            )}
          </Card>
        </div>
      )}

      {tab === 'account' && (
        <Card>
          <h3 className="text-sm font-semibold text-slate-900 mb-4">Account details</h3>
          {loadError ? (
            <div className="py-4 text-center">
              <p className="text-sm text-red-600 mb-3">Failed to load account details: {loadError}</p>
              <button
                onClick={load}
                className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 transition-colors"
              >
                Retry
              </button>
            </div>
          ) : loading || !user ? (
            <div className="space-y-3">
              <div>
                <Skeleton className="h-3 w-12 mb-1.5" />
                <Skeleton className="h-4 w-40" />
              </div>
              <div>
                <Skeleton className="h-3 w-10 mb-1.5" />
                <Skeleton className="h-4 w-52" />
              </div>
              <div>
                <Skeleton className="h-3 w-20 mb-1.5" />
                <Skeleton className="h-4 w-24" />
              </div>
              <div>
                <Skeleton className="h-3 w-20 mb-1.5" />
                <Skeleton className="h-4 w-32" />
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              <div>
                <label className="text-xs text-slate-500">Name</label>
                <p className="text-sm text-slate-900">{user.name}</p>
              </div>
              <div>
                <label className="text-xs text-slate-500">Email</label>
                <p className="text-sm text-slate-900">{user.email}</p>
              </div>
              <div>
                <label className="text-xs text-slate-500">Auth provider</label>
                <p className="text-sm text-slate-900 capitalize">{user.oauth_provider || 'Google'}</p>
              </div>
              <div>
                <label className="text-xs text-slate-500">Member since</label>
                <p className="text-sm text-slate-900">{user.created_at ? new Date(user.created_at).toLocaleDateString() : '-'}</p>
              </div>
            </div>
          )}
        </Card>
      )}
    </div>
  )
}
