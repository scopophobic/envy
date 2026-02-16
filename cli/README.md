# Envo CLI

## Install with one command (no build)

See **[INSTALL.md](../INSTALL.md)** in the repo root for one-line install (curl \| sh on Mac/Linux, irm \| iex on Windows). Requires a GitHub Release with built binaries.

---

## Easiest: no build (from repo root)

From the **Envo repo root** (where you see `envo.cmd` and `cli/`), run:

**Windows:** `.\envo.cmd login` · `.\envo.cmd whoami` · `.\envo.cmd pull --org "MyOrg" --project "api" --env "production"`

**macOS/Linux:** `chmod +x envo` once, then `./envo login` · `./envo whoami` · `./envo pull ...`

You need **Go** installed. First run may take a few seconds.

---

## Optional: build once, use from anywhere

### 1. Build the binary

From the **repo root** (or from `cli/`):

**Windows (PowerShell):**
```powershell
cd cli
go build -o envo.exe ./cmd/envo
```

**macOS / Linux:**
```bash
cd cli
go build -o envo ./cmd/envo
```

This creates `envo.exe` (Windows) or `envo` (macOS/Linux) in the `cli` folder.

### 2. Run the CLI

**Option A — Run from the same folder:**
```powershell
# Windows (from cli/)
.\envo.exe whoami
.\envo.exe login
.\envo.exe pull --org "MyOrg" --project "api" --env "production"
```

```bash
# macOS/Linux (from cli/)
./envo whoami
./envo login
./envo pull --org "MyOrg" --project "api" --env "production"
```

**Option B — Install so `envo` works from anywhere:**

Copy the binary to a folder that’s in your PATH.

**Windows:** e.g. copy to a folder you use for tools, then add that folder to your [PATH](https://www.computerhope.com/issues/ch000549.htm), or use:
```powershell
# Example: copy to a local bin folder and add to PATH for this session
mkdir -Force $HOME\bin
Copy-Item envo.exe $HOME\bin\
$env:Path += ";$HOME\bin"
envo whoami
```

**macOS / Linux:**
```bash
sudo mv envo /usr/local/bin/
# or
mkdir -p ~/bin && mv envo ~/bin && export PATH="$HOME/bin:$PATH"
envo whoami
```

---

## Commands

| Command | Description |
|--------|-------------|
| `envo login` | Sign in with Google (opens browser). Requires backend running. |
| `envo logout` | Clear saved tokens. |
| `envo whoami` | Show current user (name, email, tier). |
| `envo pull --org <org> --project <project> --env <env>` | Download secrets to `.env` in current directory. |
| `envo run --org <org> --project <project> --env <env> -- <command>` | Pull secrets, then run a command with env vars loaded. |

**Examples:**
```bash
envo login
envo whoami
envo pull --org "MyOrg" --project "api" --env "production"
envo pull --org "MyOrg" --project "api" --env "production" --dir ./my-app
envo run --org "MyOrg" --project "api" --env "production" -- npm start
```

---

## Configure API URL

If your backend is not at `http://localhost:8080`:

- **Environment variable:** set `ENVO_API_URL`
  - Windows: `$env:ENVO_API_URL = "http://127.0.0.1:8080"`
  - macOS/Linux: `export ENVO_API_URL=http://127.0.0.1:8080`
- **Per command:** `envo --api http://127.0.0.1:8080 pull ...`

Use the same host as in your backend’s `GOOGLE_REDIRECT_URL` (e.g. both `localhost` or both `127.0.0.1`).

---

## Tokens

After `envo login`, tokens are stored at:

- **Windows:** `%APPDATA%\envo\tokens.json`
- **macOS:** `~/Library/Application Support/envo/tokens.json`
- **Linux:** `~/.config/envo/tokens.json`

---

## Publishing the CLI (for others to install)

### 1. Release binaries (GitHub Releases)

Build for multiple OS/arch and upload to GitHub Releases so users can download `envo` or `envo.exe`.

**Example build script (run from `cli/`):**

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o envo-linux-amd64 ./cmd/envo
GOOS=linux GOARCH=arm64 go build -o envo-linux-arm64 ./cmd/envo

# macOS (Intel + Apple Silicon)
GOOS=darwin GOARCH=amd64 go build -o envo-darwin-amd64 ./cmd/envo
GOOS=darwin GOARCH=arm64 go build -o envo-darwin-arm64 ./cmd/envo

# Windows
GOOS=windows GOARCH=amd64 go build -o envo-windows-amd64.exe ./cmd/envo
```

Then create a GitHub Release and attach these files. Users download the right one and put it in their PATH.

### 2. Package managers (optional, later)

- **Windows:** publish to [Scoop](https://scoop.sh/) or [Chocolatey](https://chocolatey.org/) (bucket/manifest points at your GitHub Release).
- **macOS:** Homebrew formula that downloads the release binary.
- **Linux:** .deb/.rpm or a snap that uses the release binary.

### 3. Go Install (for Go users)

If the repo is public on GitHub:

```bash
go install github.com/yourusername/envo/cli/cmd/envo@latest
```

This puts `envo` in `$GOPATH/bin` or `$HOME/go/bin`. Users need Go installed.

---

## Summary- **“envo is unknown”** means the shell can’t find an `envo` executable. Build with `go build -o envo.exe ./cmd/envo` (or `-o envo` on macOS/Linux), then run `.\envo.exe` / `./envo` from that folder or add that folder to PATH.
- **Publishing:** build binaries for each OS/arch, upload to GitHub Releases, then optionally add package managers or `go install` so users can install the CLI easily.
