#requires -version 5.1
<#
.SYNOPSIS
  Проверяет зависимости, собирает tgit.exe и (по желанию) добавляет его в PATH.

.DESCRIPTION
  Windows-аналог install.sh: проверяет наличие git и нужной версии Go (по go.mod),
  собирает бинарник и кладёт его в каталог установки. Добавление в PATH — по явному
  флагу -AddToPath, чтобы не менять окружение пользователя без спроса.

.PARAMETER Prefix
  Каталог установки. По умолчанию — $env:LOCALAPPDATA\Programs\tgit (не требует прав администратора).

.PARAMETER AddToPath
  Добавить каталог установки в PATH текущего пользователя (реестр, HKCU) и в PATH текущей сессии.
  Без этого флага скрипт только покажет, что нужно сделать.

.EXAMPLE
  .\install.ps1

.EXAMPLE
  .\install.ps1 -AddToPath

.EXAMPLE
  .\install.ps1 -Prefix 'C:\tools\tgit' -AddToPath

.NOTES
  Если запуск .ps1-файлов заблокирован политикой выполнения, запустите разово:
    powershell -ExecutionPolicy Bypass -File install.ps1
#>

[CmdletBinding()]
param(
    [string]$Prefix = "$env:LOCALAPPDATA\Programs\tgit",
    [switch]$AddToPath
)

$ErrorActionPreference = "Stop"
# В PowerShell 7.3+ ненулевой код возврата внешней команды (git/go) иначе сам
# бросает завершающую ошибку раньше, чем скрипт успевает проверить $LASTEXITCODE.
$PSNativeCommandUseErrorActionPreference = $false

function Write-Step  { param($m) Write-Host "==> $m" -ForegroundColor Cyan }
function Write-Ok    { param($m) Write-Host "OK  $m" -ForegroundColor Green }
function Write-WarnMsg { param($m) Write-Host "!   $m" -ForegroundColor Yellow }
function Write-ErrMsg  { param($m) Write-Host "x   $m" -ForegroundColor Red }
function Fail        { param($m) Write-ErrMsg $m; exit 1 }

$scriptDir = $PSScriptRoot
if (-not $scriptDir) { $scriptDir = (Get-Location).Path }

# ---------- git ----------

Write-Step "Проверяю git..."
$gitCmd = Get-Command git -ErrorAction SilentlyContinue
if (-not $gitCmd) {
    Write-ErrMsg "git не найден в PATH."
    Write-Host "  Установить: winget install --id Git.Git -e"
    Write-Host "  или:        choco install git"
    Write-Host "  или скачать с официальной страницы git-scm.com"
    exit 1
}
Write-Ok "git найден: $(git --version)"

# ---------- go ----------

Write-Step "Проверяю Go..."
$goCmd = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCmd) {
    Write-ErrMsg "Go не найден в PATH."
    Write-Host "  Установить: winget install --id GoLang.Go -e"
    Write-Host "  или:        choco install golang"
    Write-Host "  или скачать с официальной страницы go.dev/dl"
    exit 1
}

$goModPath = Join-Path $scriptDir "go.mod"
if (-not (Test-Path $goModPath)) {
    Fail "Не найден go.mod рядом со скриптом ($scriptDir) — запустите install.ps1 из корня репозитория tgit."
}
$goModLine = Select-String -Path $goModPath -Pattern '^go (\d+\.\d+(\.\d+)?)' | Select-Object -First 1
if (-not $goModLine) {
    Fail "Не удалось прочитать требуемую версию Go из go.mod."
}
$requiredGo = $goModLine.Matches[0].Groups[1].Value

$goVersionRaw = (& go env GOVERSION)
if ($goVersionRaw -match 'go(\d+\.\d+(\.\d+)?)') {
    $currentGo = $Matches[1]
} else {
    Fail "Не удалось распознать версию Go из '$goVersionRaw'."
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
    Fail "Нужен Go >= $requiredGo, установлен $goVersionRaw."
}
Write-Ok "Go найден: $goVersionRaw (нужен >= $requiredGo)"

# ---------- сборка ----------

Write-Step "Собираю tgit.exe..."

$version = "dev"
$described = & git -C $scriptDir describe --tags --always --dirty 2>$null
if ($LASTEXITCODE -eq 0 -and $described) { $version = $described.Trim() }

New-Item -ItemType Directory -Force -Path $Prefix | Out-Null
$outPath = Join-Path $Prefix "tgit.exe"

Push-Location $scriptDir
try {
    & go build -ldflags "-X main.version=$version" -o $outPath .
    if ($LASTEXITCODE -ne 0) { Fail "Сборка не удалась." }
} finally {
    Pop-Location
}
Write-Ok "Сборка готова, версия: $version"
Write-Ok "tgit установлен: $outPath"

# ---------- проверка ----------

Write-Step "Проверка запуска..."
try {
    $verOut = & $outPath --version
    Write-Ok "Проверка запуска: $verOut"
} catch {
    Write-WarnMsg "Не удалось запустить $outPath --version — проверьте вручную."
}

# ---------- PATH ----------

Write-Step "Проверяю PATH..."
$normalizedPrefix = $Prefix.TrimEnd('\')
$pathEntries = $env:Path -split ';' | Where-Object { $_ -ne '' }
$alreadyInPath = $pathEntries | Where-Object { $_.TrimEnd('\') -ieq $normalizedPrefix }

if ($alreadyInPath) {
    Write-Ok "$Prefix уже в PATH — команда 'tgit' готова к использованию."
} elseif ($AddToPath) {
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $newUserPath = if ([string]::IsNullOrEmpty($userPath)) { $Prefix } else { "$userPath;$Prefix" }
    [Environment]::SetEnvironmentVariable('Path', $newUserPath, 'User')
    $env:Path = "$env:Path;$Prefix"
    Write-Ok "$Prefix добавлен в PATH пользователя. В уже открытых терминалах команда появится после перезапуска."
} else {
    Write-WarnMsg "$Prefix отсутствует в PATH."
    Write-Host ""
    Write-Host "  Добавить сейчас автоматически:"
    Write-Host "    .\install.ps1 -AddToPath"
    Write-Host ""
    Write-Host "  Либо вручную, для текущего пользователя:"
    Write-Host "    [Environment]::SetEnvironmentVariable('Path', `$env:Path + ';$Prefix', 'User')"
    Write-Host ""
}

Write-Host ""
Write-Ok "Готово. Запуск: tgit из каталога любого git-репозитория."
