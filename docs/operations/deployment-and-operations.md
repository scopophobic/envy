# Envo Deployment, Environments, Health, and CI/CD Guide

This guide explains how Envo currently runs, what `deploy.sh` does, how to check whether the service is healthy, and how to evolve from manual deployment into a safe development and production pipeline.

It is written for the current Envo architecture:

- React frontend, deployed separately (currently suitable for Vercel)
- Go backend in Docker on one AWS EC2 instance
- PostgreSQL outside the application server (for example Supabase)
- AWS KMS for production secret encryption
- Docker Compose for backend lifecycle management
- Host-level Nginx or another reverse proxy in front of the backend

## 1. The basic concepts

These words are related, but they do not mean the same thing.

### Build

A build turns source code into something runnable. For Envo this means:

- Compiling the Go backend into `envo-server`
- Compiling the React frontend into static assets
- Packaging either application into a Docker image or deployment artifact

A build should be reproducible. The same Git commit should produce the same application behavior.

### Release

A release is a version that has passed validation and is considered deployable. A release should have an immutable identifier such as:

- Git commit SHA: `a4c981e`
- Semantic version: `v0.4.0`
- Docker image: `ghcr.io/owner/envo-backend:a4c981e`

Do not rely only on an image called `latest`. It does not tell you what code is running and makes rollback harder.

### Deploy

A deployment moves a specific release into an environment such as staging or production.

### Restart

A restart stops and starts the same release. It is useful when a process is unhealthy or configuration has changed. A restart is not the same as a deployment and should not rebuild the application.

### Rollback

A rollback restores the previously known-good release when a new deployment fails. A production deployment process is incomplete until rollback is possible.

### CI and CD

- **Continuous Integration (CI):** automatically tests every proposed code change.
- **Continuous Delivery:** keeps validated releases ready for production, usually with a manual approval.
- **Continuous Deployment:** automatically sends every passing change to production.

For Envo, continuous delivery is the better starting point. Automatically deploy staging, but require an intentional production approval.

## 2. Is the current `deploy.sh` good practice?

It is acceptable as an early-stage, single-server deployment tool. It is much better than manually typing unrelated Docker commands because it makes the process repeatable and uses `set -euo pipefail` to stop on many failures.

However, it should not be considered a mature production deployment process yet.

### What it currently does well

- Requires a production environment file.
- Supports Docker Compose v2 and the older `docker-compose` command.
- Runs the backend inside a container.
- Runs database migration and seed commands.
- Uses the Compose restart policy after the container starts.
- Keeps the frontend deployment separate from the backend.

### Current weaknesses

| Current behavior | Why it matters |
| --- | --- |
| Removes the running backend before building | The service is unavailable throughout the build and startup period. |
| Uses `build --no-cache` every time | Deployments are unnecessarily slow and repeatedly download/rebuild unchanged layers. |
| Uses `sleep 5` | Five seconds does not prove that the process, database, or KMS is ready. Fast and slow machines behave differently. |
| Starts the application before migration | New code can briefly run against an old schema. Some schema changes can make startup or requests fail. |
| Builds production code on the server | Production becomes dependent on server disk space, build tools, network access, and local source state. |
| Has no immutable release tag | It is difficult to answer “which exact version is running?” |
| Has no automatic rollback | A failed release requires manual diagnosis while the service may remain unavailable. |
| Runs the seed command on every deployment | Reference data seeding should be intentionally idempotent and normally separated from routine deployments. |
| Uses `|| true` while removing the old container | Some meaningful Docker failures can be hidden. |
| Does not verify the public HTTPS URL | A locally healthy container does not prove DNS, TLS, Nginx, firewall, and routing are healthy. |

### When it is still reasonable to use

The script is reasonable while all of these are true:

- Envo has one backend server.
- Brief deployment downtime is acceptable.
- You are the main operator.
- Deployments are infrequent.
- You retain working database backups.
- You manually verify `/ready` after every deployment.

