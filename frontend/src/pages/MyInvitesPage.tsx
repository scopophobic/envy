import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { acceptMyInvitation, listMyInvitations, type OrgInvitation } from '../lib/api'
import { Card } from '../components/Card'

export function MyInvitesPage() {
  const [invites, setInvites] = useState<OrgInvitation[]>([])
  const [error, setError] = useState<string | null>(null)
  const [processingId, setProcessingId] = useState<string | null>(null)

  const load = () => {
    setError(null)
    listMyInvitations().then(setInvites).catch((e) => setError((e as Error).message))
  }

  useEffect(() => {
    load()
  }, [])

  const accept = async (invite: OrgInvitation) => {
    try {
      setProcessingId(invite.id)
      await acceptMyInvitation(invite.id)
      load()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setProcessingId(null)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">My invitations</h1>
        <p className="mt-1 text-sm text-slate-500">Pending workspace invites for your account email.</p>
      </div>

      {error && <p className="text-sm text-red-600">{error}</p>}

      <Card>
        {invites.length === 0 ? (
          <p className="text-sm text-slate-500">No pending invites right now.</p>
        ) : (
          <div className="space-y-3">
            {invites.map((invite) => (
              <div key={invite.id} className="rounded-lg border border-slate-200 p-4">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="text-sm font-semibold text-slate-900">
                      {invite.organization?.name || 'Workspace invite'}
                    </p>
                    <p className="text-xs text-slate-500">
                      Role: {invite.role?.name || 'member'} • Expires {new Date(invite.expires_at).toLocaleString()}
                    </p>
                  </div>
                  <button
                    onClick={() => accept(invite)}
                    disabled={processingId === invite.id}
                    className="rounded-md bg-violet-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-60"
                  >
                    {processingId === invite.id ? 'Accepting...' : 'Accept'}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      <Link to="/orgs" className="inline-flex text-sm text-violet-700 hover:underline">
        Back to dashboard
      </Link>
    </div>
  )
}
