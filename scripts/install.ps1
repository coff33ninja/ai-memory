param(
    [string]$InstallDir = "$env:LOCALAPPDATA\ai-memory",
    [switch]$Update,
    [switch]$UseZig
)

$ErrorActionPreference = "Stop"

$repo = "https://github.com/coff33ninja/ai-memory"

Write-Host "ai-memory installer" -ForegroundColor Cyan
Write-Host ""

# Check Go
$go = Get-Command "go" -ErrorAction SilentlyContinue
if (-not $go) {
    Write-Host "Go is required to build from source." -ForegroundColor Yellow
    Write-Host "Install from: https://go.dev/dl/" -ForegroundColor Yellow
    exit 1
}

# CGO is required (sqlite3, onnxruntime)
# Check for CGO_TRIGGER file in repo root to enable CGO builds
$repoRoot = Split-Path $PSScriptRoot -Parent
$cgEnabled = Test-Path (Join-Path $repoRoot "CGO_TRIGGER")
if (-not $cgEnabled) {
    Write-Host "WARNING: CGO_TRIGGER file not found in repo root." -ForegroundColor Yellow
    Write-Host "ai-memory requires CGO for sqlite3 and onnxruntime." -ForegroundColor Yellow
    Write-Host "Create CGO_TRIGGER file to enable CGO builds." -ForegroundColor Yellow
    Write-Host "Continuing anyway..." -ForegroundColor Yellow
}

# Check / install Zig
$zig = Get-Command "zig" -ErrorAction SilentlyContinue
if ($UseZig -and -not $zig) {
    Write-Host "Zig not found. Installing Zig..." -ForegroundColor Yellow
    $zigUrl = "https://ziglang.org/download/0.16.0/zig-x86_64-windows-0.16.0.zip"
    $zigZip = "$env:TEMP\zig.zip"
    $zigDir = "$env:LOCALAPPDATA\zig"
    try {
        Invoke-WebRequest -Uri $zigUrl -OutFile $zigZip -ErrorAction Stop
        Expand-Archive -Path $zigZip -DestinationPath $zigDir -Force
        $zigPath = "$zigDir\zig-x86_64-windows-0.16.0\zig.exe"
        if (Test-Path -LiteralPath $zigPath) {
            $env:Path += ";$(Split-Path $zigPath)"
            [Environment]::SetEnvironmentVariable("Path", [Environment]::GetEnvironmentVariable("Path","User") + ";$(Split-Path $zigPath)", "User")
            $zig = $zigPath
            Write-Host "Zig installed to $zigPath" -ForegroundColor Green
        }
    } catch {
        Write-Host "Zig install failed: $_" -ForegroundColor Red
        Write-Host "Continuing without Zig..." -ForegroundColor Yellow
        $UseZig = $false
    }
    Remove-Item $zigZip -ErrorAction SilentlyContinue
}

# Create install dir
if (-not (Test-Path -LiteralPath $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$exePath = "$InstallDir\ai-memory-server.exe"

# Check if already installed
if ((Test-Path -LiteralPath $exePath) -and -not $Update) {
    Write-Host "Already installed at: $exePath" -ForegroundColor Green
    Write-Host "Run with -Update to rebuild." -ForegroundColor Yellow
    exit 0
}

# Clone or pull
$srcDir = "$env:TEMP\ai-memory"
if (Test-Path -LiteralPath $srcDir) {
    Remove-Item -Recurse -Force $srcDir -ErrorAction SilentlyContinue
}

Write-Host "Cloning repository..." -ForegroundColor Gray
git clone --depth 1 $repo $srcDir 2>$null
if (-not $?) {
    Write-Host "Failed to clone repository. Check your network and git installation." -ForegroundColor Red
    exit 1
}

# Build
Write-Host "Building ai-memory-server.exe..." -ForegroundColor Gray
Push-Location $srcDir
try {
    $env:CGO_ENABLED = "1"
    if ($UseZig -and $zig) {
        $env:CC = "zig cc"
        Write-Host "Using Zig cc as C compiler" -ForegroundColor Cyan
    }
    go build -o $exePath -ldflags="-s -w" .\cmd\ai-memory-server\
    if (-not $?) {
        Write-Host "Build failed." -ForegroundColor Red
        exit 1
    }
} finally {
    Pop-Location
}

# Clean up source
Remove-Item -Recurse -Force $srcDir -ErrorAction SilentlyContinue

# Create default config
$configDir = "$env:USERPROFILE\.config\ai-memory"
$configPath = "$configDir\config.json"
if (-not (Test-Path -LiteralPath $configPath)) {
    if (-not (Test-Path -LiteralPath $configDir)) {
        New-Item -ItemType Directory -Path $configDir -Force | Out-Null
    }
    @{
        log_level = "info"
    } | ConvertTo-Json | Set-Content -Path $configPath
}

Write-Host ""
Write-Host "Installed: $exePath" -ForegroundColor Green
Write-Host "Config:    $configPath" -ForegroundColor Green
Write-Host ""
Write-Host "Add to opencode.json:" -ForegroundColor Cyan
Write-Host "  `"command`": `"$exePath`"" -ForegroundColor Gray
