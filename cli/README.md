# Envo CLI (`envo`)

## Build

From the repo root:

```bash
cd cli
go mod tidy
go build ./...
```

## Configure backend URL

Set `ENVO_API_URL` (default is `http://localhost:8080`):

```bash
set ENVO_API_URL=http://localhost:8080
```

Or pass `--api` to any command:

```bash
envo --api http://localhost:8080 pull ...
```

## Commands

### `envo login`

1. Runs `GET /api/v1/auth/google/login` to get a Google OAuth URL.
2. You open it in your browser and sign in.
3. The backend callback page returns JSON (contains `access_token`, `refresh_token`).
4. Paste that JSON into the CLI when prompted.

Tokens are cached at your user config directory (example on Windows):
`%APPDATA%\\envo\\tokens.json`.

### `envo pull`

Fetch secrets for an org/project/env and write a `.env` file:

```bash
envo pull --org "<org name or id>" --project "<project name or id>" --env "<env name or id>"
```

Also ensures `.env` is added to `.gitignore`.

### `envo run`

Pull secrets, write `.env`, then run a command with env vars loaded:

```bash
envo run --org "<org>" --project "<project>" --env "<env>" -- npm start
```

