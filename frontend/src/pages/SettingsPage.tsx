import { useCallback, useEffect, useState } from 'react'
import { Card } from '../components/Card'
import {
  createCheckoutSession,
  createPortalSession,
  getCurrentUser,
  getTierInfo,
  type TierInfo,
  type User,
} from '../lib/api'

type Tab = 'plans' | 'account'

const PLANS = [
  {
    id: 'free',
    name: 'Free',
    price: '$0',
    period: 'forever',
    description: 'For personal projects and trying things out.',
    limits: {
      orgs: '1',
      projects: '1 per org',
      members: '2 per org',
      secrets: '50 per env',
      audit: '7 days',
    },
  },
  {
    id: 'starter',
    name: 'Starter',
    price: '$12',
    period: '/month',
    description: 'For small teams shipping fast.',
    limits: {
      orgs: '1',
      projects: '5 per org',
      members: '8 per org',
      secrets: '200 per env',
      audit: '30 days',
    },
    popular: true,
  },
  {
    id: 'team',
    name: 'Team',
    price: '$39',
    period: '/month',
    description: 'For growing teams that need full control.',
    limits: {
      orgs: 'Unlimited',
      projects: 'Unlimited',
      members: 'Unlimited',
      secrets: 'Unlimited',
      audit: '1 year',
    },
  },
]

function UsageBar({ current, max, label }: { current: number; max: number; label: string }) {
  const isUnlimited = max === -1
  const pct = isUnlimited ? 10 : max === 0 ? 0 : Math.min((current / max) * 100, 100)
  const atLimit = !isUnlimited && max > 0 && current >= max
  return (
    <div>
      <div className="flex items-center justify-between text-xs">
        <span className="text-slate-600">{label}</span>
        <span className={atLimit ? 'font-semibold text-red-600' : 'text-slate-500'}>
          {current} / {isUnlimited ? 'unlimited' : max}
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
  const [upgrading, setUpgrading] = useState<string | null>(null)

  const load = useCallback(() => {
    getCurrentUser().then(setUser).catch(() => {})
    getTierInfo().then(setTierInfo).catch(() => {})
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

  const currentTier = user?.tier || 'free'

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
          {/* Plan Cards */}
          <div className="grid gap-4 sm:grid-cols-3">
            {PLANS.map((plan) => {
              const isCurrent = plan.id === currentTier
              const isDowngrade = PLANS.findIndex(p => p.id === currentTier) >= PLANS.findIndex(p => p.id === plan.id)
              return (
                <div
                  key={plan.id}
                  className={`relative rounded-xl border p-5 transition-all ${
                    isCurrent
                      ? 'border-violet-300 bg-violet-50/50 ring-1 ring-violet-200'
                      : plan.popular
                        ? 'border-slate-200 bg-white shadow-sm'
                        : 'border-slate-200 bg-white'
                  }`}
                >
                  {plan.popular && !isCurrent && (
                    <div className="absolute -top-2.5 left-4 rounded-full bg-violet-600 px-2.5 py-0.5 text-[10px] font-semibold text-white">
                      Popular
                    </div>
                  )}
                  {isCurrent && (
                    <div className="absolute -top-2.5 left-4 rounded-full bg-violet-600 px-2.5 py-0.5 text-[10px] font-semibold text-white">
                      Current plan
                    </div>
                  )}

                  <div className="mt-1">
                    <h3 className="text-lg font-semibold text-slate-900">{plan.name}</h3>
                    <div className="mt-1 flex items-baseline gap-1">
                      <span className="text-3xl font-bold text-slate-900">{plan.price}</span>
                      <span className="text-sm text-slate-500">{plan.period}</span>
                    </div>
                    <p className="mt-2 text-xs text-slate-500">{plan.description}</p>
                  </div>

                  <ul className="mt-4 space-y-2">
                    {Object.entries(plan.limits).map(([key, val]) => (
                      <li key={key} className="flex items-center gap-2 text-xs text-slate-600">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="shrink-0 text-violet-500">
                          <polyline points="20 6 9 17 4 12" />
                        </svg>
                        <span>
                          <span className="font-medium">{val}</span>{' '}
                          {key === 'orgs' ? 'organizations' : key === 'audit' ? 'audit retention' : key}
                        </span>
                      </li>
                    ))}
                  </ul>

                  <div className="mt-5">
                    {isCurrent ? (
                      currentTier !== 'free' ? (
                        <button
                          onClick={handleManageBilling}
                          className="w-full rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-50 transition-colors"
                        >
                          Manage billing
                        </button>
                      ) : (
                        <div className="w-full rounded-lg bg-slate-100 px-4 py-2 text-center text-sm text-slate-500">
                          Current plan
                        </div>
                      )
                    ) : isDowngrade ? (
                      <div className="w-full rounded-lg bg-slate-50 px-4 py-2 text-center text-xs text-slate-400">
                        Contact support to downgrade
                      </div>
                    ) : (
                      <button
                        onClick={() => handleUpgrade(plan.id)}
                        disabled={upgrading === plan.id}
                        className="w-full rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-violet-700 transition-colors disabled:opacity-60"
                      >
                        {upgrading === plan.id ? 'Redirecting...' : `Upgrade to ${plan.name}`}
                      </button>
                    )}
                  </div>
                </div>
              )
            })}
          </div>

          {/* Usage */}
          {tierInfo && (
            <Card>
              <h3 className="text-sm font-semibold text-slate-900 mb-4">Current usage</h3>
              <div className="grid gap-4 sm:grid-cols-2">
                <UsageBar
                  current={tierInfo.usage.owned_orgs}
                  max={tierInfo.limits.max_orgs}
                  label="Organizations"
                />
                <UsageBar
                  current={tierInfo.usage.total_projects}
                  max={tierInfo.limits.max_projects_per_org}
                  label="Projects"
                />
                <UsageBar
                  current={tierInfo.usage.total_members}
                  max={tierInfo.limits.max_devs_per_org}
                  label="Team members"
                />
                <UsageBar
                  current={tierInfo.usage.total_secrets}
                  max={tierInfo.limits.max_secrets_per_env}
                  label="Secrets"
                />
              </div>
            </Card>
          )}
        </div>
      )}

      {tab === 'account' && (
        <Card>
          <h3 className="text-sm font-semibold text-slate-900 mb-4">Account details</h3>
          {user ? (
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
          ) : (
            <p className="text-sm text-slate-500">Loading...</p>
          )}
        </Card>
      )}
    </div>
  )
}
