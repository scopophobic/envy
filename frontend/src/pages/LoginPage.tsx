import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { getGoogleRedirectUrl } from '../lib/api'
import { getAccessToken } from '../lib/auth'

export function LoginPage() {
  const nav = useNavigate()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (getAccessToken()) nav('/orgs')
  }, [nav])

  const handleLogin = () => {
    try {
      const callbackUrl = `${window.location.origin}/auth/callback`
      window.location.href = getGoogleRedirectUrl(callbackUrl)
    } catch (e) {
      setError((e as Error).message)
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-[#faf9fb]">
      {/* Background gradient */}
      <div
        className="pointer-events-none absolute -top-[60%] left-1/2 -translate-x-1/2 w-[160%] aspect-square rounded-full opacity-50"
        style={{
          background: 'radial-gradient(circle at 50% 60%, #e8d5f5 0%, #d4b8f0 25%, #c49eea 40%, #f5eefc 70%, transparent 100%)',
        }}
      />

      <div className="relative z-10 w-full max-w-sm px-6">
        {/* Logo */}
        <div className="mb-10 text-center">
          <Link to="/" className="inline-flex items-center gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-slate-900 text-sm font-bold text-white">
              E
            </div>
            <span className="text-2xl font-bold text-slate-900 tracking-tight">Envo</span>
          </Link>
        </div>

        {/* Card */}
        <div className="rounded-2xl border border-slate-200/80 bg-white/90 p-8 shadow-xl shadow-slate-200/40 backdrop-blur-sm">
          <h2 className="text-center text-lg font-semibold text-slate-900">Welcome back</h2>
          <p className="mt-1 text-center text-sm text-slate-500">Sign in to manage your secrets</p>

          <button
            onClick={handleLogin}
            className="mt-6 flex w-full items-center justify-center gap-3 rounded-xl bg-slate-900 px-4 py-3 text-sm font-medium text-white shadow-sm hover:bg-slate-800 transition-all hover:shadow-md"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" className="shrink-0">
              <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.76h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
              <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
              <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
              <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
            </svg>
            Continue with Google
          </button>

          {error && (
            <div className="mt-4 rounded-lg bg-red-50 border border-red-100 px-3 py-2 text-sm text-red-600">
              {error}
            </div>
          )}
        </div>

        <p className="mt-6 text-center text-xs text-slate-400">
          Encrypted &middot; Secure &middot; Team-ready
        </p>
      </div>
    </div>
  )
}
