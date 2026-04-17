# winget-mendix

Winget manifests for Mendix Studio Pro - side-by-side version management on Windows.

## Installation

### Add Custom Source
```powershell
winget source add mendix https://github.com/densmoe/winget-mendix
```

### Install Specific Version
```powershell
# Install Mx11.9.1
winget install Mendix.MendixStudioPro --version 11.9.1 --source mendix

# Install Mx10.24.13
winget install Mendix.MendixStudioPro --version 10.24.13 --source mendix
```

### List Available Versions
```powershell
winget search Mendix.MendixStudioPro --source mendix
```

## Architecture Support

Both x64 and ARM64 installers are supported where available.

## Developer Setup

### Prerequisites
- Windows 10+ with winget
- Go 1.21+
- 7-Zip (`choco install 7zip`)

### Generate Manifests
```powershell
cd tools/manifest-generator
go run . -manifest-dir ../../manifests
```

### Flags
- `-skip-sha`: Skip SHA256 computation (faster, uses placeholders)
- `-skip-guid`: Skip Product GUID extraction (uses placeholders)
- `-dry-run`: Preview without writing files
- `-version-types`: Filter by type (default: `LTS,MTS,Stable`)
- `-min-major`: Minimum major version (default: `10`)
- `-workers`: Parallel downloads (default: `3`)

## How It Works

1. Queries Mendix Marketplace API for all Studio Pro releases
2. Filters by version type (LTS/MTS/Stable) and minimum version (Mx10.7+)
3. Checks CDN for x64 and ARM64 installer availability
4. Downloads installers and computes SHA256 hashes
5. Extracts Product GUIDs from .exe files using 7-Zip
6. Generates three YAML manifest files per version
7. Daily GitHub Actions workflow automates updates

## Migration to Official Repository

Once stable, these manifests can be submitted to `microsoft/winget-pkgs` via pull requests.

## Related Projects

- [homebrew-mendix](https://github.com/densmoe/homebrew-mendix) - macOS Homebrew tap
