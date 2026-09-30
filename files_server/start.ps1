$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptDir

$env:MEDIA_UPLOAD_TOKEN = "your-secret-token"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Files Server Startup" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

if (-not (Test-Path "config.conf")) {
    Write-Host "ERROR: config.conf not found" -ForegroundColor Red
    exit 1
}

if (-not (Test-Path "go.mod")) {
    Write-Host "ERROR: Please run this script from files_server directory" -ForegroundColor Red
    exit 1
}

Write-Host "Building..." -ForegroundColor Yellow
go build -o files-server.exe .

if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed" -ForegroundColor Red
    exit 1
}

Write-Host "Starting files server..." -ForegroundColor Green
.\files-server.exe
