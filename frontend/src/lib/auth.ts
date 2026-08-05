const ACCESS_KEY = 'envo_access_token'
const REFRESH_KEY = 'envo_refresh_token'
const AUTH_CHANGE_EVENT = 'envo:auth-change'

function notifyAuthChange() {
  window.dispatchEvent(new Event(AUTH_CHANGE_EVENT))
}

export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS_KEY)
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_KEY)
}

export function setTokens(accessToken: string, refreshToken: string) {
  localStorage.setItem(ACCESS_KEY, accessToken)
  localStorage.setItem(REFRESH_KEY, refreshToken)
  notifyAuthChange()
}

export function clearTokens() {
  localStorage.removeItem(ACCESS_KEY)
  localStorage.removeItem(REFRESH_KEY)
  notifyAuthChange()
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

export function hasValidAccessToken(): boolean {
  const token = getAccessToken()
  if (!token) return false
  const expiry = decodeJwtClaims(token)?.exp
  return typeof expiry === 'number' && expiry * 1000 > Date.now()
}

export function subscribeToAuthChanges(onChange: () => void): () => void {
  const onStorage = (event: StorageEvent) => {
    if (event.key === ACCESS_KEY || event.key === REFRESH_KEY || event.key === null) onChange()
  }
  window.addEventListener(AUTH_CHANGE_EVENT, onChange)
  window.addEventListener('storage', onStorage)
  return () => {
    window.removeEventListener(AUTH_CHANGE_EVENT, onChange)
    window.removeEventListener('storage', onStorage)
  }
}
