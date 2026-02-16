import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Button } from '../components/Button'
import { Card } from '../components/Card'
import {
  createProject,
  getOrg,
  getTierInfo,
  listOrgProjects,
  type OrgDetail,
  type Project,
  type TierInfo,
} from '../lib/api'

export function OrgDetailPage() {
  const { id } = useParams()
  const nav = useNavigate()
  const [org, setOrg] = useState<OrgDetail | null>(null)
  const [projects, setProjects] = useState<Project[] | null>(null)
  const [tierInfo, setTierInfo] = useState<TierInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [projectName, setProjectName] = useState('')
  const [projectDesc, setProjectDesc] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const maxProjects = tierInfo?.limits.max_projects_per_org ?? 1
  const projectCount = projects?.length ?? 0
  const projectLimitReached = maxProjects !== -1 && projectCount >= maxProjects

  const load = useCallback(() => {
    if (!id) return
    getOrg(id).then(setOrg).catch((e) => setError((e as Error).message))
    listOrgProjects(id).then(setProjects).catch(() => setProjects([]))
    getTierInfo().then(setTierInfo).catch(() => {})
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

  if (error) return <p className="text-sm text-red-600 p-6">{error}</p>
  if (!org) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-slate-300 border-t-slate-600" />
        <span className="ml-2 text-sm text-slate-500">Loading...</span>
      </div>
    )
  }

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
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-semibold text-slate-900">Projects</h2>
            {projects && maxProjects !== -1 && (
              <span className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${projectLimitReached ? 'bg-red-50 text-red-600' : 'bg-slate-100 text-slate-500'}`}>
                {projectCount}/{maxProjects}
              </span>
            )}
            {maxProjects === -1 && projects && (
              <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-medium text-slate-500">
                {projectCount}
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            {!projectLimitReached ? (
              <Button onClick={() => setShowCreate(!showCreate)}>
                {showCreate ? 'Cancel' : 'New project'}
              </Button>
            ) : (
              <span className="rounded-md bg-amber-50 border border-amber-200 px-3 py-1.5 text-xs text-amber-700">
                Project limit reached — upgrade plan
              </span>
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
            <div className="flex items-center justify-center py-8">
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-slate-300 border-t-slate-600" />
              <span className="ml-2 text-sm text-slate-500">Loading...</span>
            </div>
          ) : projects.length === 0 ? (
            <Card>
              <div className="py-8 text-center">
                <div className="mx-auto mb-3 flex h-10 w-10 items-center justify-center rounded-full bg-slate-100">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="text-slate-400">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
                  </svg>
                </div>
                <p className="text-sm font-medium text-slate-700">No projects yet</p>
                <p className="mt-1 text-xs text-slate-500">Create your first project to start adding environments and secrets.</p>
              </div>
            </Card>
          ) : (
            projects.map((p) => (
              <Link key={p.id} to={`/projects/${p.id}`} className="block">
                <Card className="transition-all hover:border-slate-300 hover:shadow-sm">
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
