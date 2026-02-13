import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import { getGoogleRedirectUrl } from '../lib/api'
import { getAccessToken } from '../lib/auth'

export function LoginPage() {
  const nav = useNavigate()
  const [error, setError] = useState<string | null>(null)

  // If already logged in, redirect to dashboard
  useEffect(() => {
    const token = getAccessToken()
    if (token) {
      nav('/orgs')
    }
  }, [nav])

  const handleLogin = () => {
    try {
      const callbackUrl = `${window.location.origin}/auth/callback`
      const redirectUrl = getGoogleRedirectUrl(callbackUrl)
      window.location.href = redirectUrl
    } catch (e) {
      setError((e as Error).message)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-slate-50 to-slate-100">
      <div className="w-full max-w-md px-4">
        <div className="mb-8 text-center">
          <h1 className="text-3xl font-bold text-slate-900">Envo</h1>
          <p className="mt-2 text-sm text-slate-600">
            Sign in to access your organizations and projects
          </p>
        </div>

        <Card>
          <h2 className="text-lg font-semibold text-slate-900">Sign in</h2>
          <p className="mt-1 text-sm text-slate-600">Use Google OAuth to continue</p>

          <div className="mt-5">
            <Button onClick={handleLogin} className="w-full">
              Continue with Google
            </Button>
          </div>

          {error ? <div className="mt-4 text-sm text-red-600">{error}</div> : null}
        </Card>

        <p className="mt-6 text-center text-xs text-slate-500">
          Secure • Encrypted • Team-ready
        </p>
      </div>
    </div>
  )
}
