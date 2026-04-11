# Envo — Personal Vault: Secrets for Solo Developers & Vibecoders

## Part 1: Where the Platform Is Right Now

### Architecture

```
envo/
├── backend/     Go (Gin + GORM + PostgreSQL)
├── frontend/    React SPA (Vite + Tailwind CSS 4)
├── cli/         Go CLI (Cobra) — envo login / pull / run
├── nginx/       Reverse proxy + SPA server
└── docker-compose.yml
```

### What's Built and Working

| Area | Status | Details |
|------|--------|---------|
| **Google OAuth** | Done | Browser login + CLI login (localhost callback + exchange code) |
| **Organizations** | Done | Personal workspaces (`owner_type=personal`, auto-created on signup) + team orgs (`owner_type=org`) |
| **RBAC** | Done | 5 system roles (Owner, Admin, Secret Manager, Developer, Viewer) with granular permissions across secrets, projects, environments, members, audit, org management |
| **Projects** | Done | CRUD under orgs, tier-limited |
| **Environments** | Done | CRUD under projects (dev, staging, prod, etc.) |
| **Secrets** | Done | CRUD with AES-256-GCM encryption. AWS KMS envelope encryption in prod, HKDF-derived local encryption in dev. Workspace-scoped key isolation |
| **CLI `pull`** | Done | `envo pull --org X --project Y --env Z` → writes `.env` file, auto-adds `.env` to `.gitignore` |
| **CLI `run`** | Done | `envo run --org X --project Y --env Z -- npm start` → injects secrets as env vars into child process (never touches disk) |
| **Audit Logs** | Done | All secret CRUD + exports logged with user, IP, timestamp |
| **Tier System** | Done | Free / Starter / Team tiers with limits on devs, projects, orgs, secrets per env |
| **Billing** | Done | Razorpay integration (Stripe-ready architecture). Checkout + portal + webhook |
| **Deployment** | Done | Docker Compose (backend + nginx + optional local Postgres). Production on AWS EC2 with host Nginx + certbot SSL |
| **CLI Releases** | Done | GoReleaser + GitHub Actions on `v*` tags → cross-platform binaries |

### Data Model (Current)

```
User
 ├── OwnedOrganizations (owner_id)
 ├── OrgMemberships → OrgMember → Role → Permissions
 ├── CreatedSecrets
 ├── AuditLogs
 ├── RefreshTokens
 └── CLILoginCodes

Organization (personal | org)
 └── Projects
      └── Environments
           └── Secrets (encrypted_value + kms_key_id)

TierLimit (free / starter / team × limit_type)
```

### What's NOT Built Yet (from future.md)

- CLI push (`envo push`)
- GitHub Actions integration (`envo-action`)
- Versioned secrets / rollback
- Secret rotation alerts
- Webhook notifications on change
- Custom roles / per-project permissions
- SSO / SAML
- Audit log search/filter/export
- Self-hosted Helm chart

### UI Changes Already Made (Solo-First UX)

The following frontend changes have been implemented to give personal workspaces a vault-first identity:

- **OrgsPage** — Personal vault is a prominent hero card at top with shield icon, "My Vault" branding, quick stats (projects count, secrets count, tier limits), and recent project tags. Team orgs are separated below.
- **Layout org switcher** — Personal workspace shows as "My Vault" with a shield icon instead of a user icon. Team orgs grouped under a "Teams" header.
- **OrgDetailPage** — When viewing personal workspace: shield icon + "My Vault" title + descriptive subtitle. Breadcrumb says "Workspaces / My Vault". Vault-aware empty states and placeholder text.
- **ProjectDetailPage** — Breadcrumb now resolves actual org name (or "My Vault") instead of hardcoded "Org".
- **EnvironmentDetailPage** — Breadcrumb uses "My Vault" for personal context. Shows both `envo pull` and `envo run` CLI commands with copy buttons. Emerald color scheme for vault environments vs violet for team.
- **Backend** — `GetEnvironment` handler now returns `org_owner_type` so frontend can detect vault context.

---

## Part 2: The Problem — Why Solo Devs Need This

### The Vibecoder Reality

Solo developers and vibecoders today:
1. Have API keys scattered across 5-15 different projects
2. Copy-paste `.env` files from project to project
3. Have stale keys in old projects they forgot about
4. Share keys with themselves across machines via Slack DMs, Notes, or plain text files
5. When deploying, manually re-type or paste secrets into Railway/Render/Vercel dashboards
6. Have zero audit trail of which keys they're using where
7. Lose keys when they nuke a machine or lose a project directory

