# winget-mendix

Winget manifests for Mendix Studio Pro - side-by-side version management on Windows.

## Installation

### Local Testing (for now)

Winget doesn't support GitHub repos as direct sources. To test locally:

```powershell
# Clone the repository
git clone https://github.com/densmoe/winget-mendix.git
cd winget-mendix

# Add as local source (use absolute path)
winget source add mendix file:///C:/path/to/winget-mendix/manifests

# Install a version
winget install Mendix.MendixStudioPro --version 11.6.5 --source mendix

# List available versions
winget search Mendix.MendixStudioPro --source mendix
```

### Production Use

For global availability, manifests need to be submitted to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) via pull requests. Once stable and validated, we'll submit them upstream.

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
2. Filters by version type (LTS/MTS/Stable) and minimum version (**Mx9.24+**)
3. Checks CDN for all installer variants (machine x64, user x64, user arm64)
4. Fetches SHA256 hashes from CDN `.sha256` files (no full download needed)
5. Extracts real Product GUIDs from machine installers using 7-Zip (for proper upgrade/uninstall tracking)
6. Generates three YAML manifest files per version (version, installer, defaultLocale)
7. Daily GitHub Actions workflow processes 20 versions at a time with full GUID extraction

## Migration to Official Repository

Once stable, these manifests can be submitted to `microsoft/winget-pkgs` via pull requests.

## Related Projects

- [homebrew-mendix](https://github.com/densmoe/homebrew-mendix) - macOS Homebrew tap
