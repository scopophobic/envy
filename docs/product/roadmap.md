# Envo — Future Scope

## Shipped Baseline

- Personal and team workspaces, RBAC, custom organization roles, and audit logs.
- CLI `pull` and `run`, with automatic personal-vault selection when `--org` is omitted.
- Encrypted Vercel connections and manual environment sync from the web app or CLI.

## CLI Push
- `envo push --org <org> --project <project> --env <env>` — Upload a local `.env` file to the server, creating/updating secrets in bulk.
- Requires a new backend endpoint: `POST /api/v1/environments/:id/secrets/import`
- Should show a diff before applying (which keys will be added, updated, or removed).

## GitHub Actions Integration
- `envo-action` GitHub Action to pull secrets into CI/CD pipelines.
- Supports `ENVO_TOKEN` for service account auth (no browser login).

## Versioned Secrets
- Keep history of secret values (who changed what, when).
- Rollback to a previous version.

## Secret Rotation Alerts
- Notify team when a secret hasn't been rotated in X days.
- Configurable per environment.

## Webhook Notifications
- Fire webhooks when secrets change (for automated deployments).

## RBAC Improvements
- Per-project or per-environment permissions (currently org-wide).

## Controlled Agent Access
- Organization-owned agent identities, separate from human users.
- Hashed, expiring tokens restricted to selected projects and environments.
- Separate metadata, read, inject, write, and deployment permissions.
- Short-lived access grants, approval gates for production, and immediate revocation.
- Agent-aware audit events and a safe MCP interface that avoids returning raw values by default.

## Deployment Integrations
- Vercel is available today.
- Add Railway, Render, and Fly.io adapters behind the existing sync framework.
- Move deployment connections from individual ownership to organization ownership for team use.

## SSO / SAML
- Enterprise SSO support (Okta, Azure AD).

## Audit Log Improvements
- Search/filter audit logs.
- Export as CSV.
- Longer retention for paid tiers.

## Self-Hosted Option
- Docker Compose setup for self-hosting.
- Helm chart for Kubernetes.
