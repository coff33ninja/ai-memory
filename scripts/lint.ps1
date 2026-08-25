param(
    [switch]$Fix
)

$ErrorActionPreference = "Stop"

Write-Host "=== go vet ===" -ForegroundColor Cyan
$result = go vet -tags fts5 ./... 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host $result -ForegroundColor Red
    Write-Host "FAIL: go vet found issues." -ForegroundColor Red
    exit 1
}
Write-Host "PASS: go vet clean" -ForegroundColor Green

Write-Host "=== icon resource ===" -ForegroundColor Cyan
try {
    & "$PSScriptRoot\gen-icons.ps1" 2>$null
    if ($LASTEXITCODE -ne 0) { Write-Host "  (skipped - gen-icons.ps1 failed)" -ForegroundColor Yellow }
} catch {
    Write-Host "  (skipped - gen-icons.ps1 failed)" -ForegroundColor Yellow
}

Write-Host "=== go build ===" -ForegroundColor Cyan
$ver = (Get-Content VERSION -Raw).Trim()
go build -tags fts5 -ldflags="-X main.Version=$ver" ./cmd/ai-memory-server/
if ($LASTEXITCODE -ne 0) {
    exit 1
}
Write-Host "PASS: build ok" -ForegroundColor Green

if ($Fix) {
    Write-Host "=== go test (short) ===" -ForegroundColor Cyan
    go test -tags fts5 -short -count=1 ./... 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "FAIL: tests failed" -ForegroundColor Red
        exit 1
    }
    Write-Host "PASS: tests ok" -ForegroundColor Green
}
