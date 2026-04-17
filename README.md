# winget-mendix

Winget manifests for Mendix Studio Pro - side-by-side version management on Windows.

## Installation

### Local Testing

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

For global availability, manifests need to be submitted to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) via pull requests.

## Architecture Support

Each version includes up to three installer variants:

- **Machine x64** — traditional admin installer (`--scope machine`)
- **User x64** — no admin required (`--scope user`)
- **User ARM64** — no admin required, for ARM devices (`--scope user`)

## Developer Setup

### Prerequisites
- Go 1.21+

### Generate New Manifests

The daily workflow automatically generates manifests for new releases. To manually generate:

```bash
cd tools/manifest-generator
go run . -manifest-dir ../../manifests -min-major 9
```

For a full batch run (resumable, commits progress):

```bash
cd tools/manifest-generator
nohup ./generate-all.sh &
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
2. Filters by version type (LTS/MTS/Stable) and minimum version (Mx9.24+)
3. Skips versions that already have complete manifests (all 3 YAML files with real hashes and GUIDs)
4. Validates installer availability on CDN (HEAD request + Content-Length > 100MB to detect error pages)
5. Fetches SHA256 hashes from CDN `.sha256` sidecar files, or streams the full download for older versions
6. Computes Product GUIDs using the Inno Setup AppId formula (pure SHA1, no Windows tools needed)
7. Generates three YAML manifest files per version (version, installer, locale)

### Daily Automation

The GitHub Actions workflow runs daily and processes up to 10 new versions per run, catching up incrementally as new Mendix releases are published.

## Reverse Engineering Notes

These notes document how installer metadata was reverse-engineered, in case Mendix changes their packaging.

### ProductCode (Inno Setup AppId)

Mendix Studio Pro uses Inno Setup, which generates a ProductCode from the `AppId` parameter. The installer's `[Setup]` section contains:

```ini
AppId={code:GetSHA1OfUnicodeString|Mendix Studio Pro {VERSION}}
```

This means the ProductCode is: `{SHA1(UTF-16LE("Mendix Studio Pro <full_version>"))_is1}`

The full version string varies by major version:
- **Mx9/Mx10**: 4-part version with build number, e.g. `10.18.0.54340`
- **Mx11**: 3-part semver, e.g. `11.5.0`

This was discovered by extracting the Inno Setup header using `innoextract` built with debug mode, which revealed the `{code:GetSHA1OfUnicodeString|...}` dynamic call. Verified against 4 known ProductCodes from Windows registry entries.

The Go implementation (`guid_extractor.go`) computes this purely — no Windows, no 7-Zip, no installer download needed.

### SHA256 Hashes

Newer versions (9.24.34+) have `.sha256` sidecar files on the CDN at `{installer-url}.sha256`. Older versions require downloading the full installer (~500MB-1GB) and computing the hash in-memory.

### CDN URL Patterns

Base URL: `https://artifacts.rnd.mendix.com/modelers/`

- **Mx9/Mx10**: URLs use 4-part version with build number: `Mendix-10.18.0.54340-Setup.exe`
- **Mx11.0–11.4**: URLs use 4-part version: `Mendix-11.4.0.84320-Setup.exe`
- **Mx11.5+**: URLs use 3-part version: `Mendix-11.5.0-Setup.exe`

The generator tries the 4-part URL first and falls back to 3-part if it 404s. The CDN returns HTTP 200 with a small error page for missing files, so the generator checks `Content-Length > 100MB` to distinguish real installers from error pages.

## Migration to Official Repository

Once stable, these manifests can be submitted to `microsoft/winget-pkgs` via pull requests.

## Related Projects

- [homebrew-mendix](https://github.com/densmoe/homebrew-mendix) - macOS Homebrew tap
