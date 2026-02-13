# Envo — Future Scope

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
- Custom roles (beyond the 5 system roles).
- Per-project or per-environment permissions (currently org-wide).

## SSO / SAML
- Enterprise SSO support (Okta, Azure AD).

## Audit Log Improvements
- Search/filter audit logs.
- Export as CSV.
- Longer retention for paid tiers.

## Self-Hosted Option
- Docker Compose setup for self-hosting.
- Helm chart for Kubernetes.
