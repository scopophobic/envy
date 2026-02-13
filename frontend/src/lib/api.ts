import { clearTokens, getAccessToken, getRefreshToken, setTokens } from './auth'

const DEFAULT_BASE = 'http://localhost:8080'

export function apiBaseUrl() {
  return import.meta.env.VITE_API_URL || DEFAULT_BASE
}

async function request<T>(
  path: string,
  init: RequestInit & { auth?: boolean } = {},
): Promise<T> {
  const url = apiBaseUrl() + path
  const headers = new Headers(init.headers || {})
  headers.set('Accept', 'application/json')

  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  if (init.auth) {
    const token = getAccessToken()
    if (token) headers.set('Authorization', `Bearer ${token}`)
  }

  const resp = await fetch(url, { ...init, headers })
  const text = await resp.text()

  if (!resp.ok) {
    throw new Error(`${resp.status} ${resp.statusText}: ${text}`)
  }

  return text ? (JSON.parse(text) as T) : (null as unknown as T)
}

// ── Auth ─────────────────────────────────────────────────────────────

export type OAuthLoginUrlResp = { url: string }

export async function getGoogleLoginUrl(): Promise<string> {
  const r = await request<OAuthLoginUrlResp>('/api/v1/auth/google/login')
  return r.url
}

export function getGoogleRedirectUrl(frontendCallbackUrl: string): string {
  const base = apiBaseUrl()
  const next = encodeURIComponent(frontendCallbackUrl)
  return `${base}/api/v1/auth/google/redirect?next=${next}`
}

export type RefreshResp = { access_token: string; token_type: string; expires_in: number }

export async function refreshAccessToken(): Promise<string> {
  const refresh = getRefreshToken()
  if (!refresh) throw new Error('Missing refresh token')
  const r = await request<RefreshResp>('/api/v1/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: refresh }),
  })
  setTokens(r.access_token, refresh)
  return r.access_token
}

export async function logout(): Promise<void> {
  const refresh = getRefreshToken()
  if (refresh) {
    try {
      await request('/api/v1/auth/logout', {
        method: 'POST',
        body: JSON.stringify({ refresh_token: refresh }),
      })
    } catch {
      // ignore
    }
  }
  clearTokens()
}

export type User = {
  id: string
  email: string
  name: string
  tier: string
  oauth_provider?: string
  created_at?: string
}

export async function getCurrentUser(): Promise<User> {
  return request<User>('/api/v1/auth/me', { auth: true })
}

// ── Organizations ────────────────────────────────────────────────────

export type Org = { id: string; name: string; owner?: { id: string; email: string; name: string } }

export async function listOrgs(): Promise<Org[]> {
  return request<Org[]>('/api/v1/orgs', { auth: true })
}

export type OrgMember = {
  id: string
  org_id: string
  user_id: string
  created_at: string
  user?: { id: string; email: string; name: string }
  role?: { id: string; name: string; is_system_role: boolean }
}

export type OrgDetail = Org & { members?: OrgMember[] }

export async function getOrg(id: string): Promise<OrgDetail> {
  return request<OrgDetail>(`/api/v1/orgs/${id}`, { auth: true })
}

export async function createOrg(name: string): Promise<Org> {
  return request<Org>('/api/v1/orgs', {
    method: 'POST',
    auth: true,
    body: JSON.stringify({ name }),
  })
}

export async function updateOrg(id: string, name: string): Promise<Org> {
  return request<Org>(`/api/v1/orgs/${id}`, {
    method: 'PATCH',
    auth: true,
    body: JSON.stringify({ name }),
  })
}

export async function deleteOrg(id: string): Promise<void> {
  await request(`/api/v1/orgs/${id}`, { method: 'DELETE', auth: true })
}

// ── Members ──────────────────────────────────────────────────────────

export async function inviteMember(orgId: string, email: string, role: string): Promise<OrgMember> {
  return request<OrgMember>(`/api/v1/orgs/${orgId}/members`, {
    method: 'POST',
    auth: true,
    body: JSON.stringify({ email, role }),
  })
}

export async function updateMemberRole(orgId: string, memberId: string, role: string): Promise<OrgMember> {
  return request<OrgMember>(`/api/v1/orgs/${orgId}/members/${memberId}`, {
    method: 'PATCH',
    auth: true,
    body: JSON.stringify({ role }),
  })
}

