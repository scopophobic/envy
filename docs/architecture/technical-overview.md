# Envo — Technical Overview

> The repository is currently named `envy`, while the product, API service, and CLI are called **Envo**.

## System summary

Envo is a full-stack secret-management system consisting of:

```text
React dashboard
      │
      ▼
Go REST API ───── PostgreSQL
      │
      ├────────── AWS KMS
      ├────────── Google OAuth
      ├────────── Razorpay
      ├────────── SMTP
      └────────── Vercel API

Go CLI ────────── Go REST API
```

The main deployment uses Docker Compose with Nginx serving the React application and proxying API requests to the Go backend.

## Repository structure

```text
backend/
  cmd/server/              API entry point and route registration
  internal/config/         Environment configuration and validation
  internal/database/       PostgreSQL connection and initial data
  internal/handlers/       HTTP request handlers
  internal/middleware/     Authentication, permissions, CORS, rate limits
  internal/models/         GORM database models and migrations
  internal/services/       Business logic, encryption, billing, integrations
  internal/utils/          JWT creation and validation

frontend/
  src/components/          Shared layout and UI elements
  src/lib/                 API client, authentication, pricing
  src/pages/               Dashboard pages

cli/
  cmd/envo/                CLI entry point
  internal/api/            Backend API client
  internal/commands/       login, logout, whoami, pull, run, sync
  internal/config/         CLI API configuration
  internal/dotenv/         .env serialization and parsing
  internal/store/          Local token storage

nginx/                     SPA server and API reverse proxy
docker-compose.yml         Production/local service definitions
```

## Technology stack

### Backend

- Go
- Gin HTTP framework
- GORM
- PostgreSQL
- AWS SDK for Go v2
- Google OAuth 2.0
- JWT access and refresh tokens
- Razorpay SDK

### Frontend

- React 19
- TypeScript
- Vite
- Tailwind CSS
- React Router

### CLI

- Go
- Cobra
- Cross-platform release scripts

### Infrastructure

- Docker and Docker Compose
- Nginx
- PostgreSQL or Supabase Postgres
- AWS KMS for production encryption

## Domain model

The central hierarchy is:

```text
User
├── Owned workspaces
├── Organization memberships
├── Refresh tokens
└── Audit events

Organization
├── Owner
├── Members
├── Invitations
├── Roles
└── Projects
    └── Environments
        └── Secrets
```

Additional models include:

- Permissions
- Tier limits
- CLI login codes
- Platform connections
- Audit logs

UUIDs are used for primary identifiers. Most business resources use soft deletion where recovery or audit continuity is useful.

## Workspace types

Organizations use an `owner_type` field:

- `personal`: automatically created private vault
- `org`: collaborative team workspace

Personal workspaces block team invitation operations. Team organizations use membership and role records.

## Authentication

### Browser login

1. The frontend sends the browser to the backend Google OAuth redirect endpoint.
2. The backend creates a cryptographically random OAuth state value.
3. State and flow information are stored in HttpOnly cookies.
4. Google redirects back to the backend callback.
5. The backend validates the OAuth state and exchanges the authorization code.
6. The backend finds or creates the user and their personal workspace.
7. Access and refresh tokens are generated.
8. Tokens are delivered to the frontend callback through a URL fragment.
9. The frontend stores them and navigates to the dashboard.

Production OAuth cookies use `Secure`, `HttpOnly`, and `SameSite=Lax`. Redirect targets must match the configured frontend origin.

The browser currently stores its tokens in `localStorage`. Moving the refresh token to an HttpOnly cookie is an important remaining hardening task.

### CLI login

1. The CLI opens a temporary localhost callback server.
2. It opens the backend CLI OAuth URL in the browser.
3. Google authentication completes through the backend.
4. The backend redirects a short-lived, one-time exchange code to localhost.
5. The CLI exchanges the code for access and refresh tokens.
6. Tokens are saved in the operating system's application configuration directory.

CLI callbacks are restricted to `http://localhost` or `http://127.0.0.1`.

### Token storage

- Access tokens are short-lived JWTs.
- Refresh tokens are JWTs returned to clients.
- The database stores only a SHA-256 hash of each refresh token.
- Plaintext rows from older releases are automatically migrated at backend startup.
- Refresh tokens can be revoked during logout.

Refresh-token rotation and replay detection are not yet implemented.

## Authorization

Requests pass through JWT authentication middleware, which loads the current user and their organization memberships.

Permission middleware resolves the organization through the resource hierarchy:

```text
Secret → Environment → Project → Organization
```

Available permissions currently include:

