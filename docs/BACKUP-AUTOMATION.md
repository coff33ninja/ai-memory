# Backup Automation

Standalone backup scheduler that runs independently from the MCP server. The `ai-memory-server.exe` sits untouched until the scheduler triggers it — no MCP connection needed.

## How It Works

```
┌─────────────────────┐     ┌──────────────────────┐     ┌─────────────────┐
│  Windows Task       │────▶│  ai-memory-backup.exe │────▶│  gh repo push   │
│  Scheduler (9a/7p)  │     │  (standalone binary)  │     │  (ai-memory-    │
│                     │     │                       │     │   backup repo)  │
└─────────────────────┘     └──────────────────────┘     └─────────────────┘
```

1. Windows Task Scheduler fires at 9:00 AM and 7:00 PM
2. `ai-memory-backup.exe` starts, creates a zip of `%USERPROFILE%\.ai-memory\`
3. Pushes the zip to the private `ai-memory-backup` GitHub repo via `gh` CLI
4. Prunes old backups (keeps last 3)
5. Exits — no lingering process

## Schedule

| Time | Trigger | Why |
|------|---------|-----|
| 9:00 AM | Work start | Capture overnight changes |
| 7:00 PM | Home arrival | Capture workday changes |

## Setup

### Prerequisites

- `ai-memory-server.exe` built and on PATH (or full path configured)
- `gh` CLI authenticated (`gh auth status`)
- Windows Task Scheduler (comes with Windows)

### Option A: PowerShell Script (Recommended)

Save `scripts/backup-schedule.ps1` and register with Task Scheduler:

```powershell
# Register the 9 AM task
schtasks /create /tn "ai-memory-backup-morning" /tr "powershell -ExecutionPolicy Bypass -File C:\path\to\ai-memory\scripts\backup-schedule.ps1" /sc daily /st 09:00 /rl HIGHEST

# Register the 7 PM task
schtasks /create /tn "ai-memory-backup-evening" /tr "powershell -ExecutionPolicy Bypass -File C:\path\to\ai-memory\scripts\backup-schedule.ps1" /sc daily /st 19:00 /rl HIGHEST
```

### Option B: Go Binary (Coming Home Build)

Build `cmd/ai-memory-backup/main.go` as a standalone exe:

```powershell
# Build
go build -o ai-memory-backup.exe ./cmd/ai-memory-backup

# Register tasks
schtasks /create /tn "ai-memory-backup-morning" /tr "C:\path\to\ai-memory-backup.exe" /sc daily /st 09:00 /rl HIGHEST
schtasks /create /tn "ai-memory-backup-evening" /tr "C:\path\to\ai-memory-backup.exe" /sc daily /st 19:00 /rl HIGHEST
```

### Verify Tasks

```powershell
# List registered tasks
schtasks /query /tn "ai-memory-backup-*" /fo table

# Test run manually
schtasks /run /tn "ai-memory-backup-morning"

# Check task history
Get-WinEvent -LogName "Microsoft-Windows-TaskScheduler/Operational" -MaxEvents 10 | Where-Object { $_.Message -like "*ai-memory*" }
```

## What Gets Backed Up

```
%USERPROFILE%\.ai-memory\
├── {persona}/memory.db       # Per-persona memories + embeddings
├── shared/memory.db          # Shared memories
├── personas.json             # Persona registry
└── backup-config.json        # Backup configuration
```

**Excluded**: `skills/` directory (cloned from remote, not user data), `lib/` (ONNX models, re-downloadable).

## Restore

From the MCP server or manually:

```powershell
# List backups
schtasks /run /tn "ai-memory-backup-morning"  # triggers fresh backup
# Then in MCP: backup_status()

# Restore latest
# In MCP: restore()

# Manual restore from GitHub
gh repo clone ai-memory-backup C:\temp\ai-memory-restore
# Extract the zip to %USERPROFILE%\.ai-memory\
```

## GitHub Backup Repo

The backup target is a private GitHub repo named `ai-memory-backup`. Created automatically on first backup if it doesn't exist.

```bash
# Check repo
gh repo view ai-memory-backup

# List backups
gh api repos/{owner}/ai-memory-backup/contents --jq '.[].name'

# Download a specific backup
gh api repos/{owner}/ai-memory-backup/contents/ai-memory-backup-YYYYMMDD-HHMMSS.zip -q '.download_url' | xargs curl -O
```

## Logs

Backup output goes to Windows Event Log. Check with:

```powershell
Get-WinEvent -LogName "Application" -MaxEvents 20 | Where-Object { $_.Message -like "*ai-memory*" }
```

Or redirect to a file by modifying the task:

```powershell
schtasks /change /tn "ai-memory-backup-morning" /tr "powershell -ExecutionPolicy Bypass -File backup-schedule.ps1 >> C:\logs\ai-memory-backup.log 2>&1"
```

## Troubleshooting

| Problem | Fix |
|---------|-----|
| Task doesn't run | Check `schtasks /query` — ensure status is "Ready", not "Disabled" |
| `gh` not found | Add `gh` to PATH or use full path in task action |
| Backup fails silently | Run manually: `powershell -File backup-schedule.ps1` and check output |
| Wrong time | Verify timezone in Task Scheduler — uses system timezone |
| Permission error | Use `/rl HIGHEST` flag when creating tasks |

## Architecture

This is intentionally decoupled from the MCP server:

- **MCP server** = runtime, handles AI tool calls, manages memory
- **Backup scheduler** = standalone, fires at fixed times, pushes to GitHub
- **No shared state** = backup exe reads the same files but doesn't need the MCP server running
- **No dependency** = if the MCP server is crashed/down, backups still happen

The backup exe can be built from `cmd/ai-memory-backup/main.go` or use the PowerShell script approach. Both produce the same result: a timestamped zip pushed to GitHub.
