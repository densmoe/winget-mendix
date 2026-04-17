# Testing winget-mendix on Windows

## Prerequisites

- Windows 10/11 with winget installed
- Git for Windows

## Step 1: Clone the Repository

```powershell
cd C:\
git clone https://github.com/densmoe/winget-mendix.git
cd winget-mendix
```

## Step 2: Add Local Source

```powershell
winget source add mendix file:///C:/winget-mendix/manifests
```

Verify it was added:
```powershell
winget source list
```

## Step 3: Search Available Versions

```powershell
winget search Mendix.MendixStudioPro --source mendix
```

You should see:
- 10.18.6
- 10.18.7
- 10.24.4
- 10.24.9
- 10.24.13
- 10.24.15
- 11.6.0
- 11.6.5

## Step 4: Install a Version

### Option A: User-scoped (no admin required, ARM64 supported)

```powershell
winget install Mendix.MendixStudioPro --version 11.6.5 --source mendix --scope user
```

### Option B: Machine-scoped (requires admin)

```powershell
winget install Mendix.MendixStudioPro --version 11.6.5 --source mendix --scope machine
```

## Step 5: Verify Installation

Check if Mendix Studio Pro launched successfully:

```powershell
# Check installed programs
winget list Mendix
```

Try launching Studio Pro from Start Menu or installation directory.

## Step 6: Test Side-by-Side Install

Install a second version to verify side-by-side support:

```powershell
winget install Mendix.MendixStudioPro --version 10.24.13 --source mendix --scope user
```

Both versions should be installed independently.

## Cleanup (Optional)

```powershell
# Uninstall versions
winget uninstall Mendix.MendixStudioPro --version 11.6.5
winget uninstall Mendix.MendixStudioPro --version 10.24.13

# Remove local source
winget source remove mendix

# Delete repo
cd C:\
Remove-Item -Recurse -Force C:\winget-mendix
```

## Troubleshooting

### "Source agreement required"
Run with `--accept-source-agreements`:
```powershell
winget install Mendix.MendixStudioPro --version 11.6.5 --source mendix --accept-source-agreements
```

### "ProductCode placeholder warning"
This is expected. The placeholder GUIDs work for installation but may cause issues with upgrades/uninstall detection.

### Installation fails
Check the log:
```powershell
winget install Mendix.MendixStudioPro --version 11.6.5 --source mendix --verbose --logs
```

## Next Steps

If installation works:
1. Enable full GUID extraction in CI workflow
2. Wait for all versions to be generated (~12 days at 10 versions/day)
3. Submit to microsoft/winget-pkgs for global availability
