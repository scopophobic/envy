# Envo

Secure environment variable management for teams. Encrypted at rest, shared with one command.

## What is Envo?

Envo lets teams store, manage, and distribute `.env` secrets without ever exposing them in plaintext. Secrets are encrypted with AES-256 + AWS KMS envelope encryption before they touch the database. Team members pull secrets to their local machine with a single CLI command.

```
$ envo pull --env production
✓ Wrote 12 secrets to .env
```

No more sharing secrets over Slack. No more stale `.env` files in wikis.

## Architecture

```
envo/
├── backend/     Go API (Gin + GORM + PostgreSQL)
├── frontend/    React SPA (Vite + Tailwind CSS)
├── cli/         CLI tool (Cobra)
├── nginx/       Reverse proxy + static file server
└── docker-compose.yml
```

**Backend** — REST API handling auth (Google OAuth), organizations, projects, environments, secrets (encrypted), RBAC, audit logs, billing (Razorpay), and tier enforcement.

**Frontend** — Single-page app with org management, secret CRUD, plan/billing settings, and a top-nav layout.

**CLI** — `envo login` authenticates via browser OAuth. `envo pull` decrypts and writes secrets to `.env` in the current directory.

## Install the CLI

Use the **Envo CLI** to sign in and pull secrets into your project. Binaries are published on [GitHub Releases](https://github.com/scopophobic/envy/releases) (via GoReleaser when you push a `v*` tag).

### One-line install (recommended)

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/scopophobic/envy/main/install.sh | sh
```

When the binary goes to `~/.local/bin` (typical on macOS), the script **adds that directory to your PATH** in `~/.zshrc` or `~/.bashrc` (one marked block). Open a **new terminal** (or `source ~/.zshrc`), then run `envo`. To skip editing shell config: `ENVO_INSTALL_NO_PATH=1` before `curl` (see [`INSTALL.md`](INSTALL.md)).

**Windows (PowerShell)**

```powershell
irm https://raw.githubusercontent.com/scopophobic/envy/main/install.ps1 | iex
```

If PowerShell blocks the script, run once:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

**Windows (Git Bash)** — same shell script as macOS/Linux; installs to `~/bin/envo.exe` and updates `~/.bashrc` for PATH when needed.

### Install with Go

Requires Go 1.25+:

```bash
go install github.com/envo/cli/cmd/envo@latest
```

Ensure `$GOPATH/bin` or `$HOME/go/bin` is on your `PATH`.

### Point the CLI at your API

By default the CLI uses `http://localhost:8080`. For a hosted Envo instance:

```bash
export ENVO_API_URL=https://api.your-domain.com
# or per command:
envo --api https://api.your-domain.com login
```

Use the same host style as your backend’s Google OAuth redirect URL (`localhost` vs `127.0.0.1` must stay consistent).

### After install

```bash
envo login
envo whoami
envo pull --org "My Org" --project "api" --env "development"
envo run --org "My Org" --project "api" --env "development" -- npm start
```

| Command | What it does |
|--------|----------------|
| `envo login` | Opens the browser; sign in with Google |
| `envo logout` | Clears saved tokens |
| `envo whoami` | Shows current user |
| `envo pull --org … --project … --env …` | Writes secrets to `.env` in the current directory |
| `envo run … -- <command>` | Runs a command with secrets in the environment (nothing written to disk) |

More detail: [`INSTALL.md`](INSTALL.md) (install scripts), [`cli/README.md`](cli/README.md) (build from source, token paths).

### From this repo (contributors)

Without a release binary, from the **repository root**:

- **macOS / Linux:** `chmod +x envo && ./envo login` (wrapper runs Go)
- **Windows:** `.\envo.cmd login`

Or build once:

```bash
cd cli && go build -o envo ./cmd/envo
./envo login   # from cli/
```

## Features

- **End-to-end encryption** — AES-256-GCM with AWS KMS envelope encryption (falls back to local encryption in dev)
- **One-command pull** — `envo pull --org acme --project api --env production`
- **Google OAuth** — sign in with Google, no passwords to manage
- **Organizations & RBAC** — owner / admin / member / viewer roles with granular permissions
- **Audit logs** — every secret access and change is logged
- **Tier limits** — free / starter / team plans with per-org enforcement
- **Razorpay billing** — subscription management via Razorpay (Stripe-ready architecture)
- **Docker deployment** — single `docker-compose up` for the full stack

## Quick Start (Development)

### Prerequisites

- Go 1.25+
- Node.js 20+
- PostgreSQL 15+

### 1. Database

```bash
createdb envo_db
```

### 2. Backend

```bash
cd backend
cp .env.example .env   # edit with your DB password, JWT secret, Google OAuth keys
go run ./cmd/server -migrate
go run ./cmd/server -seed
go run ./cmd/server
```

The API starts at `http://localhost:8080`.

### 3. Frontend

```bash
cd frontend
npm install
npm run dev
```

Opens at `http://localhost:5173`.

### 4. CLI

With the API running locally, install the CLI using [Install the CLI](#install-the-cli) (e.g. `go install …` or `./envo` from the repo root), then:

```bash
export ENVO_API_URL=http://localhost:8080   # optional; this is the default
envo login
envo pull --org "My Org" --project "api" --env "production"
```

## Production Deployment (AWS EC2)

The project ships with Docker Compose for single-server deployment.

```bash
# On your EC2 instance
git clone <repo-url> envo && cd envo
cp .env.production.example .env.production
nano .env.production   # fill in real values
chmod +x deploy.sh
./deploy.sh
```

See `.env.production.example` for all required environment variables.

After deployment:
1. Point your subdomain's A record to the EC2 public IP
2. Set up HTTPS: `sudo certbot --nginx -d your-subdomain.example.com`
3. Update `GOOGLE_REDIRECT_URL` and `FRONTEND_URL` to use `https://`

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `DB_PASSWORD` | Yes | PostgreSQL password |
| `JWT_SECRET` | Yes | Random string for signing tokens |
| `GOOGLE_CLIENT_ID` | Prod | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | Prod | Google OAuth client secret |
| `GOOGLE_REDIRECT_URL` | Prod | OAuth callback URL |
| `FRONTEND_URL` | Prod | Frontend origin for CORS |
| `AWS_KMS_KEY_ID` | No | KMS key ARN (falls back to local encryption) |
| `RAZORPAY_KEY_ID` | No | Enables billing endpoints |
| `RAZORPAY_KEY_SECRET` | No | Razorpay API secret |

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go, Gin, GORM, PostgreSQL |
| Frontend | React, TypeScript, Vite, Tailwind CSS |
| CLI | Go, Cobra |
| Encryption | AES-256-GCM, AWS KMS |
| Auth | Google OAuth 2.0, JWT (access + refresh) |
| Billing | Razorpay Subscriptions |
| Deployment | Docker, Docker Compose, Nginx |

## License

MIT
