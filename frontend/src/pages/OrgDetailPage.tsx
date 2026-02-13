import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import {
  createProject,
  getCurrentUser,
  getOrg,
  listOrgProjects,
  type OrgDetail,
  type Project,
  type User,
} from '../lib/api'

export function OrgDetailPage() {
  const { id } = useParams()
  const nav = useNavigate()
  const [org, setOrg] = useState<OrgDetail | null>(null)
  const [projects, setProjects] = useState<Project[] | null>(null)
  const [user, setUser] = useState<User | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [projectName, setProjectName] = useState('')
  const [projectDesc, setProjectDesc] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const tierLimits: Record<string, number> = { free: 1, starter: 5, team: -1 }
  const maxProjects = user ? (tierLimits[user.tier] ?? 1) : 1
  const projectLimitReached = maxProjects !== -1 && projects !== null && projects.length >= maxProjects

  const load = useCallback(() => {
    if (!id) return
    getOrg(id).then(setOrg).catch((e) => setError((e as Error).message))
    listOrgProjects(id).then(setProjects).catch(() => setProjects([]))
    getCurrentUser().then(setUser).catch(() => {})
  }, [id])

  useEffect(() => { load() }, [load])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!id || !projectName.trim()) return
    setCreating(true)
    setCreateError(null)
    try {
      const project = await createProject(id, projectName.trim(), projectDesc.trim() || undefined)
      setProjectName('')
      setProjectDesc('')
      setShowCreate(false)
      load()
      nav(`/projects/${project.id}`)
    } catch (err) {
      setCreateError((err as Error).message)
    } finally {
      setCreating(false)
    }
  }

  if (error) return <p className="text-sm text-red-600">{error}</p>
  if (!org) return <p className="text-sm text-slate-500">Loading...</p>

  return (
    <div className="space-y-6">
      <div>
        <div className="text-sm text-slate-500">
          <Link to="/orgs" className="hover:text-slate-700">Organizations</Link>
          <span className="mx-1">/</span>
          <span className="font-medium text-slate-900">{org.name}</span>
        </div>
        <h1 className="mt-1 text-2xl font-bold text-slate-900">{org.name}</h1>
      </div>

      {/* Projects */}
      <div>
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-900">Projects</h2>
          <div className="flex items-center gap-2">
            {projects && maxProjects !== -1 && (
              <span className="text-xs text-slate-400">
                {projects.length}/{maxProjects}
              </span>
            )}
            {!projectLimitReached && (
              <Button onClick={() => setShowCreate(!showCreate)}>
                {showCreate ? 'Cancel' : 'New project'}
              </Button>
            )}
          </div>
        </div>

        {showCreate && (
          <Card className="mt-3">
            <form onSubmit={handleCreate} className="flex flex-wrap items-end gap-3">
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs font-medium text-slate-600">Name</span>
                <input
                  type="text"
                  value={projectName}
                  onChange={(e) => setProjectName(e.target.value)}
                  className="rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
                  placeholder="api-backend"
                  autoFocus
                />
              </label>
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs font-medium text-slate-600">Description (optional)</span>
                <input
                  type="text"
                  value={projectDesc}
                  onChange={(e) => setProjectDesc(e.target.value)}
                  className="rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-400 focus:outline-none focus:ring-1 focus:ring-slate-400"
                  placeholder="Backend API service"
                />
              </label>
              <Button type="submit" disabled={creating || !projectName.trim()}>
                {creating ? 'Creating...' : 'Create'}
              </Button>
            </form>
            {createError && <p className="mt-2 text-sm text-red-600">{createError}</p>}
          </Card>
        )}

        <div className="mt-3 space-y-2">
          {projects === null ? (
            <p className="text-sm text-slate-400">Loading...</p>
          ) : projects.length === 0 ? (
            <Card>
              <p className="py-4 text-center text-sm text-slate-500">
                No projects yet. Create one to get started.
              </p>
            </Card>
          ) : (
            projects.map((p) => (
              <Link key={p.id} to={`/projects/${p.id}`} className="block">
                <Card className="transition-colors hover:border-slate-300">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-sm font-medium text-slate-900">{p.name}</div>
                      {p.description && (
                        <div className="mt-0.5 text-xs text-slate-500">{p.description}</div>
                      )}
                    </div>
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
