#requires -version 5.1
<#
.SYNOPSIS
  Checks dependencies, builds tgit.exe, and (optionally) adds it to PATH.

.DESCRIPTION
  Windows counterpart of install.sh: checks for git and the required Go
  version (from go.mod), builds the binary, and places it in the install
  directory. Adding to PATH requires the explicit -AddToPath flag, so the
  user's environment isn't changed without asking.

.PARAMETER Prefix
  Install directory. Defaults to $env:LOCALAPPDATA\Programs\tgit (no admin rights needed).

.PARAMETER AddToPath
  Add the install directory to the current user's PATH (registry, HKCU) and
  to the current session's PATH. Without this flag the script only shows
  what needs to be done.

.EXAMPLE
  .\install.ps1

.EXAMPLE
  .\install.ps1 -AddToPath

.EXAMPLE
  .\install.ps1 -Prefix 'C:\tools\tgit' -AddToPath

.NOTES
  If running .ps1 files is blocked by execution policy, run this once:
    powershell -ExecutionPolicy Bypass -File install.ps1
#>

[CmdletBinding()]
param(
    [string]$Prefix = "$env:LOCALAPPDATA\Programs\tgit",
    [switch]$AddToPath
)

$ErrorActionPreference = "Stop"
# On PowerShell 7.3+, a nonzero exit code from an external command (git/go)
# would otherwise throw a terminating error before the script gets to check
# $LASTEXITCODE itself.
$PSNativeCommandUseErrorActionPreference = $false

function Write-Step  { param($m) Write-Host "==> $m" -ForegroundColor Cyan }
function Write-Ok    { param($m) Write-Host "OK  $m" -ForegroundColor Green }
function Write-WarnMsg { param($m) Write-Host "!   $m" -ForegroundColor Yellow }
function Write-ErrMsg  { param($m) Write-Host "x   $m" -ForegroundColor Red }
function Fail        { param($m) Write-ErrMsg $m; exit 1 }

$scriptDir = $PSScriptRoot
if (-not $scriptDir) { $scriptDir = (Get-Location).Path }

# ---------- git ----------

Write-Step "Checking git..."
$gitCmd = Get-Command git -ErrorAction SilentlyContinue
if (-not $gitCmd) {
    Write-ErrMsg "git not found in PATH."
    Write-Host "  Install: winget install --id Git.Git -e"
    Write-Host "  or:      choco install git"
    Write-Host "  or download it from the official git-scm.com page"
    exit 1
}
Write-Ok "git found: $(git --version)"

# ---------- go ----------

Write-Step "Checking Go..."
$goCmd = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCmd) {
    Write-ErrMsg "Go not found in PATH."
    Write-Host "  Install: winget install --id GoLang.Go -e"
    Write-Host "  or:      choco install golang"
    Write-Host "  or download it from the official go.dev/dl page"
    exit 1
}

$goModPath = Join-Path $scriptDir "go.mod"
if (-not (Test-Path $goModPath)) {
    Fail "go.mod not found next to the script ($scriptDir) — run install.ps1 from the root of the tgit repository."
}
$goModLine = Select-String -Path $goModPath -Pattern '^go (\d+\.\d+(\.\d+)?)' | Select-Object -First 1
if (-not $goModLine) {
    Fail "Could not read the required Go version from go.mod."
}
$requiredGo = $goModLine.Matches[0].Groups[1].Value

$goVersionRaw = (& go env GOVERSION)
if ($goVersionRaw -match 'go(\d+\.\d+(\.\d+)?)') {
    $currentGo = $Matches[1]
} else {
    Fail "Could not parse the Go version from '$goVersionRaw'."
}

function Test-VersionGe($a, $b) {
    $pa = @(($a -split '\.') + @('0', '0', '0'))
    $pb = @(($b -split '\.') + @('0', '0', '0'))
    for ($i = 0; $i -lt 3; $i++) {
        $ai = [int]$pa[$i]; $bi = [int]$pb[$i]
        if ($ai -ne $bi) { return $ai -gt $bi }
    }
    return $true
}

if (-not (Test-VersionGe $currentGo $requiredGo)) {
    Fail "Go >= $requiredGo is required, found $goVersionRaw."
}
Write-Ok "Go found: $goVersionRaw (need >= $requiredGo)"

# ---------- build ----------

Write-Step "Building tgit.exe..."

$version = "dev"
$described = & git -C $scriptDir describe --tags --always --dirty 2>$null
if ($LASTEXITCODE -eq 0 -and $described) { $version = $described.Trim() }

New-Item -ItemType Directory -Force -Path $Prefix | Out-Null
$outPath = Join-Path $Prefix "tgit.exe"

Push-Location $scriptDir
try {
    & go build -ldflags "-X main.version=$version" -o $outPath .
    if ($LASTEXITCODE -ne 0) { Fail "Build failed." }
} finally {
    Pop-Location
}
Write-Ok "Build ready, version: $version"
Write-Ok "tgit installed: $outPath"

# ---------- verification ----------

Write-Step "Checking that it runs..."
try {
    $verOut = & $outPath --version
    Write-Ok "Launch check: $verOut"
} catch {
    Write-WarnMsg "Could not run $outPath --version — please check manually."
}

# ---------- PATH ----------

Write-Step "Checking PATH..."
$normalizedPrefix = $Prefix.TrimEnd('\')
$pathEntries = $env:Path -split ';' | Where-Object { $_ -ne '' }
$alreadyInPath = $pathEntries | Where-Object { $_.TrimEnd('\') -ieq $normalizedPrefix }

if ($alreadyInPath) {
    Write-Ok "$Prefix is already on PATH — the 'tgit' command is ready to use."
} elseif ($AddToPath) {
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $newUserPath = if ([string]::IsNullOrEmpty($userPath)) { $Prefix } else { "$userPath;$Prefix" }
    [Environment]::SetEnvironmentVariable('Path', $newUserPath, 'User')
    $env:Path = "$env:Path;$Prefix"
    Write-Ok "$Prefix added to the user's PATH. Already-open terminals will pick it up after a restart."
} else {
    Write-WarnMsg "$Prefix is not on PATH."
    Write-Host ""
    Write-Host "  Add it automatically now:"
    Write-Host "    .\install.ps1 -AddToPath"
    Write-Host ""
    Write-Host "  Or manually, for the current user:"
    Write-Host "    [Environment]::SetEnvironmentVariable('Path', `$env:Path + ';$Prefix', 'User')"
    Write-Host ""
}

Write-Host ""
Write-Ok "Done. Run: tgit from inside any git repository."
