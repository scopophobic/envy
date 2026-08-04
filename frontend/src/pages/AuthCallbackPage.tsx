import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { setTokens, getAccessToken } from '../lib/auth'

export function AuthCallbackPage() {
  const nav = useNavigate()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const completeLogin = async () => {
      // Keep the token handoff in the URL fragment, then remove it immediately.
      await Promise.resolve()
      const params = new URLSearchParams(window.location.hash.substring(1))
      const accessToken = params.get('access_token')
      const refreshToken = params.get('refresh_token')

      try {
        if (accessToken && refreshToken) {
          setTokens(accessToken, refreshToken)
          window.history.replaceState(null, '', window.location.pathname)
        } else if (!getAccessToken()) {
          throw new Error('The sign-in response was incomplete. Please try again.')
        }

        const inviteToken = sessionStorage.getItem('envo_invite_token')
        if (inviteToken) {
          sessionStorage.removeItem('envo_invite_token')
          nav(`/invite/accept?token=${encodeURIComponent(inviteToken)}`, { replace: true })
        } else {
          nav('/orgs', { replace: true })
        }
      } catch (e) {
        setError((e as Error).message)
        setTimeout(() => nav('/login', { replace: true }), 2200)
      }
    }

    void completeLogin()
  }, [nav])

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        {!error && <div className="mx-auto h-6 w-6 animate-spin rounded-full border-2 border-slate-200 border-t-violet-600" />}
        <p className="mt-3 text-sm text-slate-600">{error ? 'Sign-in could not be completed' : 'Completing sign-in…'}</p>
        {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
      </div>
    </div>
  )
}
