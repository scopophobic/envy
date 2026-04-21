import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { acceptInvitation } from '../lib/api'
import { getAccessToken } from '../lib/auth'

export function InviteAcceptPage() {
  const [params] = useSearchParams()
  const nav = useNavigate()
  const [state, setState] = useState<'idle' | 'loading' | 'done' | 'error'>('idle')
  const [error, setError] = useState<string | null>(null)
  const token = params.get('token') || ''

  useEffect(() => {
    if (!token) {
      setError('Missing invitation token.')
      setState('error')
      return
    }
    if (!getAccessToken()) {
      sessionStorage.setItem('envo_invite_token', token)
      nav('/login', { replace: true })
      return
    }
    setState('loading')
    acceptInvitation(token)
      .then(() => {
        setState('done')
        setTimeout(() => nav('/orgs'), 1200)
      })
      .catch((e) => {
        setError((e as Error).message)
        setState('error')
      })
  }, [token, nav])

  return (
    <div className="mx-auto max-w-lg px-6 py-16">
      <div className="rounded-xl border border-slate-200 bg-white p-6 text-center">
        <h1 className="text-xl font-semibold text-slate-900">Accept invitation</h1>
        {state === 'loading' && <p className="mt-3 text-sm text-slate-500">Verifying invitation...</p>}
        {state === 'done' && <p className="mt-3 text-sm text-emerald-600">Invitation accepted. Redirecting…</p>}
        {state === 'error' && <p className="mt-3 text-sm text-red-600">{error}</p>}
        {state === 'error' && (
          <p className="mt-4">
            <Link to="/orgs" className="text-sm text-violet-700 hover:underline">Back to dashboard</Link>
          </p>
        )}
      </div>
    </div>
  )
}

