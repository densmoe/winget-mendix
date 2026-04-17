# Update GUIDs in existing manifests (no downloads, uses cached installers)
# Run this incrementally - processes one version at a time, you can stop/restart anytime

param(
    [string]$ManifestDir = "..\manifests\Mendix\MendixStudioPro",
    [string]$CacheDir = "$env:TEMP\mendix-installers"
)

if (!(Test-Path $CacheDir)) {
    New-Item -ItemType Directory -Path $CacheDir | Out-Null
}

Write-Host "Updating GUIDs in manifests (using cache: $CacheDir)" -ForegroundColor Cyan
Write-Host "Downloads cached for reuse. Safe to stop/restart anytime." -ForegroundColor Yellow
Write-Host ""

$versions = Get-ChildItem $ManifestDir -Directory | Sort-Object Name

foreach ($versionDir in $versions) {
    $version = $versionDir.Name
    $installerYaml = Join-Path $versionDir.FullName "Mendix.MendixStudioPro.installer.yaml"

    if (!(Test-Path $installerYaml)) {
        continue
    }

    $content = Get-Content $installerYaml -Raw

    if ($content -notmatch "PLACEHOLDER") {
        Write-Host "[$version] Already has real GUID" -ForegroundColor Green
        continue
    }

    if ($content -match "Scope: machine[\s\S]*?InstallerUrl: (https://[^\s]+)") {
        $url = $matches[1]
        $filename = [System.IO.Path]::GetFileName($url)
        $cachedFile = Join-Path $CacheDir $filename

        # Download if not cached
        if (!(Test-Path $cachedFile)) {
            Write-Host "[$version] Downloading (will cache)..." -ForegroundColor Yellow
            try {
                Invoke-WebRequest -Uri $url -OutFile $cachedFile -ErrorAction Stop
            } catch {
                Write-Host "[$version] Download failed: $_" -ForegroundColor Red
                continue
            }
        } else {
            Write-Host "[$version] Using cached installer" -ForegroundColor Cyan
        }

        # Extract GUID
        $extractDir = Join-Path ([System.IO.Path]::GetTempPath()) ("mendix-extract-" + [guid]::NewGuid())
        try {
            & 7z x $cachedFile "-o$extractDir" -y | Out-Null
            $msi = Get-ChildItem $extractDir -Filter *.msi -Recurse | Select-Object -First 1

            if ($msi) {
                $guid = (& 7z l $msi.FullName | Select-String '\{[0-9A-F]{8}-([0-9A-F]{4}-){3}[0-9A-F]{12}\}').Matches[0].Value

                if ($guid) {
                    $placeholder = "{MENDIX-STUDIO-PRO-$($version.Replace('.', '-'))-PLACEHOLDER}"
                    $newContent = $content -replace [regex]::Escape($placeholder), $guid
                    Set-Content -Path $installerYaml -Value $newContent -NoNewline

                    Write-Host "[$version] Updated with GUID: $guid" -ForegroundColor Green
                } else {
                    Write-Host "[$version] Could not extract GUID" -ForegroundColor Red
                }
            } else {
                Write-Host "[$version] No MSI found" -ForegroundColor Red
            }
        } catch {
            Write-Host "[$version] Extraction failed: $_" -ForegroundColor Red
        } finally {
            Remove-Item $extractDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    Write-Host ""
}

Write-Host "Done! Downloaded installers cached in: $CacheDir" -ForegroundColor Cyan
Write-Host "Commit and push the updated manifests when ready." -ForegroundColor Cyan
