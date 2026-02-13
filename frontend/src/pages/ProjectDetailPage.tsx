import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import {
  createEnvironment,
  getProject,
  listProjectEnvironments,
  type Environment,
  type Project,
} from '../lib/api'

export function ProjectDetailPage() {
  const { id } = useParams()
  const nav = useNavigate()
  const [project, setProject] = useState<Project | null>(null)
  const [envs, setEnvs] = useState<Environment[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [envName, setEnvName] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const load = useCallback(() => {
    if (!id) return
    getProject(id).then(setProject).catch((e) => setError((e as Error).message))
    listProjectEnvironments(id).then(setEnvs).catch(() => setEnvs([]))
  }, [id])

  useEffect(() => { load() }, [load])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!id || !envName.trim()) return
    setCreating(true)
    setCreateError(null)
    try {
      const env = await createEnvironment(id, envName.trim())
      setEnvName('')
      setShowCreate(false)
      load()
      nav(`/environments/${env.id}`)
    } catch (err) {
      setCreateError((err as Error).message)
    } finally {
      setCreating(false)
    }
  }

  if (error) return <p className="text-sm text-red-600">{error}</p>
  if (!project) return <p className="text-sm text-slate-500">Loading...</p>

  return (
    <div className="space-y-6">
      <div>
        <div className="text-sm text-slate-500">
          <Link to="/orgs" className="hover:text-slate-700">Organizations</Link>
          <span className="mx-1">/</span>
          <Link to={`/orgs/${project.org_id}`} className="hover:text-slate-700">Org</Link>
          <span className="mx-1">/</span>
          <span className="font-medium text-slate-900">{project.name}</span>
        </div>
        <h1 className="mt-1 text-2xl font-bold text-slate-900">{project.name}</h1>
        {project.description && (
          <p className="mt-1 text-sm text-slate-500">{project.description}</p>
        )}
      </div>

      {/* Environments */}
      <div>
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-900">Environments</h2>
          <Button onClick={() => setShowCreate(!showCreate)}>
            {showCreate ? 'Cancel' : 'New environment'}
          </Button>
        </div>

        {showCreate && (
          <Card className="mt-3">
            <form onSubmit={handleCreate} className="flex items-end gap-3">
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs font-medium text-slate-600">Name</span>
                <input
                  type="text"
                  value={envName}
                  onChange={(e) => setEnvName(e.target.value)}
                  className="rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
                  placeholder="production"
                  autoFocus
                />
              </label>
              <Button type="submit" disabled={creating || !envName.trim()}>
                {creating ? 'Creating...' : 'Create'}
              </Button>
            </form>
            {createError && <p className="mt-2 text-sm text-red-600">{createError}</p>}
          </Card>
        )}

        <div className="mt-3 space-y-2">
          {envs === null ? (
            <p className="text-sm text-slate-400">Loading...</p>
          ) : envs.length === 0 ? (
            <Card>
              <p className="py-4 text-center text-sm text-slate-500">
                No environments yet. Common names: development, staging, production.
              </p>
            </Card>
          ) : (
            envs.map((env) => (
              <Link key={env.id} to={`/environments/${env.id}`} className="block">
                <Card className="transition-colors hover:border-slate-300">
                  <div className="flex items-center justify-between">
                    <div className="text-sm font-medium text-slate-900">{env.name}</div>
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-slate-400">
                      <path d="M9 18l6-6-6-6" />
                    </svg>
                  </div>
                </Card>
              </Link>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
