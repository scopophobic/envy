# Envo API Endpoints & Request Flow Map

## Network Layers (Production with Host Nginx)

```
Browser
  |
  v
Host Nginx (Ubuntu service, port 443/80)
  | server_name api.envo.scopophobic.xyz
  | proxy_pass http://127.0.0.1:8080
  v
Docker envy_nginx_1 (container port 80, host port 8080)
  | location /api/   -> proxy_pass http://backend:8080
  | location /health -> proxy_pass http://backend:8080
  | location /       -> serves React SPA (index.html)
  v
Docker envy_backend_1 (Go/Gin on container port 8080)
  | All /api/v1/... routes handled here
  v
Supabase Postgres (via DB_URL)
```

---

## All Backend API Endpoints

### Public (no auth required)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/health` | inline | Health check |
| GET | `/api/v1/ping` | inline | Ping/pong |
| GET | `/api/v1/auth/google/login` | `GoogleLogin` | Returns Google OAuth URL as JSON |
| GET | `/api/v1/auth/google/redirect` | `GoogleLoginRedirect` | Browser redirect to Google OAuth (sets cookies, accepts `?next=`) |
| GET | `/api/v1/auth/google/callback` | `GoogleCallback` | Google redirects here after login; exchanges code for tokens |
| GET | `/api/v1/auth/cli/google/start` | `CLIGoogleStart` | CLI login flow start (accepts `?callback=`) |
| POST | `/api/v1/auth/cli/exchange` | `CLIExchange` | CLI exchanges one-time code for tokens |
| POST | `/api/v1/auth/refresh` | `RefreshToken` | Refresh access token |
| POST | `/api/v1/auth/logout` | `Logout` | Revoke refresh token |
| POST | `/api/v1/billing/webhook` | `HandleWebhook` | Razorpay webhook (if billing enabled) |

### Protected (JWT required)

| Method | Path | Handler | Permission | Description |
|--------|------|---------|------------|-------------|
| GET | `/api/v1/auth/me` | `GetCurrentUser` | - | Current user info |
| GET | `/api/v1/auth/tier-info` | `GetTierInfo` | - | Tier limits + usage |
| GET | `/api/v1/orgs` | `ListOrganizations` | - | List user's orgs |
| POST | `/api/v1/orgs` | `CreateOrganization` | - | Create org |
| GET | `/api/v1/orgs/:id` | `GetOrganization` | - | Get org details |
| PATCH | `/api/v1/orgs/:id` | `UpdateOrganization` | `org:manage` | Update org |
| DELETE | `/api/v1/orgs/:id` | `DeleteOrganization` | `org:manage` | Delete org |
| POST | `/api/v1/orgs/:id/members` | `InviteMember` | `members:invite` | Invite member |
| PATCH | `/api/v1/orgs/:id/members/:memberId` | `UpdateMemberRole` | `members:manage` | Change member role |
| DELETE | `/api/v1/orgs/:id/members/:memberId` | `RemoveMember` | `members:manage` | Remove member |
| GET | `/api/v1/orgs/:id/projects` | `ListOrgProjects` | - | List org projects |
| POST | `/api/v1/orgs/:id/projects` | `CreateProject` | `projects:manage` | Create project |
| GET | `/api/v1/projects/:id` | `GetProject` | - | Get project |
| PATCH | `/api/v1/projects/:id` | `UpdateProject` | `projects:manage` | Update project |
| DELETE | `/api/v1/projects/:id` | `DeleteProject` | `projects:manage` | Delete project |
| GET | `/api/v1/projects/:id/environments` | `ListProjectEnvironments` | - | List environments |
| POST | `/api/v1/projects/:id/environments` | `CreateEnvironment` | `environments:manage` | Create environment |
| GET | `/api/v1/environments/:id` | `GetEnvironment` | - | Get environment |
| PATCH | `/api/v1/environments/:id` | `UpdateEnvironment` | `environments:manage` | Update environment |
| DELETE | `/api/v1/environments/:id` | `DeleteEnvironment` | `environments:manage` | Delete environment |
| GET | `/api/v1/environments/:id/secrets` | `ListSecrets` | `secrets:read` | List secrets |
| POST | `/api/v1/environments/:id/secrets` | `CreateSecret` | `secrets:create` | Create secret |
| PATCH | `/api/v1/secrets/:id` | `UpdateSecret` | `secrets:update` | Update secret |
| DELETE | `/api/v1/secrets/:id` | `DeleteSecret` | `secrets:delete` | Delete secret |
| GET | `/api/v1/environments/:id/secrets/export` | `ExportEnvironmentSecrets` | `secrets:read` | Export decrypted secrets (CLI) |
| GET | `/api/v1/orgs/:id/audit-logs` | `ListOrgAuditLogs` | `audit:view` | Audit logs |
| POST | `/api/v1/billing/checkout` | `CreateCheckoutSession` | - | Start billing checkout |
| POST | `/api/v1/billing/portal` | `CreatePortalSession` | - | Billing portal |

---

## Frontend Routes