---

## Part 3: Bundling — How It Actually Works

### The Concept

Bundling is NOT a separate CLI tool — it's the website workflow:

1. You have secrets stored in your **Personal Vault** across different projects
2. When starting a new project, you go to the website and create a project + environment
3. You **pick and choose** secrets you need — add new ones, or reuse keys you already have stored
4. The website gives you a CLI command to pull those secrets locally
5. One command, your dev environment is ready

### The Flow in Practice

```
Website: My Vault → Create project "my-new-saas" → Create env "dev"
  → Add secrets (pick from existing or add new):
      DATABASE_URL = postgres://...
      STRIPE_KEY = sk_test_...
      OPENAI_API_KEY = sk-...
  
Website shows: envo pull --org "Personal Workspace" --project "my-new-saas" --env "dev"

Terminal:
  $ envo pull --project my-new-saas --env dev
  ✓ Wrote 3 secrets to .env
  $ npm run dev   # everything just works
```

### What This Means for the UI

The current environment page already supports this:
- **Add secret** — one key at a time
- **Bulk import** — paste a `.env` file

What we could add later (Phase 2+):
- **Secret picker** — browse your vault's secrets across all projects, select ones to copy into this environment
- **Templates** — pre-built sets of common keys (e.g., "Supabase starter" = `SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_KEY`)

---

## Part 4: The Deployment Bridge — How Secrets Get to Production

### The Honest Answer

There are two fundamentally different problems:

### Problem A: Platforms with APIs (Railway, Render, Vercel, Fly.io)

These platforms expose REST/GraphQL APIs for setting environment variables programmatically.

| Platform | API | How It Works |
|----------|-----|-------------|
| **Railway** | GraphQL API | `variableUpsert` mutation sets env vars per service |
| **Render** | REST API | `PUT /services/{id}/env-vars` replaces all env vars |
| **Vercel** | REST API | `POST /v10/projects/{id}/env` sets per-environment vars |
| **Fly.io** | REST + CLI | `flyctl secrets set KEY=VALUE` or REST API |

The bridge: Envo reads your decrypted secrets → POSTs them to the platform's API. You'd store your platform API token in Envo (encrypted, like any other secret), and Envo uses it to sync.

This is a **future feature** (`envo sync --target railway`). Each platform adapter is ~200-400 lines. Not a massive engineering lift, but needs to be done per-platform.

### Problem B: Dumb Targets (EC2, VPS, Bare Docker, K8s)

These don't have a "set env vars" API. There's no magic here — the solution is the CLI:

| Target | How |
|--------|-----|
| **AWS EC2 / VPS** | SSH in, run `envo pull --project X --env production`. Done. Replaces manually creating `.env` files. |
| **Docker run** | `envo pull` in entrypoint, or generate `.env` file before `docker run --env-file .env` |
| **Docker Compose** | `envo pull` before `docker-compose up`, env_file points to generated `.env` |
| **Kubernetes** | Future: `envo bundle --format k8s` generates a Secret manifest. `kubectl apply -f -` |
| **CI/CD (GitHub Actions)** | Future: Service account tokens (`ENVO_TOKEN` env var) let CI pull secrets without browser login |

### What's Available Today vs. What's Future

| Capability | Status | Notes |
|-----------|--------|-------|
| `envo pull` → `.env` file | **Works now** | Covers EC2, VPS, Docker, local dev |
| `envo run` → inject into process | **Works now** | Zero disk footprint, covers any app |
| Service account tokens for CI | **Not yet** | Needs non-interactive auth (no browser) |
| `envo bundle --format k8s/docker/json` | **Not yet** | Export in different formats |
| `envo sync --target railway` | **Not yet** | Push secrets to platform APIs |
| Platform connection management | **Not yet** | Store platform API tokens in Envo |

### The Priority

1. **Now**: `envo pull` and `envo run` work for all "dumb" targets (EC2, Docker, local dev)
2. **Next**: Service account tokens (unlock CI/CD use case)
3. **Later**: Platform sync adapters (Railway, Render, Vercel, Fly)
4. **Eventually**: Bundle formats for K8s, Terraform, etc.

---

## Part 5: Implementation Roadmap

### Phase 1 — Solo-First CLI (Next)

| Task | Scope | Effort |
|------|-------|--------|
| Auto-resolve personal workspace when `--org` omitted | CLI `pull.go`, `run.go` | Small |
| `.envo.yml` config file support | CLI new `config/project.go` | Small |
| `envo init` — scan `.env` files and import | CLI new `commands/init.go` + backend bulk import endpoint | Medium |
| `envo push` — upload `.env` to vault | CLI new `commands/push.go` + backend `POST /environments/:id/secrets/import` | Medium |

