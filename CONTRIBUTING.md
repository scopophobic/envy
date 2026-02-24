# Contributing to Envo

Thanks for your interest in contributing. This document covers the basics.

## Getting Started

1. Fork the repository
2. Clone your fork locally
3. Set up the development environment (see [README.md](README.md#quick-start-development))
4. Create a feature branch from `main`

```bash
git checkout -b feat/your-feature-name
```

## Project Structure

```
backend/
  cmd/server/          Entry point
  internal/
    config/            Environment configuration
    database/          DB connection + seeding
    handlers/          HTTP route handlers
    middleware/        Auth, CORS, RBAC middleware
    models/            GORM models
    services/          Business logic (encryption, billing, tiers)
    utils/             JWT manager, helpers

frontend/
  src/
    components/        Reusable UI components (Layout, Card, Button)
    lib/               API client, auth helpers
    pages/             Route-level page components

cli/
  cmd/envo/            CLI entry point
  internal/
    commands/          Cobra command definitions (login, pull)
    auth/              CLI auth flow
```

## Development Workflow

### Backend

```bash
cd backend
go run ./cmd/server              # start server
go run ./cmd/server -migrate     # run DB migrations
go run ./cmd/server -seed        # seed tier limits + roles
go build ./...                   # check compilation
```

### Frontend

```bash
cd frontend
npm run dev       # dev server with HMR
npm run build     # production build
npx tsc --noEmit  # type check without emitting
```

### CLI

```bash
cd cli
go run ./cmd/envo login
go run ./cmd/envo pull --org "test" --project "api" --env "dev"
```

## Code Guidelines

### General

- No comments that just narrate what the code does. Comments should explain *why*, not *what*.
- Keep functions small. If a function does more than one thing, split it.
- Error messages should be lowercase and descriptive.

### Go (Backend + CLI)

- Follow standard Go formatting (`gofmt`).
- Use the existing service/handler/model pattern. New features should add a service, a handler, and wire them in `main.go`.
- All secrets must go through the `Encryptor` interface. Never store plaintext.
- Database access goes through GORM. Use `database.GetDB()`.

### TypeScript (Frontend)

- Use TypeScript strictly — no `any` unless absolutely necessary.
- API functions live in `src/lib/api.ts`. Follow the existing `request<T>()` pattern.
- Pages go in `src/pages/`, reusable components in `src/components/`.
- Tailwind CSS only — no separate CSS files.

### Commit Messages

Use clear, concise commit messages:

```
feat: add audit log export endpoint
fix: correct KMS decryption for local-prefix secrets
refactor: extract billing service from handler
```

Prefix with: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`.

## Adding a New Feature

1. **Backend**: Create or update the service in `backend/internal/services/`, add a handler in `handlers/`, wire the route in `cmd/server/main.go`.
2. **Frontend**: Add the API function in `lib/api.ts`, create the page in `pages/`, add the route in `App.tsx`.
3. **CLI**: Add a new Cobra command in `cli/internal/commands/`.

## Adding a Payment Provider

The billing system uses the `PaymentProvider` interface (`backend/internal/services/payment.go`). To add a new provider (e.g., Stripe):

1. Create `stripe_provider.go` implementing `PaymentProvider`
2. Map provider-specific webhook events to the normalised constants (`EventSubscriptionActivated`, etc.)
3. Add config vars and wire in `main.go`

## Pull Requests

- One feature per PR.
- Include a brief description of what changed and why.
- Make sure `go build ./...` and `npx tsc --noEmit` pass.
- If you change the database schema, include the migration step.

## Reporting Issues

Open a GitHub issue with:
- What you expected to happen
- What actually happened
- Steps to reproduce
- Your environment (OS, Go version, Node version)
