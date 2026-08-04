import { useState } from 'react'
import { Link } from 'react-router-dom'
import { getAccessToken } from '../lib/auth'

const INSTALL_MAC_LINUX = 'curl -fsSL https://raw.githubusercontent.com/scopophobic/envy/main/install.sh | sh'
const INSTALL_WINDOWS = 'irm https://raw.githubusercontent.com/scopophobic/envy/main/install.ps1 | iex'
const INSTALL_GO = 'go install github.com/envo/cli/cmd/envo@latest'
const RELEASES_URL = 'https://github.com/scopophobic/envy/releases'

function Logo() {
  return (
    <div className="shine-hover relative flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-slate-800 to-slate-950 text-sm font-bold text-white shadow-[inset_0_1px_0_rgba(255,255,255,.18),0_5px_14px_rgba(15,23,42,.2)]">
      <span className="relative z-10">E</span>
      <span className="absolute bottom-1 right-1 h-1.5 w-1.5 rounded-full bg-emerald-400 ring-2 ring-slate-900" />
    </div>
  )
}

function CheckIcon() {
  return <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="m5 12 4 4L19 6" /></svg>
}

function ArrowIcon() {
  return <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M5 12h14M13 6l6 6-6 6" /></svg>
}

function CopyBlock({ label, code }: { label: string; code: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <div className="group rounded-2xl border border-white/10 bg-white/[.045] p-4 text-left transition-all hover:-translate-y-0.5 hover:border-white/20 hover:bg-white/[.065]">
      <div className="mb-3 flex items-center justify-between gap-2">
        <span className="text-[11px] font-semibold uppercase tracking-[0.13em] text-slate-400">{label}</span>
        <button
          type="button"
          onClick={() => {
            void navigator.clipboard.writeText(code)
            setCopied(true)
            setTimeout(() => setCopied(false), 1800)
          }}
          className={`rounded-lg border px-2.5 py-1 text-[11px] font-semibold transition-all active:scale-95 ${copied ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-300' : 'border-white/10 bg-white/[.06] text-slate-300 hover:bg-white/10 hover:text-white'}`}
        >
          {copied ? 'Copied ✓' : 'Copy'}
        </button>
      </div>
      <code className="scrollbar-subtle block overflow-x-auto whitespace-nowrap rounded-xl border border-white/[.06] bg-slate-950/80 px-3 py-3 font-mono text-[11px] text-emerald-300 sm:text-xs">{code}</code>
    </div>
  )
}