It should be improved before depending on Envo for critical production credentials.

## 3. How health checking works in Envo

Envo already exposes two different health endpoints. Keeping them separate is good practice.

### Liveness: `/health`

```bash
curl --fail --silent --show-error https://api.example.com/health
```

This confirms that the backend process can answer HTTP requests. It currently reports the service name, version, environment, and whether KMS rather than local encryption is selected.

Use liveness to answer:

> Is the application process running?

Do not make liveness fail just because the database has a temporary outage. Otherwise an orchestrator may repeatedly restart a healthy process while the real dependency remains unavailable.

### Readiness: `/ready`

```bash
curl --fail --silent --show-error https://api.example.com/ready
```

This also pings PostgreSQL with a short timeout. It returns HTTP `503` when the backend should not receive application traffic.

Use readiness to answer:

> Can this instance serve useful requests right now?

The Docker Compose health check already checks `/ready` every ten seconds. View the result with:

```bash
docker compose --env-file .env.production ps
docker inspect --format '{{json .State.Health}}' envy-backend-1
```

The generated container name can vary. Find it first with:

```bash
docker compose --env-file .env.production ps -q backend
```

### Public checks and local checks are different

A server-side check such as `curl http://127.0.0.1:8080/ready` confirms the container path on the machine. It does not verify:

- Public DNS
- HTTPS certificate validity
- Reverse-proxy configuration
- AWS security groups or host firewall
- Routing from the public internet

For actual availability, monitor the public HTTPS endpoints from another network.

### Useful manual diagnostic commands

Run these on the EC2 server:

```bash
# Container state and health
docker compose --env-file .env.production ps

# Recent backend logs
docker compose --env-file .env.production logs --tail=200 backend

# Follow new logs
docker compose --env-file .env.production logs --follow backend

# Resource usage
docker stats

# Host disk usage
df -h

# Public path, including TLS and reverse proxy
curl --fail --silent --show-error https://api.example.com/ready
```

Avoid printing `.env.production`, tokens, request bodies, secret exports, or decrypted environment values into logs or CI output.

## 4. Availability monitoring

Manual checks are useful during diagnosis, but they do not notify you when you are asleep or away from the server.

### Minimum useful monitoring

Set up an external uptime monitor with these checks:

| Target | Frequency | Expected result | Purpose |
| --- | --- | --- | --- |
| Frontend root URL | 1–5 minutes | HTTP 200 | DNS, TLS, CDN, and frontend availability |
| Backend `/health` | 1 minute | HTTP 200 and `status=ok` | Backend process availability |
| Backend `/ready` | 1 minute | HTTP 200 and `database=ok` | Backend plus database readiness |

Alert only after two or three consecutive failures to reduce noise from a single network timeout. Send alerts somewhere you will actually notice, such as email plus a phone/push channel.

Because Envo is already on AWS, CloudWatch Synthetics is a natural managed option. It can run scheduled HTTP or browser canaries and report availability and latency. AWS documents the feature in its [CloudWatch Synthetics guide](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/CloudWatch_Synthetics_Canaries.html). A simpler third-party uptime service is also sufficient at the current scale.

Never place real user or agent credentials inside a public synthetic check. Health checks should not decrypt or export secrets.

### Infrastructure monitoring

Add CloudWatch alarms for the EC2 instance:

- CPU consistently high
- Memory pressure (requires the CloudWatch Agent)
- Disk usage and inode usage
- Instance status-check failure
- Unexpected restart count
- Network anomalies

Disk alerts are particularly important for Docker hosts because old images and container logs can consume the volume.

### Application monitoring

The next observability layer should include:

- Structured JSON logs in production
- Request IDs in every error report
- HTTP request count, latency, and status-code metrics
- Database connection-pool saturation
- KMS latency and error count without logging plaintext
- Agent resolve success, denial, and rate-limit counts
- Authentication and OAuth failure counts
- Deployment version or Git SHA in `/health`

