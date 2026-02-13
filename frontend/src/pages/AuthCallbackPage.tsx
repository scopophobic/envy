import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { setTokens, getAccessToken } from '../lib/auth'

export function AuthCallbackPage() {
  const nav = useNavigate()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    // Read tokens from URL hash (backend redirects here with tokens)
    const hash = window.location.hash.substring(1) // Remove #
    console.log('AuthCallback: hash =', hash)
    
    const params = new URLSearchParams(hash)
    const accessToken = params.get('access_token')
    const refreshToken = params.get('refresh_token')
    
    console.log('AuthCallback: accessToken exists =', !!accessToken)
    console.log('AuthCallback: refreshToken exists =', !!refreshToken)

    if (accessToken && refreshToken) {
      try {
        setTokens(accessToken, refreshToken)
        // Verify tokens were saved
        const saved = getAccessToken()
        if (!saved) {
          setError('Failed to save tokens')
          setTimeout(() => nav('/login'), 2000)
          return
        }
        // Clear hash from URL
        window.history.replaceState(null, '', window.location.pathname)
        nav('/orgs')
      } catch (e) {
        console.error('AuthCallback error:', e)
        setError((e as Error).message)
        setTimeout(() => nav('/login'), 2000)
      }
    } else {
      // No tokens - check if we have them saved already (maybe page reload)
      const existing = getAccessToken()
      if (existing) {
        nav('/orgs')
      } else {
        setError('No tokens found in URL. Redirecting to login...')
        setTimeout(() => nav('/login'), 2000)
      }
    }
  }, [nav])

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <p className="text-sm text-slate-600">Completing login…</p>
        {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
      </div>
    </div>
  )
}
