# Extract GUIDs from installers in Y:\ (mapped mendix-installers folder)
# and update manifests in the repository

param(
    [string]$InstallerPath = "Y:\",
    [string]$ManifestPath = "$PSScriptRoot\..\manifests\Mendix\MendixStudioPro"
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Mendix GUID Extractor" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Installer folder: $InstallerPath" -ForegroundColor Yellow
Write-Host "Manifest folder:  $ManifestPath" -ForegroundColor Yellow
Write-Host ""

# Get all machine installers (pattern: Mendix-*-Setup.exe without User/arm64)
$installers = Get-ChildItem -Path $InstallerPath -Filter "Mendix-*-Setup.exe" |
    Where-Object { $_.Name -notmatch "User" -and $_.Name -notmatch "arm64" } |
    Sort-Object Name

Write-Host "Found $($installers.Count) machine installers" -ForegroundColor Green
Write-Host ""

$processed = 0
$updated = 0
$skipped = 0
$failed = 0

foreach ($installer in $installers) {
    $processed++
    $filename = $installer.Name

    # Extract version from filename: Mendix-X.Y.Z(.BUILD)-Setup.exe
    if ($filename -match "Mendix-(.+?)-Setup\.exe") {
        $fullVersion = $matches[1]

        # Convert to manifest version (drop build number if present)
        $parts = $fullVersion -split "\."
        if ($parts.Count -eq 4) {
            # 4-part version: drop build number
            $manifestVersion = "$($parts[0]).$($parts[1]).$($parts[2])"
        } else {
            # 3-part version: use as-is
            $manifestVersion = $fullVersion
        }

        $versionDir = Join-Path $ManifestPath $manifestVersion
        $installerManifest = Join-Path $versionDir "Mendix.MendixStudioPro.installer.yaml"

        if (-not (Test-Path $installerManifest)) {
            Write-Host "[$processed/$($installers.Count)] ⊘ $filename (no manifest)" -ForegroundColor DarkGray
            $skipped++
            continue
        }

        # Check if GUID is already real (not a placeholder)
        $content = Get-Content $installerManifest -Raw
        $checkPattern = 'ProductCode: "(.+?)"'
        if ($content -match $checkPattern -and $matches[1] -notmatch "PLACEHOLDER") {
            Write-Host "[$processed/$($installers.Count)] ✓ $manifestVersion (already has GUID)" -ForegroundColor DarkGray
            $skipped++
            continue
        }

        Write-Host "[$processed/$($installers.Count)] ⚙ $filename..." -ForegroundColor Cyan -NoNewline

        # Extract MSI from installer
        $tempDir = Join-Path $env:TEMP "mendix-guid-$([System.IO.Path]::GetRandomFileName())"
        New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

        try {
            # Extract using 7-Zip
            & 7z e "$($installer.FullName)" -o"$tempDir" "*.msi" -r -y 2>&1 | Out-Null

            $msiFiles = Get-ChildItem -Path $tempDir -Filter "*.msi"
            if ($msiFiles.Count -eq 0) {
                Write-Host "`r[$processed/$($installers.Count)] ✗ $filename (no MSI found)" -ForegroundColor Red
                $failed++
                continue
            }

            # Extract GUID from first MSI
            $msi = $msiFiles[0]
            $guid = & 7z l "$($msi.FullName)" | Select-String 'Subject' | ForEach-Object {
                if ($_ -match '\{([A-F0-9-]+)\}') {
                    $matches[1]
                }
            }

            if (-not $guid) {
                Write-Host "`r[$processed/$($installers.Count)] ✗ $filename (GUID not found)" -ForegroundColor Red
                $failed++
                continue
            }

            # Update manifest with real GUID
            $guid = "{$guid}"
            $replacePattern = 'ProductCode: ".+?"'
            $replacement = "ProductCode: `"$guid`""
            $newContent = $content -replace $replacePattern, $replacement
            $newContent | Set-Content $installerManifest -NoNewline

            Write-Host "`r[$processed/$($installers.Count)] ✓ $manifestVersion -> $guid" -ForegroundColor Green
            $updated++

        } catch {
            Write-Host "`r[$processed/$($installers.Count)] ✗ $filename (error: $_)" -ForegroundColor Red
            $failed++
        } finally {
            Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    } else {
        Write-Host "[$processed/$($installers.Count)] ⊘ $filename (invalid name format)" -ForegroundColor DarkGray
        $skipped++
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Updated:  $updated" -ForegroundColor Green
Write-Host "Skipped:  $skipped" -ForegroundColor Yellow
Write-Host "Failed:   $failed" -ForegroundColor Red
Write-Host "========================================" -ForegroundColor Cyan
