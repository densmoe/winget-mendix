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
- Go 1.21+ (for manifest generation)
- 7-Zip (for GUID extraction)
  - Windows: `choco install 7zip`
  - macOS: `brew install p7zip`

### Generate New Manifests

The daily workflow automatically generates manifests for new releases. To manually generate:

```bash
cd tools/manifest-generator
go run . -manifest-dir ../../manifests -min-major 9
```

### Extract Real GUIDs

**Windows (recommended):**
```powershell
cd tools
powershell -ExecutionPolicy Bypass -File .\update-guids-only.ps1
```

**macOS/Linux:**
```bash
cd tools
./extract-guids.sh ../manifests/Mendix/MendixStudioPro 5
```

### Generator Flags
- `-skip-sha`: Skip SHA256 computation (uses placeholders)
- `-skip-guid`: Skip Product GUID extraction (uses placeholders)
- `-dry-run`: Preview without writing files
- `-version-types`: Filter by type (default: `LTS,MTS,Stable`)
- `-min-major`: Minimum major version (default: `10`)
- `-max-versions`: Limit batch size (default: unlimited)
- `-workers`: Parallel workers (default: `5`)

## How It Works

1. Queries Mendix Marketplace API for all Studio Pro releases
2. Filters by version type (LTS/MTS/Stable) and minimum version (**Mx9.24+**)
3. Validates installer availability on CDN (HEAD request + Content-Length check to detect 404s)
4. Fetches SHA256 hashes from CDN `.sha256` files (no full download needed)
5. Optionally extracts real Product GUIDs from machine installers using 7-Zip
6. Generates three YAML manifest files per version (version, installer, defaultLocale)
7. Daily GitHub Actions workflow:
   - Fetches all releases from Marketplace
   - Filters out versions with existing manifests
   - Processes next 10 versions that need manifests
   - Skips to next batch automatically on subsequent runs

**Note:** Not all API releases have published installers. The generator automatically skips versions where installers don't exist on the CDN (e.g., some MTS/Stable patch releases for Mx9/10).

## Migration to Official Repository

Once stable, these manifests can be submitted to `microsoft/winget-pkgs` via pull requests.

## Related Projects

- [homebrew-mendix](https://github.com/densmoe/homebrew-mendix) - macOS Homebrew tap