export async function removeMember(orgId: string, memberId: string): Promise<void> {
  await request(`/api/v1/orgs/${orgId}/members/${memberId}`, { method: 'DELETE', auth: true })
}

// ── Projects ─────────────────────────────────────────────────────────

export type Project = { id: string; org_id: string; name: string; description?: string }

export async function listOrgProjects(orgId: string): Promise<Project[]> {
  return request<Project[]>(`/api/v1/orgs/${orgId}/projects`, { auth: true })
}

export async function getProject(projectId: string): Promise<Project> {
  return request<Project>(`/api/v1/projects/${projectId}`, { auth: true })
}

export async function createProject(orgId: string, name: string, description?: string): Promise<Project> {
  return request<Project>(`/api/v1/orgs/${orgId}/projects`, {
    method: 'POST',
    auth: true,
    body: JSON.stringify({ name, description: description || null }),
  })
}

export async function updateProject(projectId: string, name: string, description?: string): Promise<Project> {
  return request<Project>(`/api/v1/projects/${projectId}`, {
    method: 'PATCH',
    auth: true,
    body: JSON.stringify({ name, description: description || null }),
  })
}

export async function deleteProject(projectId: string): Promise<void> {
  await request(`/api/v1/projects/${projectId}`, { method: 'DELETE', auth: true })
}

// ── Environments ─────────────────────────────────────────────────────

export type Environment = { id: string; project_id: string; name: string }

export async function listProjectEnvironments(projectId: string): Promise<Environment[]> {
  return request<Environment[]>(`/api/v1/projects/${projectId}/environments`, { auth: true })
}

export async function createEnvironment(projectId: string, name: string): Promise<Environment> {
  return request<Environment>(`/api/v1/projects/${projectId}/environments`, {
    method: 'POST',
    auth: true,
    body: JSON.stringify({ name }),
  })
}

export async function updateEnvironment(envId: string, name: string): Promise<Environment> {
  return request<Environment>(`/api/v1/environments/${envId}`, {
    method: 'PATCH',
    auth: true,
    body: JSON.stringify({ name }),
  })
}

export async function deleteEnvironment(envId: string): Promise<void> {
  await request(`/api/v1/environments/${envId}`, { method: 'DELETE', auth: true })
}

// ── Secrets ──────────────────────────────────────────────────────────

export type Secret = {
  id: string
  environment_id: string
  key: string
  created_by: string
  created_at: string
  updated_at: string
}

export async function listSecrets(envId: string): Promise<Secret[]> {
  return request<Secret[]>(`/api/v1/environments/${envId}/secrets`, { auth: true })
}

export async function createSecret(envId: string, key: string, value: string): Promise<Secret> {
  return request<Secret>(`/api/v1/environments/${envId}/secrets`, {
    method: 'POST',
    auth: true,
    body: JSON.stringify({ key, value }),
  })
}

export async function updateSecret(secretId: string, key?: string, value?: string): Promise<Secret> {
  const body: Record<string, string> = {}
  if (key !== undefined) body.key = key
  if (value !== undefined) body.value = value
  return request<Secret>(`/api/v1/secrets/${secretId}`, {
    method: 'PATCH',
    auth: true,
    body: JSON.stringify(body),
  })
}

export async function deleteSecret(secretId: string): Promise<void> {
  await request(`/api/v1/secrets/${secretId}`, { method: 'DELETE', auth: true })
}

export type ExportedSecrets = Record<string, string>

export async function exportSecrets(envId: string): Promise<ExportedSecrets> {
  return request<ExportedSecrets>(`/api/v1/environments/${envId}/secrets/export`, { auth: true })
}

// ── Audit Logs ───────────────────────────────────────────────────────

export type AuditLog = {
  id: string
  user_id: string
  org_id: string
  action: string
  resource_type: string
  resource_id: string
  details: string
  created_at: string
  user?: { id: string; email: string; name: string }
}

export async function listAuditLogs(orgId: string): Promise<AuditLog[]> {
  return request<AuditLog[]>(`/api/v1/orgs/${orgId}/audit-logs`, { auth: true })
}
