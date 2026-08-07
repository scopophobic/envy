# Envo — Product Overview

> The repository is currently named `envy`, while the product and CLI are called **Envo**.

## The short pitch

Envo is a secure credential-control platform for developers, teams, automation, and AI agents.

Developers commonly keep API keys, database passwords, deployment tokens, and other sensitive credentials in `.env` files scattered across laptops, servers, chat messages, and deployment dashboards. This becomes difficult to manage as projects, teammates, machines, and automated agents multiply.

Envo gives every person and organization one controlled place to store credentials, decide who or what can use them, deliver them safely to applications, and record when they are accessed.

## The problem

Modern software projects depend on credentials for services such as:

- Databases
- Payment providers
- AI model providers
- Email services
- Cloud infrastructure
- Analytics and monitoring
- Deployment platforms

These credentials are often copied manually between `.env` files and dashboards. That creates several problems:

- Credentials are forgotten on old machines and projects.
- Teams do not know who has access.
- Production credentials are easily mixed with development credentials.
- New developers spend time reconstructing local environments.
- Rotating a credential requires finding every place it was copied.
- Automated tools and AI agents are frequently given more access than they need.
- There is little evidence of when sensitive credentials were used.

## The Envo solution

Envo organizes credentials in a structure developers already understand:

```text
Workspace
└── Project
    └── Environment
        └── Secrets
```

A project can have separate development, staging, and production environments. Each environment contains only the credentials required for that context.

Developers can retrieve credentials through the Envo command-line tool:

```bash
envo pull --project api --env development
```

Or run an application with its credentials injected without creating a local secret file:

```bash
envo run --project api --env development -- npm run dev
```

Envo can also synchronize an environment directly to a connected Vercel project.

## Personal vaults and organizations

Every user automatically receives a personal vault for solo projects. This makes Envo useful immediately without requiring team setup.

Organizations add collaborative controls:

- Shared projects and environments
- Member invitations
- Owners and administrators
- Developer and viewer roles
- Dedicated secret-management roles
- Custom organization roles
- Permission-controlled access
- Organization audit history

This creates a natural path from individual adoption to paid team usage.

## Designed for the agent era

Envo is not intended to be another AI chat application.

Envo now has the first version of a credential and permission layer between organizations and AI agents. An organization can give an agent limited access such as:

```text
Agent: coding-agent
Project: checkout-api
Environment: development
Credentials: database and test API keys only
Access duration: two hours
Production access: denied
```

The organization can suspend or revoke the agent immediately and review its secret resolutions. Human approval gates for especially sensitive operations are a future layer.

For stronger control, Envo can eventually broker requests so an agent can use a service without ever receiving the underlying credential.

## What works today

Envo currently includes:

- Google login for the dashboard and CLI
- Automatic personal vault creation
- Team organizations and invitations
- Projects and isolated environments
- Encrypted secret creation, editing, deletion, and export
- Built-in and custom roles
- Permission enforcement
- Organization audit logs
- CLI login, identity, pull, run, and sync commands
- Vercel deployment connections and environment synchronization
- Free, Starter, and Team plans
- Razorpay billing integration
- Administrative tier management
- Docker-based production deployment
- Organization-owned agent identities
- One-time, independently revocable agent tokens
- Environment- and secret-key-scoped agent grants
- Agent-attributed audit activity and an `ENVO_TOKEN` coding-harness workflow

## Security position

Security is the product foundation rather than an optional feature.

Envo currently provides:

- Encrypted storage for secret values
- AWS KMS support for production
- Workspace-bound encryption context
- Short-lived access tokens
- Hashed refresh tokens
- Controlled roles and permissions
- Audited secret exports and changes
- Rate limits on sensitive operations
- Restricted login redirects
- Secure production cookies
- Strict production configuration checks
- Immediate failure when production encryption is unavailable

Normal dashboard responses contain secret names and metadata, not their plaintext values. Plaintext is returned only through protected workflows that require permission and generate audit events.

## Who Envo is for

### Solo developers

- Keep credentials available across machines.
- Stop rebuilding `.env` files manually.
- Separate credentials by project and environment.
- Start applications quickly with `envo run`.

### Development teams

- Centralize ownership of project credentials.
- Control access through roles.
- Avoid sharing production credentials through chat.
- Review credential activity.

### Startups

- Establish sensible credential practices early.
- Onboard developers faster.
- Move secrets between development and deployment consistently.
- Add governance as the company grows.

### AI-agent users

- Give agents temporary, narrowly scoped access.
- Separate human and machine identities.
- Revoke access without rotating every credential.
- Require approval before production actions.
- Audit autonomous activity.

The first agent-control layer is implemented. The next milestone is stronger brokering: approvals, short-lived dynamic upstream credentials, and operations where the agent can use a service without receiving its raw key.

## Business model

Envo has a natural product-led structure:

### Free

- Personal vault
- Small team organization
- Core CLI workflows
- Basic credential management

### Starter

- Higher project and usage limits
- More development capacity
- Expanded operational workflows

### Team

- Larger organizations
- Collaboration and governance
- Advanced roles and auditing
- Future approval and agent-management features

### Future enterprise offering

- Organization-wide agent policies
- SSO and SAML
- Approval workflows
- Longer audit retention
- Compliance exports
- Private gateways and self-hosting
- Advanced deployment integrations

AI agents should be measured separately from human seats because a small engineering team may operate many automated agents and agent sessions.

## Competitive position

Envo sits between three categories:

- Developer-friendly `.env` tooling
- Team secret-management platforms
- Emerging identity and governance systems for AI agents

Its differentiation can be the combination of:

- Simple solo-developer onboarding
- A familiar project/environment model
- Strong organization controls
- Excellent CLI experience
- Deployment synchronization
- First-class, temporary access for non-human actors

The aim is to avoid forcing users to choose between convenience and control.

## Example story

A developer starts a new application using a database, an AI provider, a payment service, and an email provider.

They create a project and add development credentials to their personal vault. On another laptop they run:

```bash
envo login
envo pull --project my-app --env development
```

When the developer hires a team, the project moves into an organization. Developers receive access to development, while only selected members can manage production secrets. Changes and exports appear in the audit history.

The team later connects Vercel and synchronizes production configuration from Envo. When coding agents are introduced, each agent receives its own temporary identity and narrowly scoped access instead of receiving a developer's complete `.env` file.

## Current product stage

Envo has a working full-stack foundation: dashboard, backend, database model, encryption, CLI, team permissions, billing, deployment synchronization, and production packaging.

The next stage is about turning that foundation into a dependable commercial product by continuing production hardening, improving secret lifecycle management, expanding deployment integrations, and building controlled identities for agents and automation.

## Recommended one-minute pitch

> Every software project depends on sensitive credentials, but most developers still manage them through scattered `.env` files and deployment dashboards. That is already difficult for teams, and it becomes dangerous when AI agents and automation need access too.
>
> Envo is a secure credential-control platform for developers, organizations, and agents. It stores secrets by project and environment, delivers them through a simple CLI or deployment integration, controls access through roles and policies, and records sensitive activity.
>
> An individual can start with a free personal vault. As they grow into a team, Envo becomes the shared credential system. The next layer gives every AI agent its own temporary, restricted identity so companies can use autonomous tools without handing them permanent production credentials.

## Recommended one-line description

> Envo gives developers, teams, and AI agents secure access to the credentials they need—without giving up organizational control.