- `secrets.read`
- `secrets.create`
- `secrets.update`
- `secrets.delete`
- `projects.manage`
- `environments.manage`
- `members.invite`
- `members.manage`
- `audit.view`
- `org.manage`

The system includes Owner, Admin, Secret Manager, Developer, and Viewer roles. Organizations can create custom roles from the available permissions.

Permissions are currently organization-wide. Project-, environment-, and key-specific policies are planned.

## Secret encryption

### Production encryption

Production uses AWS KMS envelope encryption:

1. Envo asks KMS to generate a 256-bit data key.
2. The plaintext data key encrypts the secret using AES-256-GCM.
3. Workspace identity is included in the encryption context and authenticated data.
4. The plaintext data key is cleared from its byte buffer after use.
5. Envo stores the encrypted data key and encrypted secret together.
6. Decryption requires access to the same KMS key and workspace context.

At startup, production performs a real `GenerateDataKey` and `Decrypt` round trip. Startup fails if KMS is unavailable or permissions are insufficient.

AWS credentials can come from explicit environment variables or the normal AWS credential chain, including EC2/ECS IAM roles.

### Development encryption

Local development uses AES-256-GCM with keys derived from `JWT_SECRET` using HKDF and the workspace ID.

This mode is not considered suitable for production. Production refuses to use it unless `ALLOW_LOCAL_ENCRYPTION_IN_PRODUCTION=true` is explicitly configured.

### Secret responses

Normal secret-listing responses contain:

- Secret ID
- Environment ID
- Key name
- Creator
- Creation and update timestamps

They do not contain the encrypted or plaintext value.

The protected export endpoint decrypts an environment for authorized CLI and synchronization workflows. Exports are permission checked, rate limited, and audited.

## Secret operations

Implemented operations:

- List secret metadata
- Create secret
- Upsert when a key already exists
- Update key and/or value
- Soft delete
- Permanent purge
- Export an environment
- Bulk paste through the frontend
- Synchronize an environment to Vercel

Secret version history and rollback are not currently implemented.

## Audit system

Audit records include:

- User
- Organization
- Action
- Resource type and ID
- Metadata
- IP address
- Timestamp

Secret creation, updates, deletion, purge, and export activity are recorded. Audit search, filtering, CSV export, configurable retention, and non-human actor identities remain future work.

## CLI behavior

### `envo pull`

- Resolves the workspace, project, and environment by ID or name.
- Uses the personal vault when `--org` is omitted.
- Exports the environment through the protected API.
- Serializes values into `.env` syntax.
- Writes the file with mode `0600`.
- Adds `.env` to `.gitignore` if required.

### `envo run`

- Resolves and exports an environment.
- Starts a child process.
- Adds secret values directly to the child environment.
- Does not create a secret file.
- Connects the child to the current standard input, output, and error streams.

### `envo sync`

- Resolves an environment and deployment connection.
- Sends a manual synchronization request to the backend.
- Currently targets Vercel.

### CLI configuration

The API base URL can be configured with:

```bash
export ENVO_API_URL=https://api.example.com
```

Or per command:

```bash
envo --api https://api.example.com whoami
```

## Vercel integration

Platform connection records contain:

- User ID
- Platform
- Friendly name
- Encrypted token
- Encryption key identifier
- Display-safe token prefix
- Optional metadata

Vercel tokens are validated before being stored and encrypted using the configured Envo encryptor.

During synchronization, the backend:

1. Confirms that the connection belongs to the current user.
2. Decrypts the connection token.
3. Exports the selected Envo environment.
4. Creates or updates variables in the target Vercel project and environment.

Connections are currently user-owned. Organization ownership and a dedicated `deploy.sync` permission are planned.

## Billing and tiers

The billing layer uses a provider abstraction with a Razorpay implementation.

Implemented capabilities include:

- Billing status
- Subscription checkout
- Customer portal
- Standard payment orders
- Payment signature verification
- Webhook handling
- Plan pricing lookup
- User tier updates

Tier limits cover organizations, projects, team members, environments, secrets, and API usage. Initial limits are seeded into PostgreSQL.

When billing is enabled in production, a webhook signing secret is required.

## API protection

### Rate limiting

Process-local fixed-window limits protect:

- Authentication endpoints by IP and route
- Secret exports by authenticated user and route
- Platform synchronization by authenticated user and route
- Agent secret resolution by agent credential and route

Defaults:

```text
Authentication: 30 requests per minute
Secret exports: 30 requests per minute
Platform sync: 10 requests per minute
Agent secret resolve: 60 requests per minute
```

Responses include `RateLimit-Limit`, `RateLimit-Remaining`, `RateLimit-Reset`, and `Retry-After` where applicable.

The current limiter is appropriate for the single-backend Docker deployment. Multiple backend replicas will require shared state such as Redis.

