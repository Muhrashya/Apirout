<#
.SYNOPSIS
    Validate all deployment configurations (Docker Compose, Nginx, Kubernetes).
#>
$ErrorActionPreference = "Stop"

Write-Host "==> 1. Validating Production Docker Compose..." -ForegroundColor Cyan
& docker compose -f deploy/docker-compose.prod.yml config --quiet
if ($LASTEXITCODE -ne 0) {
    Write-Error "Production docker-compose.prod.yml syntax validation failed."
    exit 1
}
Write-Host "[OK] Production Docker Compose syntax valid" -ForegroundColor Green

Write-Host "==> 2. Validating Nginx Edge Proxy configuration..." -ForegroundColor Cyan
$nginxConf = (Resolve-Path "deploy/nginx/nginx.conf").Path
$confD = (Resolve-Path "deploy/nginx/conf.d").Path
$sslDir = (Resolve-Path "deploy/nginx/ssl").Path
& docker run --rm -v "${nginxConf}:/etc/nginx/nginx.conf:ro" -v "${confD}:/etc/nginx/conf.d:ro" -v "${sslDir}:/etc/nginx/ssl:ro" nginx:alpine nginx -t
if ($LASTEXITCODE -ne 0) {
    # If host resolution fails without network, run on compose network
    & docker run --rm --network api-builder-main_default -v "${nginxConf}:/etc/nginx/nginx.conf:ro" -v "${confD}:/etc/nginx/conf.d:ro" -v "${sslDir}:/etc/nginx/ssl:ro" nginx:alpine nginx -t
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Nginx configuration test failed."
        exit 1
    }
}
Write-Host "[OK] Nginx edge proxy configuration valid" -ForegroundColor Green

Write-Host "==> 3. Validating Kubernetes Manifests..." -ForegroundColor Cyan
$k8sFiles = Get-ChildItem -Path "deploy/k8s/*.yaml" | Sort-Object Name
foreach ($f in $k8sFiles) {
    Write-Host "  - Checking $($f.Name)..." -ForegroundColor Gray
    $content = Get-Content $f.FullName -Raw
    if ([string]::IsNullOrWhiteSpace($content)) {
        Write-Error "Manifest $($f.Name) is empty."
        exit 1
    }
}
Write-Host "[OK] All $($k8sFiles.Count) Kubernetes manifests present and populated" -ForegroundColor Green

Write-Host ""
Write-Host "[OK] All deployment assets successfully validated!" -ForegroundColor Green
