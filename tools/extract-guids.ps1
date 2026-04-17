# Extract GUIDs for all existing manifests locally
# Run this once on Windows to populate real ProductCodes

param(
    [string]$ManifestDir = "..\manifests\Mendix\MendixStudioPro"
)

Write-Host "Extracting Product GUIDs for all manifests..." -ForegroundColor Cyan
Write-Host "This will download machine installers (~600MB each) to extract GUIDs." -ForegroundColor Yellow
Write-Host ""

$versions = Get-ChildItem $ManifestDir -Directory | Sort-Object Name

foreach ($versionDir in $versions) {
    $version = $versionDir.Name
    $installerYaml = Join-Path $versionDir.FullName "Mendix.MendixStudioPro.installer.yaml"

    if (!(Test-Path $installerYaml)) {
        Write-Host "[$version] No installer manifest found, skipping" -ForegroundColor Gray
        continue
    }

    $content = Get-Content $installerYaml -Raw

    # Check if already has real GUID (not placeholder)
    if ($content -notmatch "PLACEHOLDER") {
        Write-Host "[$version] Already has real GUID, skipping" -ForegroundColor Green
        continue
    }

    # Extract machine installer URL
    if ($content -match "Scope: machine[\s\S]*?InstallerUrl: (https://[^\s]+)") {
        $url = $matches[1]
        Write-Host "[$version] Downloading $url..." -ForegroundColor Yellow

        $tempFile = [System.IO.Path]::GetTempFileName() + ".exe"

        try {
            # Download
            Invoke-WebRequest -Uri $url -OutFile $tempFile -ErrorAction Stop

            # Extract with 7z to temp dir
            $extractDir = [System.IO.Path]::GetTempFileName()
            Remove-Item $extractDir
            New-Item -ItemType Directory -Path $extractDir | Out-Null

            & 7z x $tempFile "-o$extractDir" -y | Out-Null

            # Find MSI
            $msiFiles = Get-ChildItem $extractDir -Filter *.msi -Recurse

            if ($msiFiles.Count -eq 0) {
                Write-Host "[$version] No MSI found in installer" -ForegroundColor Red
                continue
            }

            # Extract GUID from MSI
            $msiFile = $msiFiles[0].FullName
            $output = & 7z l $msiFile | Out-String

            # Find GUID pattern
            if ($output -match '\{[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}\}') {
                $guid = $matches[0].ToUpper()
                Write-Host "[$version] Found GUID: $guid" -ForegroundColor Green

                # Replace placeholder in YAML
                $placeholder = "{MENDIX-STUDIO-PRO-$($version.Replace('.', '-'))-PLACEHOLDER}"
                $newContent = $content -replace [regex]::Escape($placeholder), $guid

                Set-Content -Path $installerYaml -Value $newContent -NoNewline
                Write-Host "[$version] Updated manifest" -ForegroundColor Green
            } else {
                Write-Host "[$version] Could not extract GUID from MSI" -ForegroundColor Red
            }

            # Cleanup
            Remove-Item $tempFile -Force -ErrorAction SilentlyContinue
            Remove-Item $extractDir -Recurse -Force -ErrorAction SilentlyContinue

        } catch {
            Write-Host "[$version] Error: $_" -ForegroundColor Red
        }
    } else {
        Write-Host "[$version] Could not find machine installer URL" -ForegroundColor Red
    }

    Write-Host ""
}

Write-Host "Done! Commit and push the updated manifests." -ForegroundColor Cyan