### Trusted proxies

Gin only accepts forwarded client-address headers from networks configured through `TRUSTED_PROXIES`. Docker Compose defaults this to the private Docker address range used by the Nginx proxy.

### Security headers

The backend and Nginx add:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: no-referrer`
- A restrictive browser permissions policy

### Error responses

Unexpected server errors are logged internally and returned to clients as generic messages. Each request receives an `X-Request-ID` so operators can correlate client reports with server logs without exposing database, encryption, OAuth, or provider details.

### Safe caching and request efficiency

The frontend keeps authenticated metadata GET responses in an identity-scoped in-memory cache for 12 seconds by default. Concurrent identical GETs share one network request. Any successful mutation invalidates the cache, and a generation counter prevents an older in-flight response from restoring stale data.

This cache never includes exported secret values or OAuth login URLs. Browser and proxy HTTP caches are bypassed for all API traffic.

The backend briefly caches stable tier-limit rows. It deliberately does not cache agent token validation or grants: those database checks still happen on every resolve so revocation remains immediate.

Secret exports decrypt independent envelope-encrypted values concurrently with a bounded worker count. Agent last-used timestamps are written at a throttled interval because they are observability fields rather than security inputs.

### Reliability controls

- Configurable PostgreSQL connection pool with startup ping
- Read-header, read, write, and idle HTTP timeouts
- Header and request-body size limits
- SIGTERM/SIGINT graceful shutdown with request draining
- Separate process-liveness and database-readiness probes
- Nginx connection timeouts and disabled API response buffering
- One-year immutable cache headers for fingerprinted frontend assets
- No-store HTML and API responses

## Production configuration validation

Production refuses to start when:

- `JWT_SECRET` is missing, weak, or still a placeholder.
- Token expiry durations are invalid.
- The refresh duration is not longer than the access duration.
- Database configuration is incomplete.
- Google OAuth credentials are absent.
- Frontend or OAuth callback URLs are not HTTPS.
- KMS is absent without an explicit override.
- KMS cannot generate and decrypt data keys.
- Only one AWS static credential is provided.
- Only one Razorpay credential is provided.
- Billing is enabled without a webhook signing secret.
- Enabled rate limits are invalid.
- Trusted proxy syntax is invalid.

The reference configuration is in `.env.production.example`.

## Main API groups

Public endpoints include:

- Health and ping
- Google OAuth initiation and callback
- CLI login initiation and code exchange
- Access-token refresh
- Logout
- Razorpay webhook

Authenticated endpoint groups include:

- Current user and tier information
- Organizations, members, invitations, and roles
- Projects and environments
- Secret metadata and operations
- Environment export and deployment sync
- Platform connections
- Audit logs
- Billing operations
- Platform administration

The full route list is maintained in the [API endpoint reference](../reference/api-endpoints.md).

## Frontend routes

The React application contains routes for:

- Landing
- Pricing
- Login
- OAuth callback
- Invitation acceptance
- Workspace list
- Organization detail
- Members and roles
- Project detail
- Environment and secret detail
- Settings and deployment connections
- User invitations
- Super-admin management

Protected pages require a locally available access token and use the shared API client for proactive refresh and retry behavior.

## Deployment

The Docker Compose deployment includes:

- Optional local PostgreSQL
- Go backend
- Nginx/React frontend

Nginx:

- Serves the compiled React SPA.
- Proxies `/api/` and `/health` to the backend.
- Preserves real-client forwarding information.
- Compresses suitable responses.
- Caches static assets.
- Adds browser security headers.

Production can also use an external PostgreSQL database such as Supabase through `DB_URL`.

## Important environment variables

```text
ENV
PORT
TRUSTED_PROXIES

DB_URL
DB_HOST
DB_PORT
DB_USER
DB_PASSWORD
DB_NAME
DB_SSLMODE
DB_MAX_OPEN_CONNS
DB_MAX_IDLE_CONNS
DB_CONN_MAX_LIFETIME
DB_CONN_MAX_IDLE_TIME

JWT_SECRET
JWT_ACCESS_TOKEN_EXPIRY
JWT_REFRESH_TOKEN_EXPIRY

GOOGLE_CLIENT_ID
GOOGLE_CLIENT_SECRET
GOOGLE_REDIRECT_URL
FRONTEND_URL

AWS_REGION
AWS_KMS_KEY_ID
AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY
ALLOW_LOCAL_ENCRYPTION_IN_PRODUCTION

RATE_LIMIT_ENABLED
AUTH_RATE_LIMIT_PER_MINUTE
SECRET_EXPORT_RATE_LIMIT_PER_MINUTE
PLATFORM_SYNC_RATE_LIMIT_PER_MINUTE
AGENT_RESOLVE_RATE_LIMIT_PER_MINUTE

