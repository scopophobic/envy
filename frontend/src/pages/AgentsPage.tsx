import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import {
  createAgent,
  createAgentCredential,
  createAgentGrant,
  getOrg,
  listAgentCredentials,
  listAgentGrants,
  listAgents,
  listOrgProjects,
  listProjectEnvironments,
  revokeAgentCredential,
  revokeAgentGrant,
  updateAgentStatus,
  type AgentCredential,
  type AgentGrant,
  type AgentIdentity,
  type Environment,
  type OrgDetail,
  type Project,
} from '../lib/api'

const inputClass = 'w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none focus:border-violet-400 focus:ring-2 focus:ring-violet-100'

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : 'Never'
}

export function AgentsPage() {
  const { id: orgId } = useParams()
  const [org, setOrg] = useState<OrgDetail | null>(null)
  const [agents, setAgents] = useState<AgentIdentity[]>([])
  const [selectedId, setSelectedId] = useState('')
  const [credentials, setCredentials] = useState<AgentCredential[]>([])
  const [grants, setGrants] = useState<AgentGrant[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [environments, setEnvironments] = useState<Environment[]>([])
  const [projectId, setProjectId] = useState('')
  const [environmentId, setEnvironmentId] = useState('')
  const [error, setError] = useState('')
  const [busyAction, setBusyAction] = useState<string | null>(null)
  const [newName, setNewName] = useState('')
  const [newDescription, setNewDescription] = useState('')
  const [credentialName, setCredentialName] = useState('coding harness')
  const [credentialExpiry, setCredentialExpiry] = useState('')
  const [allowedKeys, setAllowedKeys] = useState('')
  const [allowAll, setAllowAll] = useState(false)
  const [grantExpiry, setGrantExpiry] = useState('')
  const [issuedToken, setIssuedToken] = useState('')
  const [tokenCopied, setTokenCopied] = useState(false)

  const selected = useMemo(() => agents.find(agent => agent.id === selectedId) ?? null, [agents, selectedId])

  const loadAgents = useCallback(async () => {
    if (!orgId) return
    try {
      const data = await listAgents(orgId)
      setAgents(data)
      setSelectedId(current => current && data.some(a => a.id === current) ? current : (data[0]?.id ?? ''))
    } catch (err) {
      setError((err as Error).message)
    }
  }, [orgId])

  const loadSelected = useCallback(async () => {
    if (!orgId || !selectedId) {
      setCredentials([])
      setGrants([])
      return
    }
    try {
      const [credentialData, grantData] = await Promise.all([
        listAgentCredentials(orgId, selectedId),
        listAgentGrants(orgId, selectedId),
      ])
      setCredentials(credentialData)
      setGrants(grantData)
    } catch (err) {
      setError((err as Error).message)
    }
  }, [orgId, selectedId])

  useEffect(() => {
    if (!orgId) return
    getOrg(orgId).then(setOrg).catch(err => setError((err as Error).message))
    listOrgProjects(orgId).then(data => {
      setProjects(data)
      setProjectId(current => current || data[0]?.id || '')
    }).catch(err => setError((err as Error).message))
    loadAgents()
  }, [orgId, loadAgents])

  useEffect(() => { loadSelected() }, [loadSelected])

  useEffect(() => {
    if (!projectId) {
      setEnvironments([])
      setEnvironmentId('')
      return
    }
    listProjectEnvironments(projectId).then(data => {
      setEnvironments(data)
      setEnvironmentId(current => data.some(env => env.id === current) ? current : (data[0]?.id ?? ''))
    }).catch(err => setError((err as Error).message))
  }, [projectId])

  async function handleCreateAgent(event: React.FormEvent) {
    event.preventDefault()
    if (!orgId || !newName.trim()) return
    setBusyAction('create-agent'); setError('')
    try {
      const agent = await createAgent(orgId, { name: newName.trim(), description: newDescription.trim() })
      setNewName(''); setNewDescription(''); setSelectedId(agent.id)
      await loadAgents()
    } catch (err) { setError((err as Error).message) } finally { setBusyAction(null) }
  }

  async function handleCreateCredential(event: React.FormEvent) {
    event.preventDefault()
    if (!orgId || !selected || !credentialName.trim()) return
    setBusyAction('create-credential'); setError(''); setIssuedToken('')
    try {
      const result = await createAgentCredential(orgId, selected.id, {
        name: credentialName.trim(),
        expires_at: credentialExpiry ? new Date(credentialExpiry).toISOString() : undefined,
      })
      setIssuedToken(result.token)
      await loadSelected()
    } catch (err) { setError((err as Error).message) } finally { setBusyAction(null) }
  }

  async function handleCreateGrant(event: React.FormEvent) {
    event.preventDefault()
    if (!orgId || !selected || !environmentId) return
    const keys = allowedKeys.split(/[\n,]/).map(key => key.trim()).filter(Boolean)
    setBusyAction('create-grant'); setError('')
    try {
      await createAgentGrant(orgId, selected.id, {
        environment_id: environmentId,
        allowed_keys: keys,
        allow_all_secrets: allowAll,
        expires_at: grantExpiry ? new Date(grantExpiry).toISOString() : undefined,
      })
      setAllowedKeys(''); setAllowAll(false); setGrantExpiry('')
      await loadSelected()
    } catch (err) { setError((err as Error).message) } finally { setBusyAction(null) }
  }

  async function changeStatus(status: 'active' | 'suspended' | 'revoked') {
    if (!orgId || !selected) return
    if (status === 'revoked' && !confirm('Permanently revoke this agent, all tokens, and all grants?')) return
    setBusyAction('status'); setError('')
    try { await updateAgentStatus(orgId, selected.id, status); await loadAgents() }
    catch (err) { setError((err as Error).message) } finally { setBusyAction(null) }
  }

  async function handleRevokeCredential(credentialId: string) {
    if (!orgId || !selected) return
    setBusyAction(`credential-${credentialId}`); setError('')
    try { await revokeAgentCredential(orgId, selected.id, credentialId); await loadSelected() }
    catch (err) { setError((err as Error).message) } finally { setBusyAction(null) }
  }

  async function handleRevokeGrant(grantId: string) {
    if (!orgId || !selected) return
    setBusyAction(`grant-${grantId}`); setError('')
    try { await revokeAgentGrant(orgId, selected.id, grantId); await loadSelected() }
    catch (err) { setError((err as Error).message) } finally { setBusyAction(null) }
  }

  return (
    <div className="space-y-7">
      <div className="overflow-hidden rounded-3xl border border-slate-800 bg-slate-950 text-white shadow-[0_22px_60px_rgba(15,23,42,.18)]">
        <div className="relative p-6 sm:p-8">
          <div className="pointer-events-none absolute -right-16 -top-20 h-64 w-64 rounded-full bg-violet-500/20 blur-3xl" />
          <div className="pointer-events-none absolute -bottom-24 left-1/3 h-52 w-52 rounded-full bg-emerald-400/10 blur-3xl" />
          <div className="relative">
            <div className="text-sm text-slate-400">
              <Link to="/orgs" className="hover:text-white">Organizations</Link><span className="mx-1">/</span>
              <Link to={`/orgs/${orgId}`} className="hover:text-white">{org?.name ?? 'Workspace'}</Link><span className="mx-1">/</span>
              <span className="font-medium text-white">Agents</span>
            </div>
            <div className="mt-5 flex flex-col justify-between gap-6 lg:flex-row lg:items-end">
              <div>
                <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-violet-400/25 bg-violet-400/10 px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.14em] text-violet-200">
                  <span className="status-pulse h-1.5 w-1.5 rounded-full bg-emerald-400 text-emerald-400" /> Agent access plane
                </div>
                <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">Controlled access for AI agents</h1>
                <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">Give every coding harness its own identity, exact secret scope, and independent kill switch—without rotating underlying credentials.</p>
              </div>
              <div className="flex flex-wrap items-center gap-2 text-xs text-slate-400">
                <span className="rounded-lg border border-white/10 bg-white/5 px-3 py-2"><strong className="text-white">{agents.filter(a => a.status === 'active').length}</strong> active</span>
                <span className="text-slate-600">→</span>
                <span className="rounded-lg border border-white/10 bg-white/5 px-3 py-2"><strong className="text-white">{grants.filter(g => !g.revoked_at).length}</strong> live grants</span>
                <span className="text-slate-600">→</span>
                <span className="rounded-lg border border-emerald-400/20 bg-emerald-400/10 px-3 py-2 text-emerald-300">Audited</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {error && <div className="page-enter flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 shadow-sm"><span className="flex h-6 w-6 items-center justify-center rounded-full bg-red-100 font-bold">!</span>{error}</div>}

      <div className="grid gap-6 lg:grid-cols-[320px_1fr]">
        <div className="space-y-4">
          <Card className="border-violet-200/60 bg-gradient-to-b from-violet-50/60 to-white">
            <form onSubmit={handleCreateAgent} className="space-y-3 p-5">
              <div className="flex items-center gap-3">
                <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-violet-100 text-violet-600"><svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 8V4H8"/><rect width="16" height="12" x="4" y="8" rx="2"/><path d="M2 14h2M20 14h2M9 13v2M15 13v2"/></svg></span>
                <div><h2 className="font-semibold text-slate-900">Create an agent</h2><p className="text-xs text-slate-500">A separate identity for each harness.</p></div>
              </div>
              <input className={inputClass} placeholder="e.g. Claude Code — backend" value={newName} onChange={e => setNewName(e.target.value)} />
              <textarea className={inputClass} rows={2} placeholder="What this identity is used for" value={newDescription} onChange={e => setNewDescription(e.target.value)} />
              <Button type="submit" loading={busyAction === 'create-agent'} disabled={!newName.trim() || busyAction !== null} className="w-full">Create agent</Button>
            </form>
          </Card>
          <Card className="overflow-hidden p-0">
            <div className="border-b border-slate-100 px-4 py-3 text-[11px] font-semibold uppercase tracking-[0.12em] text-slate-400">Identities</div>
            <div className="divide-y divide-slate-100">
              {agents.length === 0 && <p className="p-5 text-sm text-slate-500">No agent identities yet.</p>}
              {agents.map(agent => (
                <button key={agent.id} onClick={() => { setSelectedId(agent.id); setIssuedToken('') }} className={`group w-full p-4 text-left transition-all hover:bg-slate-50 ${selectedId === agent.id ? 'bg-violet-50/80 shadow-[inset_3px_0_0_#8b5cf6]' : ''}`}>
                  <div className="flex items-center gap-3">
                    <span className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-xl text-xs font-bold transition-transform group-hover:scale-105 ${selectedId === agent.id ? 'bg-violet-600 text-white shadow-md shadow-violet-200' : 'bg-slate-100 text-slate-500'}`}>{agent.name.slice(0, 2).toUpperCase()}</span>
                    <div className="min-w-0 flex-1"><div className="flex items-center justify-between gap-2"><span className="truncate text-sm font-medium text-slate-900">{agent.name}</span><span className={`rounded-full px-2 py-0.5 text-[10px] font-semibold ${agent.status === 'active' ? 'bg-emerald-50 text-emerald-700' : agent.status === 'suspended' ? 'bg-amber-50 text-amber-700' : 'bg-slate-100 text-slate-500'}`}>{agent.status}</span></div><p className="mt-1 truncate text-xs text-slate-500">Last used: {formatDate(agent.last_used_at)}</p></div>
                  </div>
                </button>
              ))}
            </div>
          </Card>
        </div>

        {selected ? <div className="space-y-6">
          <Card className="overflow-hidden p-0">
            <div className="flex flex-wrap items-start justify-between gap-3 p-5">
              <div className="flex items-center gap-3"><span className={`status-pulse h-2.5 w-2.5 rounded-full ${selected.status === 'active' ? 'bg-emerald-500 text-emerald-500' : 'bg-slate-400 text-slate-400'}`} /><div><h2 className="text-lg font-semibold text-slate-900">{selected.name}</h2><p className="mt-1 text-sm text-slate-500">{selected.description || 'No description'}</p></div></div>
              <div className="flex gap-2">
                {selected.status === 'active' ? <Button variant="secondary" disabled={busyAction !== null} onClick={() => changeStatus('suspended')}>Suspend</Button> : selected.status === 'suspended' ? <Button variant="secondary" disabled={busyAction !== null} onClick={() => changeStatus('active')}>Reactivate</Button> : null}
                {selected.status !== 'revoked' && <Button variant="danger" loading={busyAction === 'status'} disabled={busyAction !== null} onClick={() => changeStatus('revoked')}>Revoke agent</Button>}
              </div>
            </div>
          </Card>

          {issuedToken && <div className="page-enter overflow-hidden rounded-2xl border border-amber-200 bg-gradient-to-r from-amber-50 to-orange-50 shadow-[0_12px_35px_rgba(245,158,11,.09)]">
            <div className="flex items-start gap-3 border-b border-amber-200/70 p-5">
              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-amber-100 text-amber-700"><svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="7.5" cy="15.5" r="5.5"/><path d="m21 2-9.6 9.6M15 5l4 4"/></svg></span>
              <div><h3 className="font-semibold text-amber-950">Copy this token now</h3><p className="mt-0.5 text-xs text-amber-800">For security, Envo cannot show it again.</p></div>
            </div>
            <div className="p-5">
              <div className="flex flex-col gap-2 sm:flex-row"><code className="scrollbar-subtle min-w-0 flex-1 overflow-x-auto rounded-xl border border-slate-800 bg-slate-950 px-4 py-3 text-xs text-emerald-300 shadow-inner">{issuedToken}</code><Button variant="secondary" onClick={() => { navigator.clipboard.writeText(issuedToken); setTokenCopied(true); setTimeout(() => setTokenCopied(false), 1800) }}>{tokenCopied ? 'Copied ✓' : 'Copy'}</Button></div>
              <p className="mt-3 text-xs text-amber-800">Set it as <code>ENVO_TOKEN</code>, then run <code>envo run --project PROJECT --env ENV -- your-agent-command</code>.</p>
            </div>
          </div>}

          <div className="grid gap-6 xl:grid-cols-2">
            <Card><div className="p-5">
              <div className="flex items-start gap-3"><span className="flex h-9 w-9 items-center justify-center rounded-xl bg-slate-100 text-slate-600"><svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="7.5" cy="15.5" r="5.5"/><path d="m21 2-9.6 9.6M15 5l4 4"/></svg></span><div><h3 className="font-semibold text-slate-900">Credentials</h3><p className="mt-1 text-xs text-slate-500">One token per device or harness.</p></div></div>
              <form onSubmit={handleCreateCredential} className="mt-4 space-y-2"><label className="block text-[11px] font-semibold uppercase tracking-wide text-slate-400">Credential name</label><input className={inputClass} value={credentialName} onChange={e => setCredentialName(e.target.value)} /><label className="block pt-1 text-[11px] font-semibold uppercase tracking-wide text-slate-400">Expires (optional)</label><input className={inputClass} type="datetime-local" aria-label="Credential expiry (optional)" value={credentialExpiry} onChange={e => setCredentialExpiry(e.target.value)} /><Button type="submit" loading={busyAction === 'create-credential'} disabled={selected.status === 'revoked' || busyAction !== null} className="w-full">Issue token</Button></form>
              <div className="mt-4 divide-y divide-slate-100">{credentials.map(credential => <div key={credential.id} className="flex items-center justify-between gap-3 py-3"><div className="min-w-0"><p className="truncate text-sm font-medium text-slate-800">{credential.name}</p><p className="text-xs text-slate-500"><code>{credential.token_prefix}…</code> · Last used {formatDate(credential.last_used_at)}</p></div>{credential.revoked_at ? <span className="text-xs text-slate-400">Revoked</span> : <button disabled={busyAction !== null} className="rounded-lg px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 disabled:opacity-50" onClick={() => handleRevokeCredential(credential.id)}>{busyAction === `credential-${credential.id}` ? 'Revoking…' : 'Revoke'}</button>}</div>)}</div>
            </div></Card>

            <Card><div className="p-5">
              <div className="flex items-start gap-3"><span className="flex h-9 w-9 items-center justify-center rounded-xl bg-violet-100 text-violet-600"><svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="m9 12 2 2 4-4"/></svg></span><div><h3 className="font-semibold text-slate-900">Add secret grant</h3><p className="mt-1 text-xs text-slate-500">Choose the exact boundary this agent gets.</p></div></div>
              <form onSubmit={handleCreateGrant} className="mt-4 space-y-3">
                <select className={inputClass} value={projectId} onChange={e => setProjectId(e.target.value)}><option value="">Select project</option>{projects.map(project => <option key={project.id} value={project.id}>{project.name}</option>)}</select>
                <select className={inputClass} value={environmentId} onChange={e => setEnvironmentId(e.target.value)}><option value="">Select environment</option>{environments.map(env => <option key={env.id} value={env.id}>{env.name}</option>)}</select>
                <textarea className={inputClass} rows={3} disabled={allowAll} placeholder="DATABASE_URL, OPENAI_API_KEY" value={allowedKeys} onChange={e => setAllowedKeys(e.target.value)} />
                <input className={inputClass} type="datetime-local" aria-label="Grant expiry (optional)" value={grantExpiry} onChange={e => setGrantExpiry(e.target.value)} />
                <label className="flex items-center gap-2 text-sm text-slate-600"><input type="checkbox" checked={allowAll} onChange={e => setAllowAll(e.target.checked)} />Allow every secret in this environment, including future ones</label>
                <Button type="submit" loading={busyAction === 'create-grant'} disabled={!environmentId || (!allowAll && !allowedKeys.trim()) || selected.status === 'revoked' || busyAction !== null} className="w-full">Create grant</Button>
              </form>
            </div></Card>
          </div>

          <Card><div className="p-5"><div className="flex items-center justify-between"><div><h3 className="font-semibold text-slate-900">Access grants</h3><p className="mt-1 text-xs text-slate-500">Live policy evaluated on every secret request.</p></div><span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-500">{grants.filter(g => !g.revoked_at).length} live</span></div>
            <div className="mt-4 grid gap-2">{grants.length === 0 && <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50/70 px-4 py-7 text-center"><span className="mx-auto flex h-9 w-9 items-center justify-center rounded-xl bg-white text-slate-400 shadow-sm"><svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg></span><p className="mt-2 text-sm font-medium text-slate-700">No secret access</p><p className="mt-1 text-xs text-slate-500">Create a narrow grant to activate this agent.</p></div>}{grants.map(grant => <div key={grant.id} className={`flex flex-wrap items-center justify-between gap-3 rounded-xl border px-4 py-3 transition-colors ${grant.revoked_at ? 'border-slate-100 bg-slate-50/60 opacity-65' : 'border-slate-200 bg-white hover:border-violet-200 hover:bg-violet-50/30'}`}><div className="flex items-center gap-3"><span className={`h-2 w-2 rounded-full ${grant.revoked_at ? 'bg-slate-300' : 'status-pulse bg-emerald-500 text-emerald-500'}`} /><div><p className="text-sm font-medium text-slate-800">{grant.environment?.project?.name ?? 'Project'} <span className="text-slate-300">/</span> {grant.environment?.name ?? grant.environment_id}</p><p className="mt-1 max-w-xl truncate text-xs text-slate-500">{grant.allow_all_secrets ? 'All secrets in environment' : grant.allowed_keys.join(', ')} · {grant.capability}</p></div></div>{grant.revoked_at ? <span className="text-xs font-medium text-slate-400">Revoked</span> : <button disabled={busyAction !== null} className="rounded-lg px-2.5 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 disabled:opacity-50" onClick={() => handleRevokeGrant(grant.id)}>{busyAction === `grant-${grant.id}` ? 'Revoking…' : 'Revoke grant'}</button>}</div>)}</div>
          </div></Card>
        </div> : <Card className="flex min-h-72 items-center justify-center border-dashed"><div className="p-8 text-center"><span className="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-100 text-slate-400"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7"><path d="M12 8V4H8"/><rect width="16" height="12" x="4" y="8" rx="2"/><path d="M2 14h2M20 14h2"/></svg></span><p className="mt-3 text-sm font-semibold text-slate-800">No agent selected</p><p className="mt-1 text-xs text-slate-500">Create an identity to start defining controlled access.</p></div></Card>}
      </div>
    </div>
  )
}