Do not attach secret values, authorization headers, refresh tokens, agent tokens, OAuth codes, or complete request bodies to traces or error-reporting tools.

### A starting service objective

A reasonable first objective is:

- 99.9% monthly availability for `/ready`
- Alert when readiness fails for three consecutive minutes
- Alert when p95 API latency is unusually high for ten minutes
- Review every production incident and record the cause and prevention

99.9% availability permits approximately 44 minutes of downtime in a 30-day month. This is a target for learning and improvement, not a guarantee to advertise before measurement exists.

## 5. Recommended environment model

Do not use one database and one set of credentials for every environment.

| Environment | Purpose | Data and credentials | Deployment |
| --- | --- | --- | --- |
| Local development | Fast coding and testing | Local/test database, local encryption, test OAuth | Developer machine |
| CI | Automated validation | Temporary services and fake credentials | GitHub-hosted runner |
| Staging | Production-like verification | Separate database, OAuth app, KMS key, and test billing configuration | Automatic after `main` passes |
| Production | Real users and secrets | Production-only database, KMS key, OAuth app, and billing configuration | Manual approval of a tested release |

Important rules:

- Never copy production secret values into local development.
- Never let staging use the production database.
- Use separate Google OAuth callback URLs.
- Use separate KMS keys and IAM permissions.
- Keep production configuration out of Git.
- Prefer an EC2 IAM role over static AWS access keys.
- Restrict `.env.production` to the deployment user with `chmod 600` if the file remains on disk.