HTTP_READ_HEADER_TIMEOUT
HTTP_READ_TIMEOUT
HTTP_WRITE_TIMEOUT
HTTP_IDLE_TIMEOUT
HTTP_SHUTDOWN_TIMEOUT
HTTP_MAX_HEADER_BYTES
MAX_REQUEST_BODY_BYTES

TIER_CACHE_TTL
SECRET_DECRYPT_CONCURRENCY
AGENT_USAGE_WRITE_INTERVAL

RAZORPAY_KEY_ID
RAZORPAY_KEY_SECRET
RAZORPAY_WEBHOOK_SECRET
RAZORPAY_PLAN_STARTER
RAZORPAY_PLAN_TEAM

SMTP_HOST
SMTP_PORT
SMTP_USERNAME
SMTP_PASSWORD
SMTP_FROM_EMAIL
SMTP_FROM_NAME
INVITE_TOKEN_TTL_HOURS
```

Secrets and real credentials must remain in ignored `.env` files or the deployment secret store. The repository contains example files with placeholders.

## Verification

Current verification commands include:

```bash
cd backend
go test ./...
go vet ./...

cd ../cli
go test ./...

cd ../frontend
npm run lint
npm run build

cd ..
docker compose config -q
```

Backend security-sensitive packages have also been verified under the Go race detector.

Tests currently cover areas including:

- Organization permission isolation
- Invitation token generation and hashing
- Razorpay payment signatures
- Refresh-token hashing
- Production configuration validation
- OAuth redirect restrictions and cookie attributes
- Rate limiting
- CLI personal-workspace resolution

The test suite is still small relative to the security sensitivity of the product. Broader integration and authorization tests remain a priority.

## Current technical limitations

### Authentication

- Browser tokens are stored in `localStorage`.
- Refresh tokens are not rotated on use.
- Replay detection and device/session management are absent.

### Authorization

- Permissions are primarily organization-wide.
- Deployment sync currently relies on secret-read permission.
- Agent grants currently scope secret injection by environment and key; human permissions remain primarily organization-wide.

### Secret lifecycle

- No version history or rollback.
- No scheduled rotation or expiration.
- No change webhooks.
- No last-used tracking per secret.

### Automation

- Agent tokens and the `ENVO_TOKEN` execution flow are implemented; general-purpose service accounts are not.
- No GitHub Actions integration.
- No structured JSON mode across CLI commands.
- No `push`, `init`, `diff`, or `status` command.

### Operations

- Rate-limit state is local to one backend process.
- Audit search and export are limited.
- Monitoring, alerting, backup validation, and disaster recovery need formalization.
- SSO/SAML and enterprise retention controls are absent.

### Integrations

- Only Vercel is currently supported.
- Railway, Render, Fly.io, Kubernetes, and Terraform adapters are future work.

## Controlled agent-access architecture (implemented foundation)

The current agent layer introduces:

```text
AgentIdentity
├── Organization ownership
├── Enabled/suspended state
├── Creator
└── Last-used information

AgentCredential
├── Token hash and display prefix
├── Expiration and revocation
├── Independent expiration and revocation
└── Link to an organization-owned agent

AccessGrant
├── Agent
├── Environment scope
├── Named secret keys or explicit all-secret access
├── Expiration and revocation
└── `secrets.inject` capability
```

The implemented management permission and capability are:

```text
agents.manage
secrets.inject
```

Agent credentials authenticate only to `/api/v1/agent/*`. Every resolve reloads the live grant, decrypts only approved keys, emits an agent-attributed audit event, and returns a no-store response. The CLI consumes this through `ENVO_TOKEN` and `envo run`; it never saves the agent token to its human login store.

Controlled execution still delivers plaintext into the child process environment, so it cannot recall a value already received. Request brokering, approval policies, dynamic credentials, and MCP as an interface over this identity layer are the next security stages.

## Recommended engineering sequence

1. Move browser refresh tokens to HttpOnly cookies.
2. Add refresh-token rotation, token families, and replay detection.
3. Add request-body limits and broader abuse protection.
4. Create a dedicated deployment-sync permission.
5. Add project- and environment-scoped authorization.
6. ✅ Add organization-owned agent identities.
7. ✅ Issue hashed, independently revocable, expiring agent credentials.
8. ✅ Record human and agent actors in audit events.
9. Add approval-gated production grants and general service identities.
10. Expose controlled capabilities through CLI, API, and MCP.
11. Add secret versions, rollback, and rotation workflows.
12. Expand integrations and operational monitoring.
