# Install Envo CLI

One command to install. Works on macOS, Linux, and Windows.

---

## macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/scopophobic/envy/main/install.sh | sh
```

## Windows — PowerShell

```powershell
irm https://raw.githubusercontent.com/scopophobic/envy/main/install.ps1 | iex
```

If you get an execution policy error, run this first:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

## Windows — Git Bash

The same `install.sh` works in Git Bash. It detects MSYS/MINGW and downloads the Windows binary:

```bash
curl -fsSL https://raw.githubusercontent.com/scopophobic/envy/main/install.sh | sh
```

It installs to `~/bin/envo.exe`. If `~/bin` isn't in your PATH, add this to `~/.bashrc`:

```bash
export PATH="$HOME/bin:$PATH"
```

## Go developers

```bash
go install github.com/envo/cli/cmd/envo@latest
```

---

## Quick start

```bash
# 1. Sign in (opens browser)
envo login

# 2. Pull secrets to a .env file
envo pull --org my-team --project api --env development

# 3. Or run a command with secrets injected (never writes to disk)
envo run --org my-team --project api --env development -- npm start

# 4. Check who you're logged in as
envo whoami
```

---

## For maintainers: creating a release

Releases are automated via GitHub Actions. To publish a new CLI version:

```bash
git tag v0.1.0
git push origin v0.1.0
```

GoReleaser builds binaries for all platforms (macOS amd64/arm64, Linux amd64/arm64, Windows amd64) and creates a GitHub Release with all assets automatically.

The install scripts always pull the latest release.
