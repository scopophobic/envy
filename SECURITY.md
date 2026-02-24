# Security Policy

## How Envo Handles Secrets

- All secrets are encrypted **before** being written to the database.
- Encryption uses **AES-256-GCM** with data keys generated per secret.
- Data keys are themselves encrypted with **AWS KMS** (envelope encryption).
- In development without KMS, a local encryptor derives keys from the JWT secret. This is **not** suitable for production.
- Secrets are **never** returned in plaintext via the API. The frontend always shows masked values.
- The CLI decrypts secrets locally after pulling encrypted data from the API.

## Authentication

- Google OAuth 2.0 for user authentication.
- JWT access tokens (short-lived, 15 min) + refresh tokens (30 days).
- Refresh tokens are stored hashed in the database and rotated on use.

## Reporting a Vulnerability

If you find a security vulnerability, please **do not** open a public issue.

Email: **security@your-domain.com**

Include:
- Description of the vulnerability
- Steps to reproduce
- Impact assessment

We will respond within 48 hours and work with you on a fix before public disclosure.