### Phase 2 — Service Tokens + Bundle (Medium Effort)

| Task | Scope | Effort |
|------|-------|--------|
| Service account tokens — model, API, auth middleware | Backend new model + handlers + middleware update | Medium |
| Service account tokens — CLI `envo token create/list/revoke` | CLI new `commands/token.go` | Small |
| Service account tokens — Frontend UI in settings | Frontend new section in `SettingsPage` | Medium |
| `envo bundle` command with format options | CLI new `commands/bundle.go` | Medium |

### Phase 3 — Platform Sync (Killer Feature)

| Task | Scope | Effort |
|------|-------|--------|
| `envo sync` framework + Railway adapter | CLI new `commands/sync.go` + `internal/platforms/railway.go` | Medium |
| Render adapter | CLI `internal/platforms/render.go` | Small |
| Vercel adapter | CLI `internal/platforms/vercel.go` | Small |
| Fly.io adapter | CLI `internal/platforms/fly.go` | Small |
| Platform connection management in frontend | Frontend new page/section | Large |

### Phase 4 — Vault UX Polish

| Task | Scope | Effort |
|------|-------|--------|
| Secret picker — copy secrets between projects/envs | Frontend + backend | Medium |
| Environment templates ("Supabase starter", etc.) | Frontend + backend | Medium |
| `envo diff` — compare local `.env` with vault | CLI | Small |
| `envo status` — show which envs are synced/stale | CLI | Small |
| "Last pulled" timestamps per environment | Backend + frontend | Small |

---

## Part 6: Database Changes Needed (Future Phases)

### Service Tokens (Phase 2)

```go
type ServiceToken struct {
    ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    UserID      uuid.UUID      `gorm:"type:uuid;not null"`
    Name        string         `gorm:"not null"`
    TokenHash   string         `gorm:"not null" json:"-"`
    TokenPrefix string         `gorm:"size:8;not null"` // first 8 chars for display
    Scopes      datatypes.JSON `gorm:"type:jsonb"`
    LastUsedAt  *time.Time
    ExpiresAt   *time.Time
    RevokedAt   *time.Time
    CreatedAt   time.Time
}
```

### Platform Connections (Phase 3)

```go
type PlatformConnection struct {
    ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    UserID      uuid.UUID      `gorm:"type:uuid;not null"`
    Platform    string         `gorm:"not null"` // railway, render, vercel, fly
    Name        string         `gorm:"not null"`
    Credentials string         `gorm:"not null" json:"-"` // encrypted API token
    KMSKeyID    string         `gorm:"not null"`
    Metadata    datatypes.JSON `gorm:"type:jsonb"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`
}
```

### New API Endpoints (Future)

```
# Service Tokens (Phase 2)
POST   /api/v1/auth/service-tokens
GET    /api/v1/auth/service-tokens
DELETE /api/v1/auth/service-tokens/:id

# Bulk Import (Phase 1 — for envo push / envo init)
POST   /api/v1/environments/:id/secrets/import
  Body: { "secrets": {"KEY": "VALUE", ...}, "prune": false }

# Platform Connections (Phase 3)
POST   /api/v1/platforms
GET    /api/v1/platforms
DELETE /api/v1/platforms/:id

# Sync (Phase 3)
POST   /api/v1/environments/:id/sync
```

---

## Part 7: The Vibecoder Pitch

### The 30-Second Story

> You're building a SaaS. You have API keys for Stripe, Resend, OpenAI, Supabase, Cloudflare, Sentry, and a dozen more. They're in `.env` files scattered across your machine. When you deploy to Railway, you paste them one by one. When you set up a new machine, you dig through old Slack DMs to find that Stripe key.
>
> **Envo is your secrets vault.** Store all your keys in one place. Create a project, pick the secrets you need, pull them with one command. Deploy to EC2? `envo pull`. New laptop? `envo login && envo pull`. It just works.

### The DX Flow (End State)

```bash
# Day 1: Go to envo.scopophobic.xyz, log in, open My Vault
# Create project "my-saas" → env "dev" → add your secrets
# Website shows the pull command

# On your machine:
envo login
envo pull --project my-saas --env dev
npm run dev  # everything just works

# Deploy to EC2:
ssh prod
envo pull --project my-saas --env production
pm2 restart my-saas

# New machine? 30 seconds:
brew install envo && envo login
cd my-saas && envo pull
```
