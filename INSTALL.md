# Install Envo CLI (one line)

Replace `YOUR_ORG/YOUR_REPO` with your GitHub repo (e.g. `mycompany/Envo`).

---

## macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/YOUR_ORG/YOUR_REPO/main/install.sh | sh
```

Then restart your terminal (or run `source ~/.bashrc` / `source ~/.zshrc`), and run:

```bash
envo whoami
envo login
```

---

## Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/YOUR_ORG/YOUR_REPO/main/install.ps1 | iex
```

If you get an execution policy error, run first:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

Then restart the terminal and run:

```powershell
envo whoami
envo login
```

---

## Optional: install to a specific directory

**macOS/Linux:** `ENVO_INSTALL_DIR=/usr/local/bin curl -fsSL ... | sh`  
**Windows:** `$env:ENVO_INSTALL_DIR = "C:\Tools"; irm ... | iex`

---

## For maintainers: creating a release

1. **Build all binaries** (from repo root or from `cli/`):

   **macOS/Linux:**
   ```bash
   cd cli && chmod +x build-release.sh && ./build-release.sh
   ```

   **Windows:**
   ```powershell
   cd cli; .\build-release.ps1
   ```

2. **Create a new GitHub Release** (tag e.g. `v0.1.0`).

3. **Upload the files from `cli/dist/`** as release assets:
   - `envo-windows-amd64.exe`
   - `envo-darwin-amd64`
   - `envo-darwin-arm64`
   - `envo-linux-amd64`
   - `envo-linux-arm64`

4. **Update INSTALL.md** (and your main README) so the one-liner uses your real repo (e.g. `yourorg/Envo` instead of `YOUR_ORG/YOUR_REPO`).

Asset names must match exactly what the install scripts expect.
