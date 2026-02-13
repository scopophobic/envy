const ACCESS_KEY = 'envo_access_token'
const REFRESH_KEY = 'envo_refresh_token'

export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS_KEY)
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_KEY)
}

export function setTokens(accessToken: string, refreshToken: string) {
  localStorage.setItem(ACCESS_KEY, accessToken)
  localStorage.setItem(REFRESH_KEY, refreshToken)
}

export function clearTokens() {
  localStorage.removeItem(ACCESS_KEY)
  localStorage.removeItem(REFRESH_KEY)
}

export type JwtClaims = {
  user_id?: string
  email?: string
  permissions?: string[]
  exp?: number
}

export function decodeJwtClaims(token: string): JwtClaims | null {
  try {
    const parts = token.split('.')
    if (parts.length < 2) return null
    const payload = parts[1]
    const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
    return JSON.parse(json) as JwtClaims
  } catch {
    return null
  }
}

export function getPermissions(): string[] {
  const t = getAccessToken()
  if (!t) return []
  return decodeJwtClaims(t)?.permissions ?? []
}

