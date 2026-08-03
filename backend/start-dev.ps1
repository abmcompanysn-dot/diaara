param([switch]$Migrate)

Set-Location $PSScriptRoot

$goBin = "C:\Program Files\Go\bin"
if (Test-Path $goBin) { $env:PATH = "$goBin;$env:PATH" }

$envFile = Join-Path $PSScriptRoot ".env.local"
if (-not (Test-Path $envFile)) { Write-Error ".env.local introuvable"; exit 1 }

Get-Content $envFile | ForEach-Object {
  if ($_ -match '^\s*([A-Za-z0-9_]+)\s*=\s*(.*)$') {
    [Environment]::SetEnvironmentVariable($matches[1], $matches[2].Trim(), 'Process')
  }
}

if ($Migrate) {
  Write-Host "== Application des migrations =="
  go run ./cmd/migrate
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

Write-Host "== Serveur : http://localhost:8080 =="
go run ./cmd/server
