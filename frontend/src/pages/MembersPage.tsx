import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import {
  getOrg,
  getTierInfo,
  inviteMember,
  removeMember,
  updateMemberRole,
  type OrgDetail,
  type OrgMember,
  type TierInfo,
} from '../lib/api'

const ROLES = ['Owner', 'Admin', 'Secret Manager', 'Developer', 'Viewer']

export function MembersPage() {
  const { id } = useParams()
  const [org, setOrg] = useState<OrgDetail | null>(null)
  const [tierInfo, setTierInfo] = useState<TierInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [toast, setToast] = useState<string | null>(null)

  // Invite form
  const [showInvite, setShowInvite] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState('Developer')
  const [inviting, setInviting] = useState(false)
  const [inviteError, setInviteError] = useState<string | null>(null)

  const showToast = (msg: string) => {
    setToast(msg)
    setTimeout(() => setToast(null), 3000)
  }

  const load = useCallback(() => {
    if (!id) return
    getOrg(id).then(setOrg).catch((e) => setError((e as Error).message))
    getTierInfo().then(setTierInfo).catch(() => {})
  }, [id])

  useEffect(() => { load() }, [load])

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!id || !inviteEmail.trim()) return
    setInviting(true)
    setInviteError(null)
    try {
      await inviteMember(id, inviteEmail.trim(), inviteRole)
      setInviteEmail('')
      setShowInvite(false)
      showToast(`Invited ${inviteEmail.trim()}`)
      load()
    } catch (err) {
      setInviteError((err as Error).message)
    } finally {
      setInviting(false)
    }
  }

  const handleRoleChange = async (memberId: string, role: string, name: string) => {
    if (!id) return
    try {
      await updateMemberRole(id, memberId, role)
      showToast(`Updated ${name} to ${role}`)
      load()
    } catch (err) {
      alert((err as Error).message)
    }
  }

  const handleRemove = async (memberId: string, name: string) => {
    if (!id) return
    if (!confirm(`Remove ${name} from this organization?`)) return
    try {
      await removeMember(id, memberId)
      showToast(`Removed ${name}`)
      load()
    } catch (err) {
      alert((err as Error).message)
    }
  }

  if (error) return <p className="text-sm text-red-600 p-6">{error}</p>
  if (!org) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-slate-300 border-t-slate-600" />
        <span className="ml-2 text-sm text-slate-500">Loading...</span>
      </div>
    )
  }

  const members = org.members ?? []
  const maxDevs = tierInfo?.limits.max_devs_per_org ?? -1
  const memberLimitReached = maxDevs !== -1 && members.length >= maxDevs

  return (
    <div className="space-y-6">
      {/* Toast */}
      {toast && (
        <div className="fixed top-4 right-4 z-50 rounded-lg bg-slate-900 px-4 py-2.5 text-sm text-white shadow-lg">
          {toast}
        </div>
      )}

      <div>
        <div className="text-sm text-slate-500">
          <Link to="/orgs" className="hover:text-slate-700">Organizations</Link>
          <span className="mx-1">/</span>
          <Link to={`/orgs/${org.id}`} className="hover:text-slate-700">{org.name}</Link>
          <span className="mx-1">/</span>
          <span className="font-medium text-slate-900">Team</span>
        </div>
        <h1 className="mt-1 text-2xl font-bold text-slate-900">Team Members</h1>
        <p className="mt-1 text-sm text-slate-500">
          Manage team access and roles for {org.name}.
        </p>
      </div>

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-sm text-slate-500">
            {members.length} member{members.length !== 1 ? 's' : ''}
          </span>
          {maxDevs !== -1 && (
            <span className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${memberLimitReached ? 'bg-red-50 text-red-600' : 'bg-slate-100 text-slate-500'}`}>
              / {maxDevs} max
            </span>
          )}
        </div>
        {!memberLimitReached ? (
          <Button onClick={() => setShowInvite(!showInvite)}>
            {showInvite ? 'Cancel' : 'Invite member'}
          </Button>
        ) : (
          <span className="rounded-md bg-amber-50 border border-amber-200 px-3 py-1.5 text-xs text-amber-700">
            Member limit reached — upgrade plan
          </span>
        )}
      </div>

      {showInvite && (
        <Card>
          <form onSubmit={handleInvite} className="flex flex-wrap items-end gap-3">
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-slate-600">Email</span>
              <input
                type="email"
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
                className="rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
                placeholder="colleague@company.com"
                autoFocus
              />
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-slate-600">Role</span>
              <select
                value={inviteRole}
                onChange={(e) => setInviteRole(e.target.value)}
                className="rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
              >
                {ROLES.map((r) => (
                  <option key={r} value={r}>{r}</option>
                ))}
              </select>
            </label>
            <Button type="submit" disabled={inviting || !inviteEmail.trim()}>
              {inviting ? 'Inviting...' : 'Invite'}
            </Button>
          </form>
          <p className="mt-2 text-xs text-slate-400">The user must have signed in to Envo at least once.</p>
          {inviteError && <p className="mt-2 text-sm text-red-600">{inviteError}</p>}
        </Card>
      )}

      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-slate-200 text-xs text-slate-500 uppercase tracking-wide">
                <th className="pb-2.5 pr-4 font-medium">Name</th>
                <th className="pb-2.5 pr-4 font-medium">Email</th>
                <th className="pb-2.5 pr-4 font-medium">Role</th>
                <th className="pb-2.5 font-medium w-20">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {members.map((m: OrgMember) => {
                const displayName = m.user?.name ?? m.user?.email ?? '-'
                return (
                  <tr key={m.id} className="group hover:bg-slate-50/50">
                    <td className="py-2.5 pr-4">
                      <div className="flex items-center gap-2">
                        <div className="flex h-7 w-7 items-center justify-center rounded-full bg-slate-100 text-[11px] font-medium text-slate-600">
                          {(m.user?.name ?? m.user?.email ?? '?')[0].toUpperCase()}
                        </div>
                        <span className="text-slate-900">{m.user?.name ?? '-'}</span>
                      </div>
                    </td>
                    <td className="py-2.5 pr-4 text-slate-600">{m.user?.email ?? '-'}</td>
                    <td className="py-2.5 pr-4">
                      <select
                        value={m.role?.name ?? 'Developer'}
                        onChange={(e) => handleRoleChange(m.id, e.target.value, displayName)}
                        className="rounded border border-slate-200 bg-white px-2 py-1 text-xs focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
                      >
                        {ROLES.map((r) => (
                          <option key={r} value={r}>{r}</option>
                        ))}
                      </select>
                    </td>
                    <td className="py-2.5">
                      <button
                        onClick={() => handleRemove(m.id, displayName)}
                        className="rounded px-2 py-1 text-xs text-red-500 opacity-0 group-hover:opacity-100 hover:bg-red-50 transition-all"
                      >
                        Remove
                      </button>
                    </td>
                  </tr>
                )
              })}
              {members.length === 0 && (
                <tr>
                  <td colSpan={4} className="py-8 text-center text-sm text-slate-400">
                    No members yet. Invite your first team member above.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  )
}
