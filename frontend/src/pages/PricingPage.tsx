import { Link } from 'react-router-dom'
import { getAccessToken } from '../lib/auth'
import { SUBSCRIPTION_PLANS } from '../lib/pricing'

const FEATURE_ROWS = [
  { feature: 'Dashboard + CLI + API', free: true, starter: true, team: true },
  { feature: 'Encrypted secret storage', free: true, starter: true, team: true },
  { feature: 'Manual deploy sync', free: true, starter: true, team: true },
  { feature: 'Environment limits', free: 'Base limits', starter: 'Higher limits', team: 'Highest limits' },
  { feature: 'Team collaboration', free: 'Limited', starter: true, team: true },
  { feature: 'Priority support', free: false, starter: false, team: true },
]

function Cell({ value }: { value: boolean | string }) {
  if (typeof value === 'string') return <span className="text-xs text-slate-700">{value}</span>
  return value ? <span className="text-emerald-700">✓</span> : <span className="text-slate-300">—</span>
}

export function PricingPage() {
  const isLoggedIn = !!getAccessToken()

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
          <Link to="/" className="text-xl font-bold tracking-tight text-slate-900">Envo</Link>
          <div className="flex items-center gap-3">
            <Link to="/" className="text-sm text-slate-600 hover:text-slate-900">Home</Link>
            <Link
              to={isLoggedIn ? '/orgs' : '/login'}
              className="rounded-md bg-slate-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-800"
            >
              {isLoggedIn ? 'Dashboard' : 'Get Started'}
            </Link>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-6 py-12">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-slate-900">Pricing</h1>
          <p className="mx-auto mt-3 max-w-2xl text-sm text-slate-600">
            Choose a plan that matches your team stage. Start free, then move to paid plans for larger limits and collaboration.
          </p>
        </div>

        <section className="mt-10 grid gap-4 lg:grid-cols-3">
          {SUBSCRIPTION_PLANS.map((plan) => (
            <div
              key={plan.id}
              className={`rounded-xl border bg-white p-6 ${
                plan.id === 'starter' ? 'border-violet-300 shadow-md shadow-violet-100' : 'border-slate-200'
              }`}
            >
              <h2 className="text-2xl font-semibold text-slate-900">{plan.name === 'Team' ? 'Professional Team' : plan.name}</h2>
              <p className="mt-2 text-3xl font-bold text-slate-900">{plan.priceLine}<span className="text-sm font-medium text-slate-500">/mo</span></p>
              <p className="mt-2 text-sm text-slate-600">{plan.detail}</p>
              <ul className="mt-5 space-y-2">
                {plan.highlights.map((h) => (
                  <li key={h} className="flex items-start gap-2 text-sm text-slate-700">
                    <span className="text-emerald-600">✓</span>
                    <span>{h}</span>
                  </li>
                ))}
              </ul>
              <div className="mt-6">
                <Link
                  to={isLoggedIn ? '/settings' : '/login'}
                  className={`inline-flex w-full items-center justify-center rounded-md px-4 py-2 text-sm font-medium ${
                    plan.id === 'starter'
                      ? 'bg-violet-600 text-white hover:bg-violet-700'
                      : 'border border-slate-300 bg-white text-slate-800 hover:bg-slate-50'
                  }`}
                >
                  {plan.id === 'free' ? 'Get Started' : 'Start Trial'}
                </Link>
              </div>
            </div>
          ))}
        </section>

        <section className="mt-12 rounded-xl border border-slate-200 bg-white">
          <div className="border-b border-slate-200 px-5 py-4">
            <h3 className="text-lg font-semibold text-slate-900">Compare features</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="min-w-full text-left">
              <thead className="bg-slate-50">
                <tr>
                  <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">Feature</th>
                  <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">Free</th>
                  <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">Starter</th>
                  <th className="px-5 py-3 text-xs font-semibold uppercase tracking-wide text-slate-500">Professional Team</th>
                </tr>
              </thead>
              <tbody>
                {FEATURE_ROWS.map((row) => (
                  <tr key={row.feature} className="border-t border-slate-100">
                    <td className="px-5 py-3 text-sm text-slate-800">{row.feature}</td>
                    <td className="px-5 py-3 text-sm"><Cell value={row.free} /></td>
                    <td className="px-5 py-3 text-sm"><Cell value={row.starter} /></td>
                    <td className="px-5 py-3 text-sm"><Cell value={row.team} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </main>
    </div>
  )
}