| Path | Component | Description |
|------|-----------|-------------|
| `/` | `LandingPage` | Public landing |
| `/login` | `LoginPage` | Google login button |
| `/auth/callback` | `AuthCallbackPage` | Receives tokens from backend redirect |
| `/orgs` | `OrgsPage` | List orgs (protected) |
| `/orgs/:id` | `OrgDetailPage` | Org detail (protected) |
| `/orgs/:id/members` | `MembersPage` | Members (protected) |
| `/projects/:id` | `ProjectDetailPage` | Project detail (protected) |
| `/environments/:id` | `EnvironmentDetailPage` | Environment detail (protected) |
| `/settings` | `SettingsPage` | User settings (protected) |

---

## Google OAuth Login Flow (the critical path)

### Step-by-step

1. User clicks "Continue with Google" on `LoginPage`
2. Frontend JS runs:
   - `callbackUrl = window.location.origin + "/auth/callback"`
   - Redirects browser to: `VITE_API_URL + "/api/v1/auth/google/redirect?next=" + callbackUrl`
3. Backend `GoogleLoginRedirect` handler:
   - Sets cookies: `oauth_state`, `oauth_flow=web`, `post_login_next=<callbackUrl>`
   - Redirects browser to Google OAuth consent screen
4. User consents on Google
5. Google redirects browser to: `GOOGLE_REDIRECT_URL` (i.e. `/api/v1/auth/google/callback?code=...&state=...`)
6. Backend `GoogleCallback` handler:
   - Verifies state cookie
   - Exchanges code for Google tokens
   - Creates/finds user in DB
   - Generates JWT access + refresh tokens
   - Reads `post_login_next` cookie (or falls back to `FRONTEND_URL + "/auth/callback"`)
   - Redirects browser to: `<frontendCallback>#access_token=...&refresh_token=...`
7. Frontend `AuthCallbackPage`:
   - Reads tokens from URL hash
   - Stores in localStorage
   - Navigates to `/orgs`

---

## Why "Bad Gateway" on Google OAuth callback

### The issue

When Google redirects the user back to:
`https://api.envo.scopophobic.xyz/api/v1/auth/google/callback?code=...&state=...`

This request hits:
1. Host Nginx (api.envo.scopophobic.xyz:443)
2. -> `proxy_pass http://127.0.0.1:8080`
3. -> Docker envy_nginx_1 (listening on host 8080, container 80)
4. -> `location /api/` -> `proxy_pass http://backend:8080`
5. -> Go backend handles `/api/v1/auth/google/callback`

A **502 Bad Gateway** at step 2 means `envy_nginx_1` is not reachable on host port 8080.
A **502 Bad Gateway** at step 4 means `envy_backend_1` is not reachable inside Docker network.

### Diagnosis checklist

```bash
# Is the Envo nginx container running and bound to 8080?
docker ps --format '{{.Names}} {{.Ports}}'
# Expected: envy_nginx_1 0.0.0.0:8080->80/tcp

# Can host reach container nginx?
curl -I http://127.0.0.1:8080
# Expected: 200 (serves React app)

# Can host reach backend through container nginx?
curl -I http://127.0.0.1:8080/health
# Expected: 200 JSON {"status":"ok",...}

# Can host reach backend API?
curl http://127.0.0.1:8080/api/v1/ping
# Expected: {"message":"pong"}

# Check backend logs for errors
docker-compose --env-file .env.production logs backend --tail=100
```

### Common fixes

1. **Container not running**: `docker-compose --env-file .env.production up -d backend nginx`
2. **Wrong HOST_PORT**: ensure `.env.production` has `HOST_PORT=8080` matching Nginx `proxy_pass`
3. **Backend crashed** (DB connection / KMS init failure): check `docker-compose logs backend`
4. **Host Nginx not reloaded** after config change: `sudo nginx -t && sudo systemctl reload nginx`

---

## Environment Variables That Affect OAuth Flow

| Variable | Where Used | Must Match |
|----------|-----------|------------|
| `VITE_API_URL` | Frontend build (baked in) | Public backend URL that browser can reach |
| `FRONTEND_URL` | Backend CORS + post-login redirect | Public frontend URL (exact origin) |
| `GOOGLE_REDIRECT_URL` | Backend OAuth config + Google Console | Must match Google Console "Authorized redirect URIs" exactly |
| `GOOGLE_CLIENT_ID` | Backend OAuth config | Must match Google Console |
| `GOOGLE_CLIENT_SECRET` | Backend OAuth config | Must match Google Console |

### Current values in .env.production

```
VITE_API_URL=https://api.envo.scopophobic.xyz
FRONTEND_URL=https://envo.scopophobic.xyz
GOOGLE_REDIRECT_URL=https://api.envo.scopophobic.xyz/api/v1/auth/google/callback
```

### Important: domains must be consistent

- Frontend is served at: `https://api.envo.scopophobic.xyz` (via Docker nginx on HOST_PORT 8080)
- CORS allows: `https://envo.scopophobic.xyz`
- Post-login redirect goes to: `https://envo.scopophobic.xyz/auth/callback`

If the frontend is actually served from `https://api.envo.scopophobic.xyz` (same domain as backend),
then `FRONTEND_URL` should be `https://api.envo.scopophobic.xyz` (not `envo.scopophobic.xyz`).

Otherwise CORS will block frontend API calls, and the post-login redirect will go to the wrong domain.