function ProductDemo() {
  const [mode, setMode] = useState<'agent' | 'developer'>('agent')

  return (
    <div className="landing-float relative mx-auto w-full max-w-[560px] lg:mr-0">
      <div className="absolute -inset-6 rounded-[2.5rem] bg-gradient-to-br from-violet-500/15 via-transparent to-emerald-400/15 blur-2xl" />
      <div className="relative overflow-hidden rounded-[1.6rem] border border-slate-700/80 bg-slate-950 shadow-[0_35px_90px_rgba(15,23,42,.38)] ring-1 ring-white/[.05]">
        <div className="flex items-center justify-between border-b border-white/[.07] px-4 py-3">
          <div className="flex items-center gap-2">
            <div className="flex gap-1.5"><span className="h-2.5 w-2.5 rounded-full bg-red-400/70" /><span className="h-2.5 w-2.5 rounded-full bg-amber-400/70" /><span className="h-2.5 w-2.5 rounded-full bg-emerald-400/70" /></div>
            <span className="ml-2 text-[11px] font-medium text-slate-500">envo · access control</span>
          </div>
          <span className="flex items-center gap-1.5 text-[10px] font-medium text-emerald-300"><span className="status-pulse h-1.5 w-1.5 rounded-full bg-emerald-400 text-emerald-400" />Policy online</span>
        </div>

        <div className="border-b border-white/[.07] p-3">
          <div className="grid grid-cols-2 rounded-xl bg-white/[.045] p-1">
            <button onClick={() => setMode('agent')} className={`rounded-lg px-3 py-2 text-xs font-semibold transition-all ${mode === 'agent' ? 'bg-violet-500 text-white shadow-lg shadow-violet-950/40' : 'text-slate-500 hover:text-slate-300'}`}>AI agent</button>
            <button onClick={() => setMode('developer')} className={`rounded-lg px-3 py-2 text-xs font-semibold transition-all ${mode === 'developer' ? 'bg-white/10 text-white shadow-lg' : 'text-slate-500 hover:text-slate-300'}`}>Developer CLI</button>
          </div>
        </div>

        <div key={mode} className="page-enter min-h-[390px] p-5 sm:p-6">
          {mode === 'agent' ? (
            <>
              <div className="flex items-center justify-between gap-3">
                <div className="flex min-w-0 items-center gap-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-violet-500/15 text-violet-300 ring-1 ring-violet-400/20">
                    <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8"><path d="M12 8V4H8"/><rect width="16" height="12" x="4" y="8" rx="2"/><path d="M2 14h2M20 14h2M9 13v2M15 13v2"/></svg>
                  </div>
                  <div className="min-w-0"><p className="truncate text-sm font-semibold text-white">claude-code / backend</p><p className="mt-0.5 text-[11px] text-slate-500">Agent identity · active now</p></div>
                </div>
                <span className="rounded-full border border-emerald-400/20 bg-emerald-400/10 px-2.5 py-1 text-[10px] font-semibold text-emerald-300">ACTIVE</span>
              </div>

              <div className="mt-5 grid grid-cols-2 gap-2 text-left">
                <div className="rounded-xl border border-white/[.07] bg-white/[.035] p-3"><p className="text-[10px] uppercase tracking-wider text-slate-600">Environment</p><p className="mt-1.5 text-xs font-medium text-slate-200">checkout-api / dev</p></div>
                <div className="rounded-xl border border-white/[.07] bg-white/[.035] p-3"><p className="text-[10px] uppercase tracking-wider text-slate-600">Capability</p><p className="mt-1.5 text-xs font-medium text-slate-200">secrets.inject</p></div>
              </div>

              <div className="mt-4 rounded-xl border border-white/[.07] bg-white/[.025] p-4 text-left">
                <div className="flex items-center justify-between"><span className="text-[11px] font-semibold text-slate-300">Requested variables</span><span className="text-[10px] text-slate-600">3 of 3 allowed</span></div>
                <div className="mt-3 flex flex-wrap gap-2">
                  {['DATABASE_URL', 'STRIPE_TEST_KEY', 'REDIS_URL'].map(key => <span key={key} className="rounded-md border border-violet-400/15 bg-violet-400/[.07] px-2 py-1 font-mono text-[10px] text-violet-200">{key}</span>)}
                </div>
              </div>

              <div className="mt-5 space-y-3 text-left">
                {[['Identity verified', 'Token hash matched'], ['Live grant checked', 'Environment + keys allowed'], ['Secrets injected', 'Nothing written to disk']].map(([title, detail], index) => (
                  <div key={title} className="flex items-center gap-3">
                    <div className="relative flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-emerald-400/10 text-emerald-300 ring-1 ring-emerald-400/20"><CheckIcon />{index < 2 && <span className="absolute left-1/2 top-6 h-3 w-px bg-emerald-400/15" />}</div>
                    <p className="text-xs font-medium text-slate-300">{title} <span className="font-normal text-slate-600">· {detail}</span></p>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <>
              <div className="flex items-center gap-3 text-left"><div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-400/10 text-emerald-300 ring-1 ring-emerald-400/20"><svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8"><polyline points="4 17 10 11 4 5"/><line x1="12" x2="20" y1="19" y2="19"/></svg></div><div><p className="text-sm font-semibold text-white">Runtime secret delivery</p><p className="mt-0.5 text-[11px] text-slate-500">Human-authenticated CLI session</p></div></div>
              <div className="mt-6 rounded-xl border border-white/[.07] bg-black/30 p-4 font-mono text-left text-xs leading-7">
                <p className="text-slate-500"><span className="text-emerald-400">$</span> envo run --project api \\</p>
                <p className="pl-3 text-slate-400">--env development -- npm run dev</p>
                <p className="mt-3 text-violet-300">✓ Identity authenticated</p>
                <p className="text-violet-300">✓ Workspace permission checked</p>
                <p className="text-emerald-300">✓ Injecting 12 secrets into npm</p>
                <p className="mt-3 text-slate-500">&gt; api@1.0.0 dev</p>
                <p className="text-white">Server ready on localhost:3000 <span className="landing-cursor inline-block h-3 w-1.5 bg-emerald-400 align-middle" /></p>
              </div>
              <div className="mt-5 grid grid-cols-3 gap-2">
                {[['12', 'Injected'], ['0', 'Files written'], ['1', 'Audit event']].map(([value, label]) => <div key={label} className="rounded-xl border border-white/[.07] bg-white/[.03] px-2 py-3 text-center"><p className="text-lg font-bold text-white">{value}</p><p className="mt-0.5 text-[10px] text-slate-600">{label}</p></div>)}
              </div>
            </>
          )}
        </div>
      </div>

      <div className="landing-drift absolute -right-3 -top-5 hidden items-center gap-2 rounded-xl border border-emerald-200 bg-white px-3 py-2 text-xs font-semibold text-slate-700 shadow-xl shadow-slate-900/10 sm:flex">
        <span className="flex h-6 w-6 items-center justify-center rounded-lg bg-emerald-50 text-emerald-600"><CheckIcon /></span> Revocation is live
      </div>
      <div className="landing-drift-delayed absolute -bottom-5 -left-5 hidden rounded-xl border border-violet-200 bg-white px-3 py-2 shadow-xl shadow-slate-900/10 sm:block">
        <p className="text-[10px] font-semibold uppercase tracking-wider text-violet-500">Audit actor</p><p className="mt-0.5 text-xs font-semibold text-slate-800">AI agent · not a human token</p>
      </div>
    </div>
  )
}

export function LandingPage() {
  const isLoggedIn = !!getAccessToken()
  const appPath = isLoggedIn ? '/orgs' : '/login'

  return (
    <div className="min-h-screen overflow-hidden bg-[#f8fafc] text-slate-950">
      <div className="landing-grid pointer-events-none fixed inset-0 opacity-50" />

      <nav className="sticky top-0 z-50 border-b border-slate-200/70 bg-white/80 backdrop-blur-xl">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-5 sm:px-8">
          <Link to="/" className="flex items-center gap-2.5"><Logo /><span className="text-xl font-bold tracking-tight">Envo</span></Link>
          <div className="hidden items-center gap-7 md:flex">
            <a href="#product" className="text-sm font-medium text-slate-500 hover:text-slate-950">Product</a>
            <a href="#agents" className="text-sm font-medium text-slate-500 hover:text-slate-950">Agent access</a>
            <a href="#install-cli" className="text-sm font-medium text-slate-500 hover:text-slate-950">CLI</a>
            <Link to="/pricing" className="text-sm font-medium text-slate-500 hover:text-slate-950">Pricing</Link>
          </div>
          <div className="flex items-center gap-2">
            {!isLoggedIn && <Link to="/login" className="hidden rounded-lg px-3 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-100 sm:inline-flex">Sign in</Link>}
            <Link to={appPath} className="group inline-flex items-center gap-2 rounded-xl bg-slate-950 px-4 py-2.5 text-sm font-semibold text-white shadow-[0_6px_18px_rgba(15,23,42,.16)] hover:-translate-y-0.5 hover:bg-slate-800 hover:shadow-[0_10px_26px_rgba(15,23,42,.2)] active:translate-y-0 active:scale-[.98]">{isLoggedIn ? 'Open dashboard' : 'Start free'}<span className="transition-transform group-hover:translate-x-0.5"><ArrowIcon /></span></Link>
          </div>
        </div>
      </nav>

      <main className="relative">
        <section className="relative mx-auto grid max-w-7xl items-center gap-16 px-5 pb-24 pt-16 sm:px-8 sm:pt-24 lg:grid-cols-[1.02fr_.98fr] lg:pb-32 lg:pt-28">
          <div className="landing-reveal relative z-10 text-center lg:text-left">
            <div className="inline-flex items-center gap-2 rounded-full border border-violet-200 bg-white/80 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-[0.13em] text-violet-700 shadow-sm backdrop-blur">
              <span className="status-pulse h-1.5 w-1.5 rounded-full bg-emerald-500 text-emerald-500" /> Credential control for humans + agents
            </div>
            <h1 className="mt-7 text-5xl font-bold leading-[1.03] tracking-[-0.045em] text-slate-950 sm:text-6xl lg:text-[4.4rem]">
              Give every runtime<br className="hidden sm:block" /> what it needs.
              <span className="mt-1 block bg-gradient-to-r from-violet-600 via-violet-500 to-emerald-500 bg-clip-text text-transparent">Nothing more.</span>
            </h1>
            <p className="mx-auto mt-6 max-w-xl text-base leading-7 text-slate-600 sm:text-lg lg:mx-0">
              Envo is the secure credential layer for developers, organizations, deployments, and AI agents—with scoped access, instant revocation, and a complete audit trail.
            </p>
            <div className="mt-9 flex flex-col items-center gap-3 sm:flex-row sm:justify-center lg:justify-start">
              <Link to={appPath} className="group inline-flex w-full items-center justify-center gap-2 rounded-xl bg-slate-950 px-6 py-3.5 text-sm font-semibold text-white shadow-[0_12px_30px_rgba(15,23,42,.2)] hover:-translate-y-0.5 hover:bg-slate-800 hover:shadow-[0_18px_38px_rgba(15,23,42,.24)] active:translate-y-0 sm:w-auto">{isLoggedIn ? 'Open your workspaces' : 'Build your secure workspace'}<span className="transition-transform group-hover:translate-x-1"><ArrowIcon /></span></Link>
              <a href="#product" className="inline-flex w-full items-center justify-center gap-2 rounded-xl border border-slate-200 bg-white/80 px-6 py-3.5 text-sm font-semibold text-slate-700 shadow-sm backdrop-blur hover:-translate-y-0.5 hover:border-slate-300 hover:bg-white hover:shadow-md sm:w-auto"><span className="flex h-5 w-5 items-center justify-center rounded-full bg-violet-100 text-violet-600"><svg width="9" height="9" viewBox="0 0 10 10" fill="currentColor"><path d="M8.5 5 2.5 8.5v-7L8.5 5Z"/></svg></span>See how it works</a>
            </div>
            <div className="mt-7 flex flex-wrap items-center justify-center gap-x-5 gap-y-2 text-xs text-slate-500 lg:justify-start">
              {['Free personal vault', 'No card required', 'Self-hostable'].map(item => <span key={item} className="flex items-center gap-1.5"><span className="text-emerald-500"><CheckIcon /></span>{item}</span>)}
            </div>
          </div>
          <ProductDemo />
        </section>

        <section className="border-y border-slate-200/80 bg-white/65 backdrop-blur">
          <div className="mx-auto flex max-w-7xl flex-col items-center justify-between gap-5 px-5 py-6 sm:px-8 lg:flex-row">
            <p className="text-center text-xs font-semibold uppercase tracking-[0.16em] text-slate-400 lg:text-left">One control layer across your workflow</p>
            <div className="flex flex-wrap items-center justify-center gap-x-7 gap-y-3 text-sm font-semibold text-slate-500">
              {['Local development', 'Team workspaces', 'AI coding agents', 'Vercel delivery'].map((item, index) => <span key={item} className="flex items-center gap-2"><span className={`h-1.5 w-1.5 rounded-full ${index === 2 ? 'bg-violet-500' : 'bg-emerald-400'}`} />{item}</span>)}
            </div>
          </div>
        </section>

        <section id="product" className="scroll-mt-24 mx-auto max-w-7xl px-5 py-24 sm:px-8 sm:py-32">
          <div className="mx-auto max-w-2xl text-center">
            <p className="text-xs font-semibold uppercase tracking-[0.16em] text-violet-600">Built around control</p>
            <h2 className="mt-4 text-3xl font-bold tracking-tight text-slate-950 sm:text-5xl">Secrets move. Control stays with you.</h2>
            <p className="mt-5 text-base leading-7 text-slate-500">From a developer’s terminal to an autonomous coding harness, every access path uses identity, scope, and audit—not shared `.env` files.</p>
          </div>

          <div className="mt-14 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            <article className="landing-card group relative overflow-hidden rounded-3xl border border-slate-200 bg-slate-950 p-7 text-white lg:col-span-2">
              <div className="absolute -right-16 -top-16 h-56 w-56 rounded-full bg-violet-500/20 blur-3xl transition-transform duration-700 group-hover:scale-125" />
              <div className="relative max-w-lg"><span className="flex h-11 w-11 items-center justify-center rounded-2xl bg-violet-400/10 text-violet-300 ring-1 ring-violet-400/20"><svg width="21" height="21" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7"><path d="M12 8V4H8"/><rect width="16" height="12" x="4" y="8" rx="2"/><path d="M2 14h2M20 14h2M9 13v2M15 13v2"/></svg></span><p className="mt-7 text-xs font-semibold uppercase tracking-[0.15em] text-violet-300">Agent identities</p><h3 className="mt-2 text-2xl font-bold tracking-tight">Stop giving AI agents your human credentials.</h3><p className="mt-3 text-sm leading-6 text-slate-400">Issue an independent token, grant exact environment variables, set expiration, then suspend or revoke access without rotating the real secrets.</p></div>
              <div className="relative mt-8 flex flex-wrap gap-2">{['Organization-owned', 'Hashed tokens', 'Key-level grants', 'Live revocation'].map(item => <span key={item} className="rounded-lg border border-white/10 bg-white/[.05] px-2.5 py-1.5 text-[11px] text-slate-300">{item}</span>)}</div>
            </article>

            <article className="landing-card group rounded-3xl border border-emerald-200/70 bg-gradient-to-br from-emerald-50 to-white p-7">
              <span className="flex h-11 w-11 items-center justify-center rounded-2xl bg-white text-emerald-600 shadow-sm ring-1 ring-emerald-200 transition-transform group-hover:-rotate-3 group-hover:scale-105"><svg width="21" height="21" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7"><polyline points="4 17 10 11 4 5"/><line x1="12" x2="20" y1="19" y2="19"/></svg></span><p className="mt-7 text-xs font-semibold uppercase tracking-[0.15em] text-emerald-700">Runtime delivery</p><h3 className="mt-2 text-xl font-bold tracking-tight">Inject, don’t scatter.</h3><p className="mt-3 text-sm leading-6 text-slate-500"><code className="font-mono text-emerald-700">envo run</code> puts approved secrets directly into the child process. Nothing needs to touch disk.</p>
            </article>

            <article className="landing-card group rounded-3xl border border-slate-200 bg-white p-7">
              <span className="flex h-11 w-11 items-center justify-center rounded-2xl bg-violet-50 text-violet-600 transition-transform group-hover:scale-105"><svg width="21" height="21" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/></svg></span><p className="mt-7 text-xs font-semibold uppercase tracking-[0.15em] text-violet-600">Organizations</p><h3 className="mt-2 text-xl font-bold tracking-tight">People get roles. Agents get grants.</h3><p className="mt-3 text-sm leading-6 text-slate-500">Keep human collaboration and machine access separate while governing both from the same workspace.</p>
            </article>

            <article className="landing-card group relative overflow-hidden rounded-3xl border border-slate-200 bg-white p-7 lg:col-span-2">
              <div className="grid gap-7 sm:grid-cols-[1fr_auto] sm:items-end"><div><span className="flex h-11 w-11 items-center justify-center rounded-2xl bg-amber-50 text-amber-600 transition-transform group-hover:rotate-3 group-hover:scale-105"><svg width="21" height="21" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7"><path d="M3 3v18h18"/><path d="m7 16 4-4 4 2 5-6"/></svg></span><p className="mt-7 text-xs font-semibold uppercase tracking-[0.15em] text-amber-600">Auditability</p><h3 className="mt-2 text-xl font-bold tracking-tight">Know who—or what—used access.</h3><p className="mt-3 max-w-lg text-sm leading-6 text-slate-500">Secret reads record the human or agent actor. Agent requests include the credential, grant, lease, purpose, and session context.</p></div><div className="space-y-2 text-xs"><div className="rounded-xl border border-slate-200 bg-slate-50 px-3 py-2"><span className="text-slate-400">actor</span> <strong className="ml-3 text-slate-700">agent</strong></div><div className="rounded-xl border border-slate-200 bg-slate-50 px-3 py-2"><span className="text-slate-400">action</span> <strong className="ml-3 text-slate-700">secret_read</strong></div><div className="rounded-xl border border-emerald-200 bg-emerald-50 px-3 py-2"><span className="text-emerald-600">decision</span> <strong className="ml-3 text-emerald-800">allowed</strong></div></div></div>
            </article>
          </div>
        </section>

        <section id="agents" className="scroll-mt-24 bg-slate-950 py-24 text-white sm:py-32">
          <div className="mx-auto grid max-w-7xl gap-14 px-5 sm:px-8 lg:grid-cols-2 lg:items-center">
            <div>
              <div className="inline-flex items-center gap-2 rounded-full border border-violet-400/20 bg-violet-400/10 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-[0.14em] text-violet-200"><span className="status-pulse h-1.5 w-1.5 rounded-full bg-violet-400 text-violet-400" />Built for the agent era</div>
              <h2 className="mt-6 text-3xl font-bold leading-tight tracking-tight sm:text-5xl">An agent gets a job.<br/><span className="text-slate-500">Not your entire vault.</span></h2>
              <p className="mt-6 max-w-xl text-base leading-7 text-slate-400">Create a distinct identity for Claude Code, Codex, CI, or another harness. Bind it to one environment and a named set of variables. Every future request rechecks that live policy.</p>
              <div className="mt-8 space-y-4">
                {['Tokens are shown once and stored only as hashes', 'Agent credentials cannot use human management APIs', 'Revoking the token, grant, or agent stops future requests'].map(item => <div key={item} className="flex items-start gap-3 text-sm text-slate-300"><span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-400/10 text-emerald-300"><CheckIcon /></span>{item}</div>)}
              </div>
              <Link to={appPath} className="group mt-9 inline-flex items-center gap-2 rounded-xl bg-white px-5 py-3 text-sm font-semibold text-slate-950 shadow-xl hover:-translate-y-0.5 hover:bg-slate-100">Create an agent identity <span className="transition-transform group-hover:translate-x-1"><ArrowIcon /></span></Link>
            </div>

            <div className="relative">
              <div className="landing-beam absolute left-10 top-10 h-[calc(100%-5rem)] w-px bg-gradient-to-b from-violet-400 via-emerald-400 to-transparent" />
              <div className="space-y-4">
                {[{n:'01',title:'Create identity',desc:'claude-code / checkout-api',tone:'violet'},{n:'02',title:'Issue credential',desc:'envo_agent_… · expires in 7 days',tone:'slate'},{n:'03',title:'Grant exact scope',desc:'development · DATABASE_URL + 2 keys',tone:'emerald'},{n:'04',title:'Resolve + audit',desc:'3 injected · lease recorded',tone:'emerald'}].map(step => (
                  <div key={step.n} className="group relative ml-0 flex gap-4 rounded-2xl border border-white/[.08] bg-white/[.04] p-4 backdrop-blur transition-all hover:translate-x-1 hover:border-white/15 hover:bg-white/[.065] sm:ml-4 sm:p-5">
                    <span className={`relative z-10 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl text-xs font-bold ring-1 ${step.tone === 'violet' ? 'bg-violet-400/15 text-violet-300 ring-violet-400/20' : step.tone === 'emerald' ? 'bg-emerald-400/10 text-emerald-300 ring-emerald-400/20' : 'bg-white/[.06] text-slate-300 ring-white/10'}`}>{step.n}</span>
                    <div><p className="text-sm font-semibold text-white">{step.title}</p><p className="mt-1 font-mono text-[11px] text-slate-500">{step.desc}</p></div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        <section className="mx-auto max-w-7xl px-5 py-24 sm:px-8 sm:py-32">
          <div className="mx-auto max-w-2xl text-center"><p className="text-xs font-semibold uppercase tracking-[0.16em] text-emerald-600">Simple workflow</p><h2 className="mt-4 text-3xl font-bold tracking-tight sm:text-5xl">From vault to runtime in three steps.</h2></div>
          <div className="mt-14 grid gap-5 md:grid-cols-3">
            {[['01','Organize','Store encrypted secrets by workspace, project, and environment.'],['02','Authorize','Use human roles or narrow agent grants to define access.'],['03','Deliver','Inject into a process or manually sync to a deployment platform.']].map(([number,title,desc], index) => <div key={number} className="group relative rounded-3xl border border-slate-200 bg-white p-7 shadow-sm transition-all hover:-translate-y-1 hover:border-violet-200 hover:shadow-xl hover:shadow-violet-950/[.05]"><div className="flex items-center justify-between"><span className="text-xs font-bold text-violet-600">{number}</span>{index < 2 && <ArrowIcon />}</div><h3 className="mt-10 text-xl font-bold">{title}</h3><p className="mt-3 text-sm leading-6 text-slate-500">{desc}</p></div>)}
          </div>
        </section>

        <section id="install-cli" className="scroll-mt-24 px-5 pb-24 sm:px-8 sm:pb-32">
          <div className="relative mx-auto max-w-6xl overflow-hidden rounded-[2rem] border border-slate-800 bg-slate-950 px-5 py-10 text-white shadow-[0_28px_80px_rgba(15,23,42,.22)] sm:p-10">
            <div className="pointer-events-none absolute -right-20 -top-24 h-72 w-72 rounded-full bg-violet-500/15 blur-3xl" />
            <div className="relative grid gap-10 lg:grid-cols-[.75fr_1.25fr] lg:items-center">
              <div><p className="text-xs font-semibold uppercase tracking-[0.16em] text-emerald-300">Envo CLI</p><h2 className="mt-4 text-3xl font-bold tracking-tight">Secure delivery that stays out of your way.</h2><p className="mt-4 text-sm leading-6 text-slate-400">Install once, authenticate, and move approved secrets into the runtime you choose. Use <code className="text-emerald-300">ENVO_TOKEN</code> for controlled agent execution.</p><a href={RELEASES_URL} className="group mt-6 inline-flex items-center gap-2 text-sm font-semibold text-white hover:text-emerald-300">View releases <span className="transition-transform group-hover:translate-x-1"><ArrowIcon /></span></a></div>
              <div className="grid gap-3"><CopyBlock label="macOS / Linux" code={INSTALL_MAC_LINUX} /><CopyBlock label="Windows PowerShell" code={INSTALL_WINDOWS} /><CopyBlock label="Go 1.25+" code={INSTALL_GO} /></div>
            </div>
          </div>
        </section>

        <section className="border-y border-slate-200 bg-white/80 px-5 py-24 text-center sm:px-8">
          <div className="mx-auto max-w-3xl"><div className="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-950 text-white shadow-xl"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="m9 12 2 2 4-4"/></svg></div><h2 className="mt-7 text-3xl font-bold tracking-tight sm:text-5xl">Use powerful agents.<br/>Keep organizational control.</h2><p className="mx-auto mt-5 max-w-xl text-base leading-7 text-slate-500">Start with a personal vault, grow into team workspaces, and give every automated actor an identity of its own.</p><div className="mt-8 flex flex-col justify-center gap-3 sm:flex-row"><Link to={appPath} className="group inline-flex items-center justify-center gap-2 rounded-xl bg-slate-950 px-6 py-3.5 text-sm font-semibold text-white shadow-lg hover:-translate-y-0.5 hover:bg-slate-800">{isLoggedIn ? 'Open dashboard' : 'Start free'}<span className="transition-transform group-hover:translate-x-1"><ArrowIcon /></span></Link><Link to="/pricing" className="inline-flex items-center justify-center rounded-xl border border-slate-200 bg-white px-6 py-3.5 text-sm font-semibold text-slate-700 hover:-translate-y-0.5 hover:border-slate-300 hover:shadow-md">View pricing</Link></div></div>
        </section>
      </main>

      <footer className="bg-slate-950 text-slate-400">
        <div className="mx-auto flex max-w-7xl flex-col gap-8 px-5 py-10 sm:px-8 md:flex-row md:items-end md:justify-between">
          <div><div className="flex items-center gap-2.5"><Logo /><span className="text-lg font-bold text-white">Envo</span></div><p className="mt-3 max-w-sm text-xs leading-5 text-slate-500">Credential control for developers, organizations, deployments, and AI agents.</p></div>
          <div className="flex flex-wrap gap-x-6 gap-y-3 text-xs"><a href="#product" className="hover:text-white">Product</a><a href="#agents" className="hover:text-white">Agent access</a><a href="#install-cli" className="hover:text-white">Install CLI</a><Link to="/pricing" className="hover:text-white">Pricing</Link><a href={RELEASES_URL} className="hover:text-white">GitHub releases</a></div>
        </div>
      </footer>
    </div>
  )
}
