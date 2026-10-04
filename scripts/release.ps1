<#
.SYNOPSIS
    Automated Release and Tagging Script for Apirout.
.DESCRIPTION
    Validates git status, runs test suite and web build, synchronizes version manifests,
    creates an annotated SemVer git tag, and optionally pushes to origin.
.PARAMETER Version
    The release version, e.g. "v1.0.0" or "1.0.0".
.PARAMETER Push
    Switch to automatically push the new tag and commits to origin.
#>
param(
    [Parameter(Mandatory=$true, Position=0)]
    [string]$Version,

    [switch]$Push
)

$ErrorActionPreference = "Stop"

# 1. Normalize version strings
$CleanVersion = $Version.Trim()
$Tag = if ($CleanVersion.StartsWith("v")) { $CleanVersion } else { "v$CleanVersion" }
$SemVer = if ($CleanVersion.StartsWith("v")) { $CleanVersion.Substring(1) } else { $CleanVersion }

if ($SemVer -notmatch '^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$') {
    Write-Error "Invalid SemVer version format '$Version'. Expected format: '1.0.0' or 'v1.0.0' (or with pre-release suffix)."
    exit 1
}

Write-Host "==> Starting release preparation for $Tag ($SemVer)..." -ForegroundColor Cyan

# 2. Check if tag already exists
$existingTag = git tag -l $Tag
if ($existingTag) {
    Write-Error "Tag '$Tag' already exists locally or remotely! Please specify a new version."
    exit 1
}

# 3. Pre-flight check: working tree must be clean
$status = git status --porcelain
if ($status) {
    Write-Error "Working directory has uncommitted changes. Please commit or stash changes before releasing:`n$status"
    exit 1
}

# 4. Pre-flight check: run test suite
Write-Host "==> Running full test suite..." -ForegroundColor Yellow
& go test -count=1 ./...
if ($LASTEXITCODE -ne 0) {
    Write-Error "Go test suite failed. Aborting release."
    exit 1
}

# 5. Pre-flight check: run web console build
Write-Host "==> Verifying web console build..." -ForegroundColor Yellow
Push-Location web
try {
    & npm run build
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Web console build failed. Aborting release."
        exit 1
    }
} finally {
    Pop-Location
}

# 6. Synchronize package.json version if needed
$pkgJsonPath = "web/package.json"
if (Test-Path $pkgJsonPath) {
    $pkgJson = Get-Content $pkgJsonPath -Raw | ConvertFrom-Json
    if ($pkgJson.version -ne $SemVer) {
        Write-Host "==> Synchronizing $pkgJsonPath to version $SemVer..." -ForegroundColor Green
        $pkgJson.version = $SemVer
        $newJson = $pkgJson | ConvertTo-Json -Depth 10
        [System.IO.File]::WriteAllText((Resolve-Path $pkgJsonPath).Path, $newJson + "`n", [System.Text.UTF8Encoding]::new($false))
        
        git add $pkgJsonPath
        git commit -m "chore(release): bump package manifest to $Tag"
    }
}

# 7. Create annotated Git tag
Write-Host "==> Creating annotated tag $Tag..." -ForegroundColor Green
$tagMessage = "Release $Tag - Apirout Open Source API Engine & Gateway"
& git tag -a $Tag -m $tagMessage
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to create tag $Tag."
    exit 1
}

Write-Host "==> Tag $Tag created successfully!" -ForegroundColor Green

# 8. Push to remote if requested
if ($Push) {
    Write-Host "==> Pushing commits and tag $Tag to origin..." -ForegroundColor Cyan
    & git push origin main --tags
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to push to origin."
        exit 1
    }
    Write-Host "==> Successfully released and pushed $Tag to origin!" -ForegroundColor Green
} else {
    Write-Host ""
    Write-Host "Release $Tag prepared locally. To push to GitHub, run:" -ForegroundColor Yellow
    Write-Host "  git push origin main --tags" -ForegroundColor White
}