GitHub Actions environments can separate staging and production secrets, restrict deployment branches, and optionally require approval before production secrets become available. See [GitHub deployment environments](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments) and [GitHub Actions secrets](https://docs.github.com/en/actions/concepts/security/secrets).

## 6. The recommended Envo pipeline

Use a simple feature-branch workflow. Envo does not need a permanent `develop` branch yet.

```text
Feature branch
      │
      ▼
Pull request ──► CI: lint, tests, build, security checks
      │
      ▼ reviewed merge
     main ─────► Build one immutable image
      │         Deploy automatically to staging
      │         Run readiness + smoke checks
      ▼
Production approval
      │
      ▼
Database migration ─► Deploy exact tested image ─► Verify ─► Record release
                                                  │
                                                  └── failure ─► Roll back
```

### Pull-request CI

Create `.github/workflows/ci.yml` and run it for pull requests and pushes to `main`.

It should run independent jobs where possible:

#### Backend

```bash
go test ./...
go vet ./...
go test -race ./...
```

#### Frontend

```bash
npm ci
npm run lint
npm run build
```

#### CLI

```bash
go test ./...
go vet ./...
```

#### Packaging

- Build the backend Docker image.
- Build the frontend production artifact.
- Validate Docker Compose configuration.
- Optionally scan dependencies and images for known vulnerabilities.

CI must not receive production application secrets. Most unit tests should run with fake values or temporary infrastructure.

### Build once, deploy the same artifact

After `main` passes:

1. Build the backend image in CI.
2. Tag it with the commit SHA.
3. Push it to a registry such as GitHub Container Registry or Amazon ECR.
4. Deploy that exact tag to staging.
5. Promote the same tag to production after approval.

Do not rebuild between staging and production. Otherwise production is not running the artifact that staging tested.

### Staging deployment

Staging can deploy automatically after a merge to `main`:

1. Pull the immutable image.
2. Run backward-compatible database migrations.
3. Start or replace the backend.
4. Poll `/ready` rather than sleeping for a fixed duration.
5. Test `/health`, `/ready`, and one harmless public API route.
6. Mark the deployment successful only after the public URL passes.

### Production deployment

Start with `workflow_dispatch` or a version tag and a protected `production` GitHub environment. GitHub supports deployment environments, approvals, branch restrictions, and deployment history; see [Deploying with GitHub Actions](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/control-deployments).

Production should:

1. Accept an already-tested image tag.
2. Prevent two deployments from running simultaneously.
3. Record the currently running image as the rollback target.
4. Confirm that a recent database backup exists.
5. Run compatible migrations.
6. Deploy the image.
7. Poll the public `/ready` endpoint with a timeout.
8. Run smoke tests.
9. Roll back automatically if verification fails.
10. Record who deployed which commit and when.

GitHub Actions concurrency groups prevent overlapping deployments to the same environment. See [GitHub Actions concurrency](https://docs.github.com/en/actions/concepts/workflows-and-actions/concurrency).

### How the pipeline reaches EC2

There are two reasonable stages:

#### Simple first version

- GitHub Actions connects to EC2 using a tightly restricted deployment SSH key.
- The deployment user can only operate the Envo application directory and required Docker commands.
- The workflow tells EC2 to pull and run a specific image tag.

This is understandable and quick, but it creates a long-lived SSH credential that must be protected and rotated.

#### Better AWS-native version

- GitHub Actions uses OpenID Connect to obtain short-lived AWS permissions.
- AWS Systems Manager Run Command triggers deployment on EC2.
- EC2 pulls the image using its IAM role.
- No permanent AWS access key or public deployment SSH port is required.

Move to this version after the basic CI and staging pipeline are stable.

## 7. Making single-server deployments safer

A single EC2 host cannot provide full high availability, but deployment safety can still improve substantially.

### Near-term deployment sequence

Replace the current stop-build-start sequence with:

1. Validate configuration.
2. Pull a prebuilt image tagged with a commit SHA.
3. Confirm the image exists locally.
4. Record the previous image tag.
5. Run migration as a one-off container.
6. Replace the backend container.
7. Poll the container and public `/ready` endpoints for up to a defined timeout.
8. Run smoke checks.
9. Restore the previous image if checks fail.
10. Remove old images later, retaining at least the current and previous release.

This still creates a short replacement window, but avoids taking the application down during compilation.

### Blue/green deployment

When short downtime is no longer acceptable:

- Run `backend-blue` and `backend-green` on separate local ports.
- Deploy the new version to the inactive slot.
- Wait until it passes readiness checks.
- Point Nginx at the healthy slot and reload Nginx.
- Keep the previous slot briefly for fast rollback.

This provides low-downtime deployment on one host, but the EC2 instance is still a single point of failure.

### True high availability

When the business requires protection from an EC2 instance or availability-zone failure, move to multiple backend instances behind a load balancer, for example ECS/Fargate or an Auto Scaling Group.

Before adding replicas, Envo must also move process-local rate limits to shared storage such as Redis. Otherwise each replica enforces a separate limit.

Do not adopt Kubernetes merely because it is common. It adds significant operational complexity and is unnecessary for Envo’s current scale.

## 8. Database migrations and backups

Database changes are the hardest part of rollback because application code can be reverted quickly while changed or deleted data may not be recoverable.

### Migration rules

- Run migrations once per release, not independently on every replica.
- Back up before risky or destructive changes.
- Use explicit, versioned migrations as the project matures instead of relying indefinitely on automatic schema inference.
- Use the **expand-and-contract** pattern:
  1. Add new columns/tables without removing old ones.
  2. Deploy code that supports both schemas.
  3. Backfill data if needed.
  4. Switch all code to the new schema.
  5. Remove old schema in a later release.
- Avoid long table locks during peak usage.
- Test migrations against a staging copy with similar schema and data volume.

### Backup rules

Verify the database provider’s actual backup and point-in-time recovery configuration; never assume it is enabled.

At minimum:

- Automated database backups
- Defined retention period
- Point-in-time recovery when the plan supports it
- A periodic restore test into an isolated database
- Written recovery steps and expected recovery time

For Envo, a database backup contains encrypted secret ciphertext. Recovery also depends on retaining access to the correct AWS KMS key. Protect the KMS key from accidental deletion and understand its deletion waiting period. Losing both usable ciphertext and KMS access makes secret recovery impossible.

## 9. Logs, rollback, and incident response

### Every deployment should record

- Git commit SHA
- Image digest or immutable tag
- Deployment environment
- Start and completion time
- Initiating person or workflow
- Migration result
- Health-check result
- Previous release tag

### Basic rollback runbook

1. Stop further deployments.
2. Confirm whether failure is application, database, KMS, proxy, DNS, or host related.
3. If the schema remains backward compatible, redeploy the previous image.
4. Poll `/ready` and the public URL.
5. Verify authentication and a non-destructive application flow.
6. Do not reverse a database migration blindly.
7. Record the incident timeline and cause.

### Basic outage triage order

```text
Public URL
  ├─ DNS/TLS failure?       -> DNS provider, certificate, reverse proxy
  ├─ HTTP 502/504?          -> backend container, port, proxy timeout
  ├─ /health fails?         -> process/container/host
  ├─ /health works but
  │  /ready fails?          -> PostgreSQL/network/pool
  └─ both work but feature
     fails?                 -> application logs, KMS, OAuth, billing, permissions
```

## 10. Recommended implementation phases

### Phase 1: Do now

- Add CI for backend, frontend, and CLI.
- Configure an external monitor for frontend, `/health`, and `/ready`.
- Add EC2 CPU, memory, disk, and instance alarms.
- Include the real Git SHA in `/health`.
- Change deployment verification from `sleep 5` to readiness polling.
- Stop rebuilding with `--no-cache` by default.
- Document and test manual rollback.
- Verify database backups and perform one restore exercise.

### Phase 2: Safe delivery

- Build immutable backend images in CI.
- Push images to GHCR or ECR.
- Add a separate staging environment.
- Automatically deploy `main` to staging.
- Add a manually approved production workflow.
- Serialize deployments with a concurrency group.
- Run migrations before replacing the application.
- Automatically restore the previous image when smoke checks fail.

### Phase 3: Operational maturity

- Use GitHub OIDC and AWS Systems Manager instead of a permanent SSH key.
- Add structured logs, metrics, dashboards, and error tracking.
- Adopt versioned database migrations.
- Add blue/green deployment if deployment downtime matters.
- Move to multiple backend instances only when availability requirements justify it.
- Move distributed rate limiting to shared infrastructure before scaling horizontally.

## 11. A practical weekly operating routine

Even with automation, operational habits matter.

### Each deployment

- Review CI results.
- Know the release SHA.
- Confirm backup status for schema-changing releases.
- Deploy staging first.
- Verify staging.
- Approve production intentionally.
- Watch logs and availability for several minutes after deployment.

### Weekly

- Review uptime and latency.
- Review failed authentication, KMS, agent, and rate-limit events.
- Check disk usage and old Docker images.
- Check pending system and Docker security updates.
- Confirm backups are still running.

### Monthly or quarterly

- Restore a backup in isolation.
- Test the rollback process.
- Rotate deployment credentials where applicable.
- Review IAM permissions and remove unused access.
- Patch the EC2 operating system with a planned restart window.
- Review alert thresholds and incidents.

## 12. The recommended target for Envo today

The best next architecture is not Kubernetes and not a complicated microservice platform. It is:

- Local development with fast feedback
- Pull-request CI on every change
- One immutable backend image per commit
- A separate staging environment
- Automatic staging deployment from `main`
- Manual production approval
- A safer EC2 deployment using the exact staging-tested image
- External uptime monitoring and CloudWatch infrastructure alarms
- Verified database backups
- Readiness-based deployment checks
- A known previous image for rollback

This gives Envo most of the safety and clarity of a professional delivery system while remaining understandable for a small team.

