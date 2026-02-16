import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import {
  createSecret,
  deleteSecret,
  getEnvironment,
  listSecrets,
  updateSecret,
  type EnvironmentDetail,
  type Secret,
} from '../lib/api'

export function EnvironmentDetailPage() {
  const { id: envId } = useParams()
  const [env, setEnv] = useState<EnvironmentDetail | null>(null)
  const [secrets, setSecrets] = useState<Secret[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

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

  // Toast
  const [toast, setToast] = useState<string | null>(null)
  const showToast = (msg: string) => {
    setToast(msg)
    setTimeout(() => setToast(null), 3000)
  }

  const loadSecrets = useCallback(async () => {
    if (!envId) return
    try {
      const secretsList = await listSecrets(envId)
      setSecrets(secretsList)
    } catch (e) {
      setError((e as Error).message)
    }
  }, [envId])

  useEffect(() => {
    if (!envId) return
    getEnvironment(envId).then(setEnv).catch(() => {})
    loadSecrets()
  }, [envId, loadSecrets])

  const cliCommand = env
    ? `envo pull --org "${env.org_name}" --project "${env.project_name}" --env "${env.name}"`
    : `envo pull --org "<org>" --project "<project>" --env "<env>"`

  const handleCopyCmd = () => {
    navigator.clipboard.writeText(cliCommand)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

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
      showToast(`Secret ${newKey.trim()} added`)
      loadSecrets()
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
      showToast(`Imported ${created} secret${created !== 1 ? 's' : ''}`)
      loadSecrets()
      if (created === 0) {
        setBulkError('No valid KEY=VALUE pairs found')
      }
    } catch (err) {
      setBulkError((err as Error).message)
      loadSecrets()
    } finally {
      setBulkImporting(false)
    }
  }

  const handleEdit = async (secretId: string) => {
    try {
      const keyToSend = editKey.trim() || undefined
      const valueToSend = editValue.trim() || undefined
      if (!keyToSend && !valueToSend) return
      await updateSecret(secretId, keyToSend, valueToSend)
      setEditingId(null)
      setEditKey('')
      setEditValue('')
      showToast('Secret updated')
      loadSecrets()
    } catch (err) {
      alert((err as Error).message)
    }
  }

  const handleDelete = async (secretId: string, key: string) => {
    if (!confirm(`Delete secret "${key}"?`)) return
    try {
      await deleteSecret(secretId)
      showToast(`Secret ${key} deleted`)
      loadSecrets()
    } catch (err) {
      alert((err as Error).message)
    }
  }

  if (error) return <p className="text-sm text-red-600 p-6">{error}</p>

  return (
    <div className="space-y-6">
      {/* Toast */}
      {toast && (
        <div className="fixed top-4 right-4 z-50 rounded-lg bg-slate-900 px-4 py-2.5 text-sm text-white shadow-lg">
          {toast}
        </div>
      )}

      {/* Breadcrumbs + header */}
      <div>
        <div className="text-sm text-slate-500">
          <Link to="/orgs" className="hover:text-slate-700">Organizations</Link>
          {env && (
            <>
              <span className="mx-1">/</span>
              <Link to={`/orgs/${env.org_id}`} className="hover:text-slate-700">{env.org_name}</Link>
              <span className="mx-1">/</span>
              <Link to={`/projects/${env.project_id}`} className="hover:text-slate-700">{env.project_name}</Link>
              <span className="mx-1">/</span>
              <span className="font-medium text-slate-900">{env.name}</span>
            </>
          )}
        </div>
        <h1 className="mt-1 text-2xl font-bold text-slate-900">
          {env ? `${env.name} — Secrets` : 'Secrets'}
        </h1>
        <p className="mt-1 text-sm text-slate-500">
          Secrets are encrypted at rest. Use the CLI command below to pull them into your project.
        </p>
      </div>

      {/* CLI command — copy-pasteable */}
      <Card className="border-violet-100 bg-violet-50/50">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-wide text-violet-600">
            Pull secrets via CLI
          </div>
          <div className="mt-1.5 flex items-center gap-2 rounded-md bg-white border border-violet-200 px-3 py-2">
            <code className="flex-1 truncate text-[13px] text-violet-900 font-mono select-all">
              {cliCommand}
            </code>
            <button
              onClick={handleCopyCmd}
              className="shrink-0 rounded border border-violet-200 bg-violet-50 px-2.5 py-1 text-xs font-medium text-violet-700 hover:bg-violet-100 transition-colors"
            >
              {copied ? 'Copied!' : 'Copy'}
            </button>
          </div>
          <p className="mt-1.5 text-[11px] text-violet-500">
            Writes <code className="font-medium">.env</code> to whichever directory your terminal is open in.
          </p>
        </div>
      </Card>

      {/* Actions */}
      <div className="flex gap-2">
        <Button onClick={() => { setShowAdd(!showAdd); setShowBulk(false) }}>
          {showAdd ? 'Cancel' : 'Add secret'}
        </Button>
        <button
          onClick={() => { setShowBulk(!showBulk); setShowAdd(false) }}
          className="rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
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
                className="rounded-md border border-slate-300 px-3 py-2 text-sm font-mono focus:border-violet-400 focus:outline-none focus:ring-1 focus:ring-violet-400"
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
                className="rounded-md border border-slate-300 px-3 py-2 text-sm font-mono focus:border-violet-400 focus:outline-none focus:ring-1 focus:ring-violet-400"
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
                className="h-40 rounded-md border border-slate-300 px-3 py-2 font-mono text-sm focus:border-violet-400 focus:outline-none focus:ring-1 focus:ring-violet-400"
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
          <div className="flex items-center justify-center py-8">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-slate-300 border-t-slate-600" />
            <span className="ml-2 text-sm text-slate-500">Loading secrets...</span>
          </div>
        ) : secrets.length === 0 ? (
          <div className="py-10 text-center">
            <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-slate-100">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="text-slate-400">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                <path d="M7 11V7a5 5 0 0 1 10 0v4" />
              </svg>
            </div>
            <p className="text-sm font-medium text-slate-700">No secrets yet</p>
            <p className="mt-1 text-xs text-slate-500">Add your first secret above, or use bulk import to paste a .env file.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500 uppercase tracking-wide">
                  <th className="pb-2.5 pr-4 font-medium">Key</th>
                  <th className="pb-2.5 pr-4 font-medium">Value</th>
                  <th className="pb-2.5 font-medium w-32">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-50">
                {secrets.map((s) => (
                  <tr key={s.id} className="group hover:bg-slate-50/50">
                    {editingId === s.id ? (
                      <>
                        <td className="py-2.5 pr-4">
                          <input
                            type="text"
                            value={editKey}
                            onChange={(e) => setEditKey(e.target.value.toUpperCase())}
                            className="w-full rounded border border-slate-300 px-2 py-1.5 font-mono text-sm focus:border-violet-400 focus:outline-none focus:ring-1 focus:ring-violet-400"
                            placeholder="KEY"
                          />
                        </td>
                        <td className="py-2.5 pr-4">
                          <input
                            type="text"
                            value={editValue}
                            onChange={(e) => setEditValue(e.target.value)}
                            className="w-full rounded border border-slate-300 px-2 py-1.5 font-mono text-sm focus:border-violet-400 focus:outline-none focus:ring-1 focus:ring-violet-400"
                            placeholder="new value (leave empty to keep current)"
                          />
                        </td>
                        <td className="py-2.5">
                          <div className="flex gap-2">
                            <button
                              onClick={() => handleEdit(s.id)}
                              className="rounded bg-slate-900 px-2.5 py-1 text-xs font-medium text-white hover:bg-slate-700 transition-colors"
                            >
                              Save
                            </button>
                            <button
                              onClick={() => { setEditingId(null); setEditKey(''); setEditValue('') }}
                              className="text-xs text-slate-500 hover:text-slate-700"
                            >
                              Cancel
                            </button>
                          </div>
                        </td>
                      </>
                    ) : (
                      <>
                        <td className="py-2.5 pr-4 font-mono font-medium text-slate-900">{s.key}</td>
                        <td className="py-2.5 pr-4 font-mono text-slate-400 tracking-wider">
                          {'*'.repeat(16)}
                        </td>
                        <td className="py-2.5">
                          <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                            <button
                              onClick={() => {
                                setEditingId(s.id)
                                setEditKey(s.key)
                                setEditValue('')
                              }}
                              className="rounded px-2 py-1 text-xs text-slate-600 hover:bg-slate-100 transition-colors"
                            >
                              Edit
                            </button>
                            <button
                              onClick={() => handleDelete(s.id, s.key)}
                              className="rounded px-2 py-1 text-xs text-red-500 hover:bg-red-50 transition-colors"
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
            {secrets.length} secret{secrets.length !== 1 ? 's' : ''} — encrypted at rest
          </div>
        )}
      </Card>
    </div>
  )
}
