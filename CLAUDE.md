# winget-mendix

Winget manifest generator for Mendix Studio Pro versions on Windows.

## Project Structure

```
tools/manifest-generator/  Go CLI that generates winget manifests
manifests/                 Generated YAML manifests (winget format)
.github/workflows/         Daily automation (update-manifests.yml)
```

## Key Commands

### Generate manifests
```powershell
cd tools/manifest-generator
go run . -manifest-dir ../../manifests
go run . -manifest-dir ../../manifests -skip-sha      # Fast mode
go run . -manifest-dir ../../manifests -dry-run       # Preview only
```

### Test locally
```powershell
# Add local source
winget source add mendix file:///{absolute-path-to-repo}/manifests

# Install
winget install Mendix.MendixStudioPro --version 11.9.1 --source mendix
```

## Architecture

- **marketplace.go**: Mendix API client (queries releases from marketplace.mendix.com)
- **guid_extractor.go**: Extracts Product GUID from .exe using 7-Zip
- **templates.go**: YAML manifest templates (package, installer, locale)
- **main.go**: CLI orchestration and file generation

## Mendix Version Patterns

- **Mx11+**: Semver format `11.9.1`, manifest version = download version
- **Mx10**: 4-part format `10.24.13.86719`, manifest version = `10.24.13` (drop build)

## Installer URLs

- x64: `https://artifacts.rnd.mendix.com/modelers/Mendix-{VERSION}-Setup.exe`
- ARM64: `https://artifacts.rnd.mendix.com/modelers/Mendix-{VERSION}-Setup-ARM64.exe`

## Git Workflow

- Default branch: `main`
- Never push directly to `main` - use branches and PRs
- Daily workflow commits new manifests automatically

## Safety Rules

- NEVER commit AWS credentials or secrets
- NEVER push to production without review
- See ~/.claude/rules/security.md for full guidelines
