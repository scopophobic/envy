# Envo – MVP Plan (Best-Practice, Low-Cost)

This document defines the **best approach for your product**, now under the brand **Envo** (CLI command: `envo`).

---

## 0. The Best Way (Clear Answer)

> ✅ **CLI token caching + on-demand `.env` generation**

* Minimal compute cost
* Seamless developer UX (no re-login, no copy-paste)
* Supports future runtime-only secrets injection
* Easy CLI ergonomics: `envo login`, `envo pull`, `envo run`

---

## 1. MVP Scope

### Admin Dashboard

* Create orgs, projects, environments
* Add secrets
* Invite developers by email
* Audit logs

### CLI Tool (`envo`)

* Login once, token cached 15–30 mins
* Fetch secrets and write `.env` automatically
* Auto-add `.env` to `.gitignore`
* Pull secrets per project/environment

### Backend API (Go)

* Auth, role-based access (Admin/Developer)
* Secret encryption/decryption via AWS KMS
* Tier enforcement (projects/orgs/dev limits)
* Audit logging

---

## 2. Core Workflow (Ideal UX)

### Admin

1. Create org → project → environment
2. Add secrets
3. Invite developers via email

### Developer

```bash
envo login
envo pull --project api --env dev
envo run npm start
```

* CLI checks cached token
* Refreshes automatically if expired
* Writes secrets to `.env`
* No repeated login, no manual copy-paste

---

## 3. CLI Token Caching

* Token TTL: 15–30 mins
* Stored securely (OS keychain or encrypted file)
* Only refreshed on CLI usage
* Negligible compute cost for server or local machine

---

## 4. `.env` File Strategy

* Written on-demand by CLI
* Auto-overwrites existing `.env`
* Auto-added to `.gitignore`
* Reduces friction for devs

**Security note:** Devs can access env locally, but central control + audit prevents accidental leaks.

---

## 5. Tier & Limits

| Tier    | Price      | Devs | Projects  | Orgs      |
| ------- | ---------- | ---- | --------- | --------- |
| Free    | $0         | 2    | 1         | 1         |
| Starter | $10/month  | 8    | 3–5       | 1         |
| Team    | $20+/month | 20+  | Unlimited | Unlimited |

> Tiered limits prevent abuse, encourage upgrades, and make costs predictable.

---

## 6. Tech Stack

| Layer       | Technology                    | Notes                                  |
| ----------- | ----------------------------- | -------------------------------------- |
| Backend API | Go (Fiber/Gin)                | REST API for CLI and dashboard         |
| CLI         | Go                            | Single static binary, uses Cobra/Viper |
| Database    | PostgreSQL                    | Encrypted secrets, audit logs          |
| Encryption  | AWS KMS                       | Envelope encryption                    |
| Frontend    | React + TypeScript + Tailwind | Admin dashboard                        |
| Auth        | JWT + Magic Link/OAuth        | Developer login & session management   |
| Email       | SendGrid / AWS SES            | Invitation emails                      |

---

## 7. Infrastructure & Cost (Cheapest Sensible Setup)

| Component       | Monthly Cost |
| --------------- | ------------ |
| EC2 t3.small    | ~$20         |
| RDS db.t3.micro | ~$15         |
| AWS KMS         | ~$1–2        |
| Domain          | ~$5          |
| Email           | Free         |
| **Total**       | **~$40–50**  |

> Optionally, use Fly.io or Railway (~$15–25) for cheaper MVP hosting.

---

## 8. Security Model

* TLS everywhere
* Secrets encrypted at rest
* Short-lived tokens for CLI
* Audit logs for all secret fetches
* Role-based access (Admin/Developer)
* Prevents accidental leaks, central revocation possible

---

## 9. Phase 2 Roadmap

* Runtime-only secret injection (no `.env` on disk)
* Enhanced access policies
* Enterprise SSO support
* Secret rotation automation

---

## 10. Branding & CLI Command

* Product name: **Envo**
* CLI command: `envo`
* Domain options: `getenvo.dev`, `envo.dev` (if available), `envo.cloud`
* Positioning sentence:

> **Envo – CLI-first platform for securely sharing and using env secrets across teams.**

---

## 11. Next Steps

1. Reserve domain & GitHub org (e.g., `getenvo`)
2. CLI skeleton in Go (`envo login`, `envo pull`)
3. Backend API MVP
4. Minimal admin dashboard
5. Implement sharing workflow first, runtime injection later

---

**This plan provides a full MVP blueprint: secure, low-cost, dev-first, and ready for scale under the brand Envo.**
