import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import {
  getOrg,
  inviteMember,
  removeMember,
  updateMemberRole,
  type OrgDetail,
  type OrgMember,
} from '../lib/api'
import { getPermissions } from '../lib/auth'

const ROLES = ['Owner', 'Admin', 'Secret Manager', 'Developer', 'Viewer']

export function MembersPage() {
  const { id } = useParams()
  const perms = getPermissions()
  const canInvite = perms.includes('members.invite') || perms.includes('members.manage')
  const canManage = perms.includes('members.manage')
  const [org, setOrg] = useState<OrgDetail | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Invite form
  const [showInvite, setShowInvite] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState('Developer')
  const [inviting, setInviting] = useState(false)
  const [inviteError, setInviteError] = useState<string | null>(null)

  const load = useCallback(() => {
    if (!id) return
    getOrg(id).then(setOrg).catch((e) => setError((e as Error).message))
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
      load()
    } catch (err) {
      setInviteError((err as Error).message)
    } finally {
      setInviting(false)
    }
  }

  const handleRoleChange = async (memberId: string, role: string) => {
    if (!id) return
    try {
      await updateMemberRole(id, memberId, role)
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
      load()
    } catch (err) {
      alert((err as Error).message)
    }
  }

  if (error) return <p className="text-sm text-red-600">{error}</p>
  if (!org) return <p className="text-sm text-slate-500">Loading...</p>

  const members = org.members ?? []

  return (
    <div className="space-y-6">
      <div>
        <div className="text-sm text-slate-500">
          <Link to="/orgs" className="hover:text-slate-700">Organizations</Link>
          <span className="mx-1">/</span>
          <Link to={`/orgs/${org.id}`} className="hover:text-slate-700">{org.name}</Link>
          <span className="mx-1">/</span>
          <span className="font-medium text-slate-900">Team</span>
        </div>
        <h1 className="mt-1 text-2xl font-bold text-slate-900">Team Members</h1>
      </div>

      {canInvite && (
        <div className="flex justify-end">
          <Button onClick={() => setShowInvite(!showInvite)}>
            {showInvite ? 'Cancel' : 'Invite member'}
          </Button>
        </div>
      )}

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
          {inviteError && <p className="mt-2 text-sm text-red-600">{inviteError}</p>}
        </Card>
      )}

      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-slate-100 text-xs text-slate-500">
                <th className="pb-2 pr-4 font-medium">Name</th>
                <th className="pb-2 pr-4 font-medium">Email</th>
                <th className="pb-2 pr-4 font-medium">Role</th>
                {canManage && <th className="pb-2 font-medium">Actions</th>}
              </tr>
            </thead>
            <tbody>
              {members.map((m: OrgMember) => (
                <tr key={m.id} className="border-b border-slate-50">
                  <td className="py-2.5 pr-4 text-slate-900">{m.user?.name ?? '-'}</td>
                  <td className="py-2.5 pr-4 text-slate-600">{m.user?.email ?? '-'}</td>
                  <td className="py-2.5 pr-4">
                    {canManage ? (
                      <select
                        value={m.role?.name ?? 'Developer'}
                        onChange={(e) => handleRoleChange(m.id, e.target.value)}
                        className="rounded border border-slate-200 px-2 py-1 text-xs"
                      >
                        {ROLES.map((r) => (
                          <option key={r} value={r}>{r}</option>
                        ))}
                      </select>
                    ) : (
                      <span className="text-slate-600">{m.role?.name ?? '-'}</span>
                    )}
                  </td>
                  {canManage && (
                    <td className="py-2.5">
                      <button
                        onClick={() => handleRemove(m.id, m.user?.name ?? m.user?.email ?? 'member')}
                        className="text-xs text-red-500 hover:text-red-700"
                      >
                        Remove
                      </button>
                    </td>
                  )}
                </tr>
              ))}
              {members.length === 0 && (
                <tr>
                  <td colSpan={canManage ? 4 : 3} className="py-6 text-center text-slate-400">
                    No members
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
