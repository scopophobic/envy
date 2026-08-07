# Envo

Envo is a secret-management platform for developers, teams, applications, and coding agents. It provides one place to store credentials, organize them by project and environment, control who or what can use them, and revoke future access without distributing the original credentials again.

The repository is named `envy`; the product, API, and CLI are called **Envo**.

## What Envo provides

### Workspaces and access

- A personal vault created for every user
- Team organizations with invitations and member management
- Owner, Admin, Secret Manager, Developer, and Viewer roles
- Custom organization roles and granular permissions
- Projects and separate development, staging, and production environments
- Organization audit logs

### Secret handling

- AES-256-GCM encryption before database storage
- AWS KMS envelope encryption in production
- Workspace-scoped local encryption for development only
- Metadata-only secret lists in the dashboard
- Audited secret export for authorized users
- Soft deletion and explicit permanent deletion

### Human and agent delivery

- A web dashboard for managing workspaces and secrets
- A Go CLI for pulling `.env` files or injecting secrets directly into a process
- Organization-owned agent identities for coding harnesses and automation
- Independent agent credentials stored only as hashes
- Project, environment, and key-level agent grants
- Live credential, grant, and agent revocation checks

### Integrations and operations

- Google OAuth for browser and CLI authentication
- Encrypted Vercel connections and manual environment sync
- Optional Razorpay subscriptions and billing
- Optional SMTP delivery for team invitations
- Health and database-readiness endpoints
- Docker Compose deployment support

## How it fits together

```text
                         ┌────────────────────┐
 Browser dashboard ─────►│                    │─────► PostgreSQL
                         │    Envo Go API     │
 Human CLI ─────────────►│                    │─────► AWS KMS
                         │                    │
 Agent or harness ──────►│  isolated agent   │─────► Google OAuth
                         │       API          │
                         └────────────────────┘─────► Razorpay / SMTP / Vercel
```

Humans authenticate with Google and receive short-lived access tokens. Agents use separate opaque credentials beginning with `envo_agent_`. Agent credentials cannot authenticate to human management routes, and human tokens are not used as agent credentials.

When an agent resolves secrets, Envo checks the credential, agent status, and current grants against the database. Suspending the agent or revoking its credential or grant blocks future requests. A value that has already been delivered to a process cannot be recalled, so grants should remain narrow and credentials should be short-lived where practical.

## Technology

| Area | Technology |
| --- | --- |
| Backend | Go, Gin, GORM |
| Frontend | React, TypeScript, Vite, Tailwind CSS |
| CLI | Go, Cobra |
| Database | PostgreSQL |
| Production encryption | AWS KMS with envelope encryption |
| Authentication | Google OAuth 2.0, JWT access and refresh tokens |
| Deployment | Docker Compose and Nginx |

## Repository layout

```text
backend/       Go API, domain services, models, and migrations
cli/           Cross-platform Envo CLI
frontend/      React dashboard and public pages
nginx/         Production SPA and reverse-proxy configuration
docs/          Product, architecture, operations, and development guides
deploy.sh      Early single-server backend deployment helper
docker-compose.yml
```

## Local development

### Requirements

- Go 1.25 or newer
- Node.js 20 or newer
- PostgreSQL
- A Google OAuth client for login testing

AWS KMS is optional during local development. Envo falls back to its development-only local encryptor when KMS is not configured.

### 1. Configure PostgreSQL

Create a local PostgreSQL database and user, then place the connection settings in `backend/.env`. The defaults expect:

```text
host: localhost
port: 5432
database: envo_db
user: envo
```

You can also set a complete `DB_URL` instead of individual database fields.

### 2. Configure and start the backend

```bash
cd backend
cp .env.example .env
```

Before starting the API, edit `backend/.env` and set at least:

```dotenv
ENV=development
JWT_SECRET=replace-with-a-random-development-secret
DB_HOST=localhost
DB_PORT=5432
DB_USER=envo
DB_PASSWORD=your-local-password
DB_NAME=envo_db
DB_SSLMODE=disable
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URL=http://127.0.0.1:8080/api/v1/auth/google/callback
FRONTEND_URL=http://localhost:5173
AWS_KMS_KEY_ID=
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
```

Generate a suitable local JWT secret with:

```bash
openssl rand -hex 32
```

Add the exact `GOOGLE_REDIRECT_URL` to the authorized redirect URIs in Google Cloud Console.

Initialize and start the backend:

```bash
go run ./cmd/server -migrate
go run ./cmd/server -seed
go run ./cmd/server
```

Development mode also applies model migrations during normal startup, but the explicit commands make first-time setup clear.

The API is available at `http://localhost:8080`.

### 3. Start the frontend

In another terminal:

```bash
cd frontend
cp .env.example .env.local
npm ci
npm run dev
```

Set the frontend API URL in `frontend/.env.local`:

```dotenv
VITE_API_URL=http://localhost:8080
```

Vite normally serves the dashboard at `http://localhost:5173`.

### 4. Run the CLI from source

```bash
cd cli
go run ./cmd/envo --api http://localhost:8080 login
go run ./cmd/envo --api http://localhost:8080 whoami
```

## Install the CLI

### macOS and Linux

```bash
curl -fsSL https://raw.githubusercontent.com/scopophobic/envy/main/install.sh | sh
```

### Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/scopophobic/envy/main/install.ps1 | iex
```

See the [installation guide](docs/getting-started/cli-installation.md) for supported platforms, PATH configuration, and release information.

## CLI examples

Authenticate and inspect the current account:

```bash
envo login
envo whoami
```

Use the personal vault. `--org` is optional here:

```bash
envo pull --project api --env development
envo run --project api --env development -- npm run dev
```

Select a team workspace explicitly:

```bash
envo pull --org acme --project api --env production
```

Sync an environment to a configured Vercel connection:

```bash
envo sync --project web --env production \
  --connection vercel-prod \
  --target-project prj_123 \
  --target-env production
```

`envo pull` writes a mode-`0600` `.env` file and ensures the file is ignored by Git. `envo run` keeps secret values off disk and injects them only into the child process.

Complete command documentation is available in the [CLI usage guide](docs/cli/usage.md).

## Agent and coding-harness access

Create an agent identity in an organization, issue a credential, and grant it access to a specific environment and set of keys. The raw credential is shown once.

Provide it to the harness at runtime:

```bash
export ENVO_TOKEN=envo_agent_...

envo agent whoami
envo run \
  --project api \
  --env development \
  --keys DATABASE_URL,TEST_API_KEY \
  -- your-agent-command
```

The CLI removes `ENVO_TOKEN` before starting the child process and injects only the resolved secret values. Agent access uses the isolated `/api/v1/agent/*` API and does not write human login tokens to the CLI token store.

## Security model

- Normal dashboard and API responses contain secret metadata, not plaintext values.
- Production secret values use KMS-generated data keys and AES-256-GCM encryption.
- Refresh tokens are stored as SHA-256 hashes.
- Agent credentials are stored as SHA-256 hashes with a short display prefix.
- OAuth redirects are restricted to the configured frontend origin.
- Production OAuth cookies are `Secure`, `HttpOnly`, and `SameSite=Lax`.
- Organization permissions are enforced on protected resources.
- Sensitive operations create audit records.
- Authentication, secret export, platform sync, and agent resolution are rate-limited.
- API and secret-delivery responses are marked `no-store`.

Local encryption is for development only. Production startup requires a usable AWS KMS key unless the unsafe local-encryption override is explicitly enabled.

Read [SECURITY.md](SECURITY.md) before operating Envo with real credentials. Report vulnerabilities privately and never place credentials, tokens, or secret values in a public issue.

## Performance and reliability

- Short-lived, identity-scoped frontend metadata caching
- In-flight GET request deduplication
- No caching of plaintext secrets, credentials, grants, or authorization decisions
- Bounded PostgreSQL connection pooling
- Parallel KMS decryption with bounded concurrency
- HTTP request-size, header, and timeout limits
- Graceful server shutdown
- Docker health checks backed by database readiness
- Immutable caching for fingerprinted frontend assets

The backend exposes:

```text
GET /health   process liveness
GET /ready    application readiness, including a PostgreSQL ping
```

Example checks:

```bash
curl --fail --show-error http://localhost:8080/health
curl --fail --show-error http://localhost:8080/ready
```

## Production deployment

Production requires:

- HTTPS frontend and Google OAuth callback URLs
- A random JWT secret of at least 32 characters
- PostgreSQL with TLS
- A usable AWS KMS key and appropriate IAM permissions
- A Razorpay webhook signing secret when Razorpay billing is enabled
- Correct reverse-proxy and trusted-proxy configuration

Use `.env.production.example` as the configuration reference. Real production values belong in an ignored environment file or deployment secret store and must never be committed.

The current `deploy.sh` is an early single-EC2 helper. Before relying on Envo for critical production credentials, use readiness-based verification, tested database backups, immutable image versions, and a documented rollback procedure. See the [deployment and operations guide](docs/operations/deployment-and-operations.md).

Use the supported Compose plugin command:

```bash
docker compose --env-file .env.production ps
docker compose --env-file .env.production logs --tail=100 backend
```

The old Python `docker-compose` v1 command is legacy and can fail against newer Docker image metadata.

## Testing

Backend:

```bash
cd backend
go test ./...
go vet ./...
go test -race ./...
```

Frontend:

```bash
cd frontend
npm ci
npm run lint
npm run build
```

CLI:

```bash
cd cli
go test ./...
go vet ./...
```

## Documentation

| Document | Contents |
| --- | --- |
| [Documentation index](docs/README.md) | Entry point for all project documentation |
| [Product overview](docs/product/overview.md) | Product idea, users, positioning, and business model |
| [Personal vault](docs/product/personal-vault.md) | Personal workspace and solo-developer workflows |
| [Roadmap](docs/product/roadmap.md) | Planned capabilities |
| [Technical overview](docs/architecture/technical-overview.md) | Architecture, data model, authentication, encryption, and integrations |
| [API reference](docs/reference/api-endpoints.md) | Routes, permissions, OAuth flow, and troubleshooting |
| [Deployment and operations](docs/operations/deployment-and-operations.md) | Environments, health, monitoring, CI/CD, backups, and rollback |
| [CLI installation](docs/getting-started/cli-installation.md) | Installation and release instructions |
| [CLI usage](docs/cli/usage.md) | CLI commands, tokens, and agent operation |
| [Contribution guide](CONTRIBUTING.md) | Development workflow and contribution rules |
| [Security policy](SECURITY.md) | Security boundaries and vulnerability reporting |

## Contributing

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) for setup, code conventions, tests, and pull-request guidance.

## License

Envo is available under the [MIT License](LICENSE).
