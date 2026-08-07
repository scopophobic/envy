# Security Policy

## How Envo Handles Secrets

- All secrets are encrypted **before** being written to the database.
- Encryption uses **AES-256-GCM** with data keys generated per secret.
- Data keys are themselves encrypted with **AWS KMS** (envelope encryption).
- In development without KMS, a local encryptor derives keys from the JWT secret. This is **not** suitable for production.
- Normal frontend responses contain secret metadata only. The protected export endpoint decrypts values for authorized CLI execution and records an audit event.
- Production startup requires a usable KMS key unless local encryption is explicitly acknowledged as an unsafe override.

## Authentication

- Google OAuth 2.0 for user authentication.
- JWT access tokens (short-lived, 15 min) + refresh tokens (30 days).
- Refresh tokens are stored as SHA-256 hashes. Tokens created by older releases are migrated at startup and remain usable by clients.
- OAuth redirects are restricted to the configured frontend origin.
- Production OAuth state cookies are Secure, HttpOnly, and SameSite=Lax.
- Agent identities use opaque `envo_agent_…` credentials. Only a SHA-256 token hash and a short display prefix are stored.
- Agent tokens authenticate only to the isolated `/api/v1/agent/*` API; they are not accepted as human JWTs.

## Agent Access

- Agents are owned by an organization and can be suspended or permanently revoked.
- Each credential is independently revocable and can have an expiry.
- Grants are scoped to one environment and either named secret keys or an explicit “all secrets” choice.
- The server reloads active grants for every secret resolution; revocation therefore blocks future requests without rotating the underlying secrets.
- Agent secret responses use no-store headers, have bounded request bodies, are rate limited, and record the agent, credential, grant, lease, purpose, session, and secret count in the audit trail.
- `envo run` consumes and strips `ENVO_TOKEN` before starting the child, then injects the approved values. Once delivered, Envo cannot erase those values from that process; brokered upstream requests and dynamic credentials remain the stronger future control model.

## API Protection

- Authentication, secret export, agent resolve, and platform sync endpoints are rate limited.
- Reverse-proxy forwarding headers are accepted only from configured trusted networks.
- API and Nginx responses include clickjacking, MIME-sniffing, referrer, and browser-permission protections.
- Production configuration rejects weak JWT secrets, insecure callback URLs, partial credentials, unavailable KMS configuration, and billing without a webhook signing secret.
- API responses are marked `no-store`. Nginx proxy buffering is disabled so plaintext secret responses are not written to proxy temporary files.
- Request bodies, headers, read/write time, and idle connections are bounded. The server drains active requests during graceful shutdown.

## Caching and Performance Boundaries

- The browser uses a short-lived, identity-scoped in-memory cache for metadata GET requests and deduplicates identical requests already in flight.
- Successful mutations invalidate the entire metadata cache. A generation guard prevents an older in-flight GET from repopulating stale data afterward.
- Decrypted secret values, OAuth login URLs, token validation, agent credentials, and grant decisions are never cached.
- Agent authentication and grants are read from PostgreSQL on every resolve request, preserving immediate revocation.
- Stable tier-limit metadata is cached briefly in the backend. This cache is not used for authentication or authorization.
- KMS-backed secret decryption uses bounded concurrency. Agent “last used” timestamp writes are throttled because they are observability metadata, not a policy input.

## Reporting a Vulnerability

If you find a security vulnerability, please **do not** open a public issue.

Email: **security@your-domain.com**

Include:
- Description of the vulnerability
- Steps to reproduce
- Impact assessment

We will respond within 48 hours and work with you on a fix before public disclosure.
