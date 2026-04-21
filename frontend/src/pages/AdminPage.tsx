import { useEffect, useState } from 'react'
import { Card } from '../components/Card'
import { listAdminUsers, updateAdminUserTier, type User } from '../lib/api'

export function AdminPage() {
  const [users, setUsers] = useState<User[]>([])
  const [query, setQuery] = useState('')
  const [error, setError] = useState<string | null>(null)

  const load = () => {
    listAdminUsers(query).then(setUsers).catch((e) => setError((e as Error).message))
  }

  useEffect(() => { load() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const setTier = async (userId: string, tier: 'free' | 'starter' | 'team') => {
    try {
      await updateAdminUserTier(userId, tier)
      load()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Admin space</h1>
        <p className="mt-1 text-sm text-slate-500">Global controls for users and subscription tiers.</p>
      </div>
      <Card>
        <div className="flex gap-2">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search users by name/email"
            className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
          />
          <button className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white" onClick={load}>Search</button>
        </div>
      </Card>
      {error && <p className="text-sm text-red-600">{error}</p>}
      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-slate-200 text-xs text-slate-500 uppercase">
                <th className="py-2 pr-4">Name</th>
                <th className="py-2 pr-4">Email</th>
                <th className="py-2 pr-4">Tier</th>
                <th className="py-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id} className="border-b border-slate-100">
                  <td className="py-2 pr-4">{u.name}</td>
                  <td className="py-2 pr-4">{u.email}</td>
                  <td className="py-2 pr-4 capitalize">{u.tier}</td>
                  <td className="py-2 space-x-2">
                    <button className="rounded border px-2 py-1 text-xs" onClick={() => setTier(u.id, 'free')}>Free</button>
                    <button className="rounded border px-2 py-1 text-xs" onClick={() => setTier(u.id, 'starter')}>Starter</button>
                    <button className="rounded border px-2 py-1 text-xs" onClick={() => setTier(u.id, 'team')}>Team</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  )
}

