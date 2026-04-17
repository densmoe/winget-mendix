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

Each version includes up to three installer variants:

- **Machine x64** — traditional admin installer (`--scope machine`)
- **User x64** — no admin required (`--scope user`)
- **User ARM64** — no admin required, for ARM devices (`--scope user`)

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

## How It Works

1. Queries Mendix Marketplace API for all Studio Pro releases
2. Filters by version type (LTS/MTS/Stable) and minimum version (Mx10+)
3. Checks CDN for all installer variants (machine x64, user x64, user arm64)
4. Fetches SHA256 hashes from CDN `.sha256` files (no full download needed)
5. Optionally extracts Product GUIDs from machine installers using 7-Zip
6. Generates three YAML manifest files per version (package, installer, locale)
7. Daily GitHub Actions workflow automates updates

## Migration to Official Repository

Once stable, these manifests can be submitted to `microsoft/winget-pkgs` via pull requests.

## Related Projects

- [homebrew-mendix](https://github.com/densmoe/homebrew-mendix) - macOS Homebrew tap
