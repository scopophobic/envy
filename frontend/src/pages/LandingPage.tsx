import { useState } from 'react'
import { Link } from 'react-router-dom'
import { getAccessToken } from '../lib/auth'

const INSTALL_MAC_LINUX =
  'curl -fsSL https://raw.githubusercontent.com/scopophobic/envy/main/install.sh | sh'
const INSTALL_WINDOWS = 'irm https://raw.githubusercontent.com/scopophobic/envy/main/install.ps1 | iex'
const INSTALL_GO = 'go install github.com/envo/cli/cmd/envo@latest'

const RELEASES_URL = 'https://github.com/scopophobic/envy/releases'

function CopyBlock({ label, code }: { label: string; code: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <div className="rounded-xl border border-slate-200/80 bg-white/70 p-4 text-left backdrop-blur-sm">
      <div className="mb-2 flex items-center justify-between gap-2">
        <span className="text-xs font-semibold uppercase tracking-wide text-slate-500">{label}</span>
        <button
          type="button"
          onClick={() => {
            void navigator.clipboard.writeText(code)
            setCopied(true)
            setTimeout(() => setCopied(false), 2000)
          }}
          className="shrink-0 rounded-md border border-slate-200 bg-white px-2 py-1 text-[11px] font-medium text-slate-600 hover:bg-slate-50 transition-colors"
        >
          {copied ? 'Copied' : 'Copy'}
        </button>
      </div>
      <pre className="overflow-x-auto rounded-lg bg-slate-900 px-3 py-2.5 font-mono text-[11px] leading-relaxed text-slate-100 sm:text-xs">
        <code className="select-all">{code}</code>
      </pre>
    </div>
  )
}

