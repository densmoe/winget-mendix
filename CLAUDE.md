# winget-mendix

Winget manifest generator for Mendix Studio Pro versions.

## Project Structure

```
tools/manifest-generator/  Go CLI that generates winget manifests
  main.go                  CLI orchestration, SHA256 fetching, manifest completeness checks
  marketplace.go           Mendix Marketplace API client
  guid_extractor.go        ProductCode computation (SHA1-based, no Windows needed)
  guid_extractor_test.go   Tests against known-good ProductCodes
  templates.go             YAML manifest templates (version, installer, locale)
  generate-all.sh          Resumable batch script for generating all manifests
manifests/                 Generated YAML manifests (winget format)
.github/workflows/         CI + daily automation
```

## Key Commands

### Generate manifests
```bash
cd tools/manifest-generator
go run . -manifest-dir ../../manifests -min-major 9
go run . -manifest-dir ../../manifests -skip-sha      # Fast mode (placeholder hashes)
go run . -manifest-dir ../../manifests -dry-run        # Preview only
```

### Run tests
```bash
cd tools/manifest-generator
go test ./...
```

### Test locally on Windows
```powershell
winget source add mendix file:///{absolute-path-to-repo}/manifests
winget install Mendix.MendixStudioPro --version 11.9.1 --source mendix
```

## Architecture

The generator runs on any OS (Linux, macOS, Windows). No external tools needed — everything is pure Go.

- **marketplace.go**: Fetches releases from `marketplace.mendix.com/xas/` with CSRF token session init
- **guid_extractor.go**: Computes ProductCode as `{SHA1(UTF-16LE("Mendix Studio Pro <full_version>"))_is1}`
- **main.go**: Orchestrates generation with resumability — `manifestComplete()` checks all 3 YAML files exist with real hashes and GUIDs
- **templates.go**: Three templates per version (version, installer, locale)

## Mendix Version Patterns

- **Mx9**: 4-part `9.24.35.71123`, manifest version = `9.24.35`, ProductCode uses full 4-part
- **Mx10**: 4-part `10.18.0.54340`, manifest version = `10.18.0`, ProductCode uses full 4-part
- **Mx11.0–11.4**: 4-part URLs on CDN, 3-part manifest version
- **Mx11.5+**: 3-part everywhere (`11.5.0`)

The generator tries 4-part CDN URLs first and falls back to 3-part.

## Installer URLs

CDN base: `https://artifacts.rnd.mendix.com/modelers/`

- **Machine x64**: `Mendix-{VERSION}-Setup.exe`
- **User x64**: `Mendix-{VERSION}-User-x64-Setup.exe`
- **User ARM64**: `Mendix-{VERSION}-User-arm64-Setup.exe`

SHA256 sidecar files at `{url}.sha256` (available from 9.24.34+). Older versions need full download for hash computation.

CDN returns 200 with small error pages for missing files — the generator checks `Content-Length > 100MB`.

## Git Workflow

- Default branch: `main`
- Daily workflow commits new manifests automatically (up to 10/day)

## Safety Rules

- NEVER commit AWS credentials or secrets
- See ~/.claude/rules/security.md for full guidelines
