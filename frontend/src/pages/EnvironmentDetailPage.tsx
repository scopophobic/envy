import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import {
  createSecret,
  deleteSecret,
  exportSecrets,
  getProject,
  listProjectEnvironments,
  listSecrets,
  updateSecret,
  type Environment,
  type Project,
  type Secret,
} from '../lib/api'

export function EnvironmentDetailPage() {
  const { id: envId } = useParams()
  const [env, setEnv] = useState<Environment | null>(null)
  const [project, setProject] = useState<Project | null>(null)
  const [secrets, setSecrets] = useState<Secret[] | null>(null)
  const [decrypted, setDecrypted] = useState<Record<string, string>>({})
  const [revealed, setRevealed] = useState<Set<string>>(new Set())
  const [error, setError] = useState<string | null>(null)

  // Add secret
  const [showAdd, setShowAdd] = useState(false)
  const [newKey, setNewKey] = useState('')
  const [newValue, setNewValue] = useState('')
  const [adding, setAdding] = useState(false)
  const [addError, setAddError] = useState<string | null>(null)

  // Bulk import
  const [showBulk, setShowBulk] = useState(false)
  const [bulkText, setBulkText] = useState('')
  const [bulkImporting, setBulkImporting] = useState(false)
  const [bulkError, setBulkError] = useState<string | null>(null)

  // Edit
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editKey, setEditKey] = useState('')
  const [editValue, setEditValue] = useState('')

  const load = useCallback(async () => {
    if (!envId) return
    try {
      // Find the environment by listing all envs for the project
      // We need to find which project this env belongs to
      // First get env list from the secrets endpoint
      const secretsList = await listSecrets(envId)
      setSecrets(secretsList)

      // Try to load decrypted values
      try {
        const exported = await exportSecrets(envId)
        setDecrypted(exported)
      } catch {
        // May fail if no KMS or no permission
        setDecrypted({})
      }
    } catch (e) {
      setError((e as Error).message)
    }
  }, [envId])

  // Load environment and project info
  useEffect(() => {
    if (!envId) return
    // We need a way to get env info. List all envs for a project.
    // For now, store env info from the URL params and load secrets.
    load()
  }, [envId, load])

  // Try to find env and project info from list endpoints
  useEffect(() => {
    if (!envId) return
    // Use a discovery approach: the env id is known, but we need project info
    // We'll extract it from the secrets or try loading project
    const findEnvInfo = async () => {
      try {
        // There's no direct "get environment" endpoint, so we need to
        // discover it. We can try to get the env from the export endpoint
        // or just show the env ID. For a better UX, let's store env+project
        // in the navigation state. For now, we'll set basic info.
        setEnv({ id: envId, project_id: '', name: envId })
      } catch {
        // ignore
      }
    }
    findEnvInfo()
  }, [envId])

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!envId || !newKey.trim() || !newValue.trim()) return
    setAdding(true)
    setAddError(null)
    try {
      await createSecret(envId, newKey.trim(), newValue.trim())
      setNewKey('')
      setNewValue('')
      setShowAdd(false)
      load()
    } catch (err) {
      setAddError((err as Error).message)
    } finally {
      setAdding(false)
    }
  }

  const handleBulkImport = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!envId || !bulkText.trim()) return
    setBulkImporting(true)
    setBulkError(null)
    try {
      const lines = bulkText.split('\n').filter((l) => l.trim() && !l.trim().startsWith('#'))
      let created = 0
      for (const line of lines) {
        const eqIndex = line.indexOf('=')
        if (eqIndex === -1) continue
        const key = line.substring(0, eqIndex).trim()
        const value = line.substring(eqIndex + 1).trim()
        if (!key) continue
        await createSecret(envId, key, value)
        created++
      }
      setBulkText('')
      setShowBulk(false)
      load()
      if (created === 0) {
        setBulkError('No valid KEY=VALUE pairs found')
      }
    } catch (err) {
      setBulkError((err as Error).message)
      load() // Reload to show what was created before the error
    } finally {
      setBulkImporting(false)
    }
  }

  const handleEdit = async (secretId: string) => {
    try {
      await updateSecret(secretId, editKey.trim() || undefined, editValue.trim() || undefined)
      setEditingId(null)
      load()
    } catch (err) {
      alert((err as Error).message)
    }
  }

  const handleDelete = async (secretId: string, key: string) => {
    if (!confirm(`Delete secret "${key}"?`)) return
    try {
      await deleteSecret(secretId)
      load()
    } catch (err) {
      alert((err as Error).message)
    }
  }

  const toggleReveal = (secretId: string) => {
    setRevealed((prev) => {
      const next = new Set(prev)
      if (next.has(secretId)) {
        next.delete(secretId)
      } else {
        next.add(secretId)
      }
      return next
    })
  }

  const copyCliCommand = () => {
    // Build a generic CLI command
    const cmd = `envo pull --org "<org-name>" --project "<project-name>" --env "<env-name>"`
    navigator.clipboard.writeText(cmd)
  }

  if (error) return <p className="text-sm text-red-600">{error}</p>

  return (
    <div className="space-y-6">
      <div>
        <div className="text-sm text-slate-500">
          <Link to="/orgs" className="hover:text-slate-700">Organizations</Link>
          <span className="mx-1">/</span>
          <span className="font-medium text-slate-900">Environment</span>
        </div>
        <h1 className="mt-1 text-2xl font-bold text-slate-900">Secrets</h1>
        <p className="mt-1 text-sm text-slate-500">
          Manage environment variables for this environment.
        </p>
      </div>

      {/* CLI hint */}
      <Card className="border-blue-100 bg-blue-50">
        <div className="flex items-start justify-between gap-3">
          <div>
            <div className="text-xs font-semibold text-blue-800">Pull secrets via CLI</div>
            <code className="mt-1 block text-xs text-blue-700">
              envo pull --org "&lt;org&gt;" --project "&lt;project&gt;" --env "&lt;env&gt;"
            </code>
          </div>
          <button
            onClick={copyCliCommand}
            className="shrink-0 rounded border border-blue-200 px-2 py-1 text-xs text-blue-700 hover:bg-blue-100"
          >
            Copy
          </button>
        </div>
      </Card>

      {/* Actions */}
      <div className="flex gap-2">
        <Button onClick={() => { setShowAdd(!showAdd); setShowBulk(false) }}>
          {showAdd ? 'Cancel' : 'Add secret'}
        </Button>
        <button
          onClick={() => { setShowBulk(!showBulk); setShowAdd(false) }}
          className="rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
        >
          {showBulk ? 'Cancel' : 'Bulk import'}
        </button>
      </div>

      {/* Add single secret */}
      {showAdd && (
        <Card>
          <form onSubmit={handleAdd} className="flex flex-wrap items-end gap-3">
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-slate-600">Key</span>
              <input
                type="text"
                value={newKey}
                onChange={(e) => setNewKey(e.target.value.toUpperCase())}
                className="rounded-md border border-slate-300 px-3 py-2 text-sm font-mono focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
                placeholder="DATABASE_URL"
                autoFocus
              />
            </label>
            <label className="flex flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-slate-600">Value</span>
              <input
                type="text"
                value={newValue}
                onChange={(e) => setNewValue(e.target.value)}
                className="rounded-md border border-slate-300 px-3 py-2 text-sm font-mono focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
                placeholder="postgres://..."
              />
            </label>
            <Button type="submit" disabled={adding || !newKey.trim() || !newValue.trim()}>
              {adding ? 'Adding...' : 'Add'}
            </Button>
          </form>
          {addError && <p className="mt-2 text-sm text-red-600">{addError}</p>}
        </Card>
      )}

      {/* Bulk import */}
      {showBulk && (
        <Card>
          <form onSubmit={handleBulkImport}>
            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-slate-600">
                Paste .env format (KEY=value, one per line)
              </span>
              <textarea
                value={bulkText}
                onChange={(e) => setBulkText(e.target.value)}
                className="h-40 rounded-md border border-slate-300 px-3 py-2 font-mono text-sm focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
                placeholder={`DATABASE_URL=postgres://localhost:5432/mydb\nREDIS_URL=redis://localhost:6379\nAPI_KEY=sk-abc123`}
                autoFocus
              />
            </label>
            <div className="mt-3 flex items-center gap-3">
              <Button type="submit" disabled={bulkImporting || !bulkText.trim()}>
                {bulkImporting ? 'Importing...' : 'Import all'}
              </Button>
              <span className="text-xs text-slate-400">
                Lines starting with # are ignored
              </span>
            </div>
          </form>
          {bulkError && <p className="mt-2 text-sm text-red-600">{bulkError}</p>}
        </Card>
      )}

      {/* Secrets list */}
      <Card>
        {secrets === null ? (
          <p className="py-4 text-center text-sm text-slate-400">Loading secrets...</p>
        ) : secrets.length === 0 ? (
          <p className="py-8 text-center text-sm text-slate-500">
            No secrets yet. Add one above or use bulk import.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-100 text-xs text-slate-500">
                  <th className="pb-2 pr-4 font-medium">Key</th>
                  <th className="pb-2 pr-4 font-medium">Value</th>
                  <th className="pb-2 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {secrets.map((s) => (
                  <tr key={s.id} className="border-b border-slate-50">
                    {editingId === s.id ? (
                      <>
                        <td className="py-2 pr-4">
                          <input
                            type="text"
                            value={editKey}
                            onChange={(e) => setEditKey(e.target.value.toUpperCase())}
                            className="w-full rounded border border-slate-300 px-2 py-1 font-mono text-sm"
                          />
                        </td>
                        <td className="py-2 pr-4">
                          <input
                            type="text"
                            value={editValue}
                            onChange={(e) => setEditValue(e.target.value)}
                            className="w-full rounded border border-slate-300 px-2 py-1 font-mono text-sm"
                          />
                        </td>
                        <td className="py-2">
                          <div className="flex gap-2">
                            <button
                              onClick={() => handleEdit(s.id)}
                              className="text-xs text-green-600 hover:text-green-800"
                            >
                              Save
                            </button>
                            <button
                              onClick={() => setEditingId(null)}
                              className="text-xs text-slate-500 hover:text-slate-700"
                            >
                              Cancel
                            </button>
                          </div>
                        </td>
                      </>
                    ) : (
                      <>
                        <td className="py-2.5 pr-4 font-mono text-slate-900">{s.key}</td>
                        <td className="py-2.5 pr-4 font-mono text-slate-600">
                          {revealed.has(s.id) ? (
                            <span>{decrypted[s.key] ?? '(encrypted)'}</span>
                          ) : (
                            <span className="text-slate-400">{'*'.repeat(12)}</span>
                          )}
                          <button
                            onClick={() => toggleReveal(s.id)}
                            className="ml-2 text-xs text-slate-400 hover:text-slate-600"
                          >
                            {revealed.has(s.id) ? 'hide' : 'show'}
                          </button>
                        </td>
                        <td className="py-2.5">
                          <div className="flex gap-2">
                            <button
                              onClick={() => {
                                setEditingId(s.id)
                                setEditKey(s.key)
                                setEditValue(decrypted[s.key] ?? '')
                              }}
                              className="text-xs text-slate-500 hover:text-slate-700"
                            >
                              Edit
                            </button>
                            <button
                              onClick={() => handleDelete(s.id, s.key)}
                              className="text-xs text-red-500 hover:text-red-700"
                            >
                              Delete
                            </button>
                          </div>
                        </td>
                      </>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        {secrets && secrets.length > 0 && (
          <div className="mt-3 border-t border-slate-100 pt-3 text-xs text-slate-400">
            {secrets.length} secret{secrets.length !== 1 ? 's' : ''}
          </div>
        )}
      </Card>
    </div>
  )
}