export function LandingPage() {
  const isLoggedIn = !!getAccessToken()

  return (
    <div className="relative min-h-screen bg-[#f5f3f0] overflow-hidden">
      {/* Heavy grain texture */}
      <div className="pointer-events-none fixed inset-0 z-[1] mix-blend-multiply opacity-[0.45]"
        style={{
          backgroundImage: `url("data:image/svg+xml,%3Csvg viewBox='0 0 512 512' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.85' numOctaves='5' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)' opacity='0.35'/%3E%3C/svg%3E")`,
          backgroundSize: '256px 256px',
        }}
      />

      {/* Nav */}
      <nav className="relative z-10 mx-auto flex max-w-6xl items-center justify-between px-6 py-5">
        <div className="flex items-center gap-2.5">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-slate-900 text-sm font-bold text-white">
            E
          </div>
          <span className="text-xl font-bold text-slate-900 tracking-tight">Envo</span>
        </div>
        <div className="flex items-center gap-4">
          <a
            href="#install-cli"
            className="hidden sm:inline-flex text-sm text-slate-500 hover:text-slate-800 transition-colors"
          >
            Install CLI
          </a>
          <Link
            to="/login"
            className="hidden sm:inline-flex text-sm text-slate-500 hover:text-slate-800 transition-colors"
          >
            Sign in
          </Link>
          <Link
            to={isLoggedIn ? '/orgs' : '/login'}
            className="rounded-full bg-slate-900 px-5 py-2 text-sm font-medium text-white shadow-sm hover:bg-slate-800 transition-colors"
          >
            {isLoggedIn ? 'Dashboard' : 'Get Started'}
          </Link>
        </div>
      </nav>

      {/* Hero */}
      <main className="relative z-10 mx-auto max-w-3xl px-6 pt-20 pb-16 text-center sm:pt-28 sm:pb-20">
        {/* Tag */}
        <div className="inline-flex items-center rounded-full border border-slate-300 bg-white/60 px-4 py-1.5 text-xs font-medium text-slate-600 backdrop-blur-sm">
          End-to-end encrypted secrets
        </div>

        {/* Heading */}
        <h1 className="mt-8 text-5xl font-bold leading-[1.1] tracking-tight text-slate-900 sm:text-6xl lg:text-7xl">
          Share secrets,<br />
          not risk
        </h1>

        {/* Subheading */}
        <p className="mx-auto mt-6 max-w-md text-base text-slate-500 leading-relaxed sm:text-lg">
          Secure environment variables for your team.
          One command to pull, zero chance of leaking.
        </p>

        {/* CTA */}
        <div className="mt-10 flex flex-col items-center gap-3 sm:flex-row sm:justify-center">
          <Link
            to={isLoggedIn ? '/orgs' : '/login'}
            className="rounded-full bg-slate-900 px-8 py-3.5 text-sm font-semibold text-white shadow-lg shadow-slate-900/20 hover:bg-slate-800 transition-all"
          >
            {isLoggedIn ? 'Open Dashboard' : 'Start for Free'}
          </Link>
          <a
            href="#install-cli"
            className="rounded-full border border-slate-300 bg-white/80 px-8 py-3.5 text-sm font-semibold text-slate-800 hover:bg-white transition-all"
          >
            Install CLI
          </a>
        </div>

        {/* Terminal preview */}
        <div className="mx-auto mt-16 max-w-md">
          <div className="overflow-hidden rounded-xl bg-slate-900 shadow-2xl shadow-slate-400/20 ring-1 ring-slate-300/40">
            <div className="flex items-center gap-1.5 border-b border-slate-700/50 px-4 py-2.5">
              <div className="h-2.5 w-2.5 rounded-full bg-red-400/80" />
              <div className="h-2.5 w-2.5 rounded-full bg-amber-400/80" />
              <div className="h-2.5 w-2.5 rounded-full bg-green-400/80" />
              <span className="ml-2 text-[11px] text-slate-500">terminal</span>
            </div>
            <div className="px-5 py-4 font-mono text-[12px] leading-relaxed sm:text-[13px] text-left">
              <div className="text-slate-400">
                <span className="text-green-400">$</span> envo login
              </div>
              <div className="mt-1 text-slate-500">Opens browser — sign in with Google</div>
              <div className="mt-3 text-slate-400">
                <span className="text-green-400">$</span> envo pull --org &quot;My Vault&quot; --project api --env dev
              </div>
              <div className="mt-1 text-emerald-400">Wrote 12 secrets to .env</div>
            </div>
          </div>
        </div>
      </main>

      {/* Install CLI */}
      <section id="install-cli" className="relative z-10 scroll-mt-20 mx-auto max-w-2xl px-6 pb-20">
        <div className="rounded-2xl border border-slate-200/80 bg-white/60 p-6 sm:p-8 backdrop-blur-sm">
          <h2 className="text-xl font-bold text-slate-900 sm:text-2xl">Install the CLI</h2>
          <p className="mt-2 text-sm text-slate-500 leading-relaxed">
            Download the latest <code className="rounded bg-slate-100 px-1 py-0.5 font-mono text-xs">envo</code> binary from{' '}
            <a href={RELEASES_URL} className="font-medium text-slate-700 underline underline-offset-2 hover:text-slate-900">
              GitHub Releases
            </a>
            . The shell installer updates your PATH when it installs to <code className="font-mono text-xs text-slate-600">~/.local/bin</code> — open a new terminal, then run <code className="font-mono text-xs text-slate-600">envo login</code>.
          </p>

          <div className="mt-6 space-y-4">
            <CopyBlock label="macOS / Linux" code={INSTALL_MAC_LINUX} />
            <CopyBlock label="Windows (PowerShell)" code={INSTALL_WINDOWS} />
            <CopyBlock label="Go install (requires Go 1.25+)" code={INSTALL_GO} />
          </div>

          <p className="mt-5 text-xs text-slate-500 leading-relaxed">
            <span className="font-medium text-slate-600">Self-hosted?</span> Set{' '}
            <code className="rounded bg-slate-100 px-1 font-mono text-[11px]">ENVO_API_URL</code> to your API base URL (no trailing slash), or use{' '}
            <code className="rounded bg-slate-100 px-1 font-mono text-[11px]">envo --api https://api.example.com login</code>.
          </p>

          <p className="mt-3 text-xs text-slate-500">
            After installing: <code className="font-mono text-[11px] text-slate-700">envo login</code>
            {' → '}
            <code className="font-mono text-[11px] text-slate-700">envo pull --org … --project … --env …</code>
            {' '}
            (copy the exact command from your environment page in the dashboard).
          </p>
        </div>
      </section>

      {/* Features */}
      <section className="relative z-10 mx-auto max-w-4xl px-6 pb-24">
        <div className="grid gap-6 sm:grid-cols-3">
          {[
            {
              icon: (
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <rect x="3" y="11" width="18" height="11" rx="2" />
                  <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                </svg>
              ),
              title: 'AES-256 + KMS encryption',
              desc: 'Every secret is encrypted with AWS KMS envelope encryption. Never stored in plaintext.',
            },
            {
              icon: (
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <polyline points="4 17 10 11 4 5" />
                  <line x1="12" y1="19" x2="20" y2="19" />
                </svg>
              ),
              title: 'One command, done',
              desc: 'envo pull writes your .env file. No copying, no Slack, no risk.',
            },
            {
              icon: (
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                </svg>
              ),
              title: 'Team access control',
              desc: 'Role-based permissions, audit logs, and org-level controls for your entire team.',
            },
          ].map((f, i) => (
            <div key={i} className="rounded-xl border border-slate-200/80 bg-white/50 p-5 text-center backdrop-blur-sm">
              <div className="mx-auto mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-slate-100 text-slate-600 ring-1 ring-slate-200">
                {f.icon}
              </div>
              <h3 className="text-sm font-semibold text-slate-900">{f.title}</h3>
              <p className="mt-1.5 text-xs text-slate-500 leading-relaxed">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Footer */}
      <footer className="relative z-10 border-t border-slate-200/60">
        <div className="mx-auto flex max-w-6xl flex-col items-center gap-2 px-6 py-5 sm:flex-row sm:justify-between">
          <span className="text-xs text-slate-400">Envo</span>
          <div className="flex flex-wrap items-center justify-center gap-x-4 gap-y-1 text-xs text-slate-400">
            <a href={RELEASES_URL} className="hover:text-slate-600 transition-colors">CLI releases</a>
            <a href="#install-cli" className="hover:text-slate-600 transition-colors">Install</a>
            <span>Secure secrets for modern teams</span>
          </div>
        </div>
      </footer>
    </div>
  )
}
