# Winget-Mendix Implementation Plan

## Project Context

Create a Windows Package Manager (winget) repository for installing Mendix Studio Pro versions side-by-side on Windows. This mirrors the functionality of `homebrew-mendix` (macOS) but uses winget manifests.

**User:** densmoe (Dennis Möller)  
**Source repo:** `densmoe/homebrew-mendix` (GitHub)  
**New repo:** `densmoe/winget-mendix` (to be created)  
**Distribution:** Custom winget source initially, migrate to microsoft/winget-pkgs later

## Key Requirements

1. **Side-by-side installs** - multiple Mendix Studio Pro versions installed simultaneously
2. **ARM64 support** - must support ARM64 from day one (Parallels VM usage)
3. **Automated updates** - daily GitHub Actions workflow to check for new versions
4. **Product GUID extraction** - automatically extract from .exe installers
5. **Independent operation** - users can set up with `gh` CLI without manual GitHub UI steps

## Mendix Version Patterns

| Version Type | Format | Example | Installer URL |
|-------------|--------|---------|---------------|
| Mx11+ | Semver (3 parts) | `11.9.1` | `Mendix-11.9.1-Setup.exe` |
| Mx10 | 4 parts with build | `10.24.13.86719` | `Mendix-10.24.13.86719-Setup.exe` |

**Important:** Mx10 manifest version should be `10.24.13` (drop build number) but installer URL uses full version.

**CDN:** `https://artifacts.rnd.mendix.com/modelers/`

**Minimum version:** Mx10.7 (earlier versions lack Mac/Windows modern installers)

## Repository Structure

```
winget-mendix/
├── README.md
├── CLAUDE.md
├── .gitignore
├── manifests/
│   └── Mendix/
│       └── MendixStudioPro/
│           ├── 11.9.1/
│           │   ├── Mendix.MendixStudioPro.yaml
│           │   ├── Mendix.MendixStudioPro.installer.yaml
│           │   └── Mendix.MendixStudioPro.locale.en-US.yaml
│           └── 10.24.13/
│               └── (same structure)
├── tools/
│   └── manifest-generator/
│       ├── main.go
│       ├── marketplace.go
│       ├── guid_extractor.go
│       ├── templates.go
│       ├── go.mod
│       └── go.sum
└── .github/workflows/
    ├── update-manifests.yml
    └── ci.yml
```

## Implementation Steps

### Phase 1: Repository Setup

```bash
# Create GitHub repo using gh CLI
gh repo create winget-mendix --public --description "Winget manifests for Mendix Studio Pro - side-by-side version management on Windows" --clone

cd winget-mendix
git checkout -b initial-setup

# Create directory structure
mkdir -p manifests/Mendix/MendixStudioPro
mkdir -p tools/manifest-generator
mkdir -p .github/workflows
```

### Phase 2: Core Files

#### .gitignore
```
# Binaries
*.exe
!tools/manifest-generator/manifest-generator.exe
*.dll
*.so
*.dylib

# Test downloads
downloads/
temp/
*.msi
*.pkg

# Go
go.work
go.work.sum

# IDE
.vscode/
.idea/
*.swp
*.swo
```

#### go.mod (tools/manifest-generator/go.mod)
```go
module github.com/densmoe/winget-mendix/tools/manifest-generator

go 1.21

require github.com/google/uuid v1.6.0
```

### Phase 3: Go Implementation

#### marketplace.go
**Copy from homebrew-mendix with minor adaptations:**

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type MarketplaceClient struct {
	baseURL    string
	httpClient *http.Client
	csrfToken  string
}

type Release struct {
	Major       int
	Minor       int
	Patch       int
	Build       int
	VersionType string
	VersionFull string
	IsStable    bool
}

func NewMarketplaceClient() (*MarketplaceClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := &MarketplaceClient{
		baseURL: "https://marketplace.mendix.com/xas/",
		httpClient: &http.Client{
			Jar: jar,
		},
	}

	if err := client.initSession(); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *MarketplaceClient) initSession() error {
	reqID := uuid.New().String()
	payload := map[string]interface{}{
		"action": "get_session_data",
		"params": map[string]interface{}{},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mx-ReqToken", reqID)
	req.Header.Set("Cookie", "DeviceType=Desktop; Profile=Responsive")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}

	if csrf, ok := result["csrftoken"].(string); ok {
		c.csrfToken = csrf
	}

	return nil
}

func (c *MarketplaceClient) FetchReleases(versionTypes []string, minMajor int) ([]Release, error) {
	var allReleases []Release
	offset := 0
	pageSize := 50

	for {
		releases, hasMore, err := c.fetchPage(offset, pageSize, versionTypes, minMajor)
		if err != nil {
			return nil, err
		}

		allReleases = append(allReleases, releases...)

		if !hasMore {
			break
		}
		offset += pageSize
	}

	return allReleases, nil
}

func (c *MarketplaceClient) fetchPage(offset, limit int, versionTypes []string, minMajor int) ([]Release, bool, error) {
	reqID := uuid.New().String()
	payload := map[string]interface{}{
		"action": "retrieve_by_xpath",
		"params": map[string]interface{}{
			"xpath": "//AppStore.Framework",
			"schema": map[string]interface{}{
				"amount": limit,
				"offset": offset,
				"sort": [][]interface{}{
					{"Major", "desc"},
					{"Minor", "desc"},
					{"Patch", "desc"},
					{"Build", "desc"},
				},
			},
			"count":      true,
			"aggregates": false,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, false, err
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mx-ReqToken", reqID)
	req.Header.Set("X-Csrf-Token", c.csrfToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, err
	}

	var result struct {
		Objects      []map[string]interface{} `json:"objects"`
		HasMoreItems bool                     `json:"hasMoreItems"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, false, err
	}

	var releases []Release
	for _, obj := range result.Objects {
		release, err := parseRelease(obj, versionTypes, minMajor)
		if err != nil {
			continue
		}
		if release != nil {
			releases = append(releases, *release)
		}
	}

	return releases, result.HasMoreItems, nil
}

func parseRelease(obj map[string]interface{}, versionTypes []string, minMajor int) (*Release, error) {
	versionText := getString(obj, "VersionText")
	versionType := getString(obj, "VersionType")
	status := getString(obj, "Status")

	if status == "Deprecated" {
		return nil, nil
	}

	if !contains(versionTypes, versionType) {
		return nil, nil
	}

	versionText = strings.Split(versionText, " (build")[0]
	parts := strings.Split(versionText, ".")

	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid version: %s", versionText)
	}

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])
	build := 0
	if len(parts) == 4 {
		build, _ = strconv.Atoi(parts[3])
	}

	if major < minMajor {
		return nil, nil
	}

	fullVersion := versionText
	if build > 0 {
		fullVersion = fmt.Sprintf("%d.%d.%d.%d", major, minor, patch, build)
	}

	return &Release{
		Major:       major,
		Minor:       minor,
		Patch:       patch,
		Build:       build,
		VersionType: versionType,
		VersionFull: fullVersion,
		IsStable:    versionType == "LTS" || versionType == "MTS" || versionType == "Stable",
	}, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
```

#### guid_extractor.go
```go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var guidPattern = regexp.MustCompile(`\{[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}\}`)

func ExtractProductGUID(exePath string) (string, error) {
	tempDir, err := os.MkdirTemp("", "mendix-extract-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract .exe with 7z
	cmd := exec.Command("7z", "x", exePath, fmt.Sprintf("-o%s", tempDir), "-y")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("7z extract failed: %w", err)
	}

	// Find MSI files
	msiFiles, err := filepath.Glob(filepath.Join(tempDir, "*.msi"))
	if err != nil || len(msiFiles) == 0 {
		// Try nested
		msiFiles, _ = filepath.Glob(filepath.Join(tempDir, "**", "*.msi"))
	}

	if len(msiFiles) == 0 {
		return "", fmt.Errorf("no MSI found in installer")
	}

	// Extract ProductCode from MSI
	cmd = exec.Command("7z", "l", msiFiles[0])
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("7z list MSI failed: %w", err)
	}

	// Search for GUID pattern
	matches := guidPattern.FindAllString(string(output), -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no GUID found in MSI")
	}

	// Return first valid GUID (usually ProductCode)
	return strings.ToUpper(matches[0]), nil
}

func GUIDPlaceholder(version string) string {
	return fmt.Sprintf("{MENDIX-STUDIO-PRO-%s-PLACEHOLDER}", strings.ReplaceAll(version, ".", "-"))
}
```

#### templates.go
```go
package main

import "text/template"

var packageManifestTemplate = template.Must(template.New("package").Parse(`PackageIdentifier: Mendix.MendixStudioPro
PackageVersion: {{.Version}}
PackageName: Mendix Studio Pro
Publisher: Mendix
PublisherUrl: https://www.mendix.com/
PublisherSupportUrl: https://www.mendix.com/support/
PackageUrl: https://www.mendix.com/studio-pro/
License: Proprietary
ShortDescription: Low-code application development platform
ManifestVersion: 1.4.0
`))

var installerManifestTemplate = template.Must(template.New("installer").Parse(`PackageIdentifier: Mendix.MendixStudioPro
PackageVersion: {{.Version}}
InstallModes:
  - interactive
  - silent
Installers:{{range .Installers}}
  - Architecture: {{.Arch}}
    InstallerType: exe
    InstallerUrl: {{.URL}}
    InstallerSha256: {{.SHA256}}
    ProductCode: "{{.GUID}}"{{end}}
InstallationNotes: "Multiple versions can be installed side-by-side."
ManifestVersion: 1.4.0
`))

var localeManifestTemplate = template.Must(template.New("locale").Parse(`PackageIdentifier: Mendix.MendixStudioPro
PackageVersion: {{.Version}}
PackageLocale: en-US
Publisher: Mendix
PublisherUrl: https://www.mendix.com/
PublisherSupportUrl: https://www.mendix.com/support/
PackageUrl: https://www.mendix.com/studio-pro/
License: Proprietary
ShortDescription: Low-code application development platform
ManifestVersion: 1.4.0
`))

type ManifestData struct {
	Version    string
	Installers []InstallerData
}

type InstallerData struct {
	Arch   string
	URL    string
	SHA256 string
	GUID   string
}
```

#### main.go (key sections)
```go
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	manifestDir := flag.String("manifest-dir", "../../manifests", "Directory for manifests")
	skipSHA := flag.Bool("skip-sha", false, "Skip SHA256 computation")
	skipGUID := flag.Bool("skip-guid", false, "Skip Product GUID extraction")
	dryRun := flag.Bool("dry-run", false, "Preview only")
	versionTypes := flag.String("version-types", "LTS,MTS,Stable", "Comma-separated version types")
	minMajor := flag.Int("min-major", 10, "Minimum major version")
	workers := flag.Int("workers", 3, "Parallel workers for downloads")
	flag.Parse()

	types := strings.Split(*versionTypes, ",")
	
	client, err := NewMarketplaceClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	releases, err := client.FetchReleases(types, *minMajor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching releases: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d releases\n", len(releases))

	for _, release := range releases {
		manifestVersion := manifestVersionFor(release)
		versionDir := filepath.Join(*manifestDir, "Mendix", "MendixStudioPro", manifestVersion)

		if _, err := os.Stat(versionDir); err == nil {
			fmt.Printf("Skipping %s (already exists)\n", manifestVersion)
			continue
		}

		fmt.Printf("Processing %s...\n", manifestVersion)

		// Check both x64 and arm64 installers
		installers := []InstallerData{}
		for _, arch := range []string{"x64", "arm64"} {
			url := installerURL(release, arch)
			
			if !urlExists(url) {
				fmt.Printf("  %s: not available\n", arch)
				continue
			}

			var sha string
			var guid string

			if *skipSHA {
				sha = "SHA256_PLACEHOLDER"
			} else {
				sha, err = computeSHA256(url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  Error computing SHA256: %v\n", err)
					continue
				}
			}

			if *skipGUID {
				guid = GUIDPlaceholder(manifestVersion)
			} else {
				tempFile, _ := downloadToTemp(url)
				defer os.Remove(tempFile)
				
				guid, err = ExtractProductGUID(tempFile)
				if err != nil {
					fmt.Printf("  Warning: GUID extraction failed: %v\n", err)
					guid = GUIDPlaceholder(manifestVersion)
				}
			}

			installers = append(installers, InstallerData{
				Arch:   arch,
				URL:    url,
				SHA256: sha,
				GUID:   guid,
			})
			fmt.Printf("  %s: OK\n", arch)
		}

		if len(installers) == 0 {
			fmt.Printf("  No installers found, skipping\n")
			continue
		}

		if *dryRun {
			fmt.Printf("  [DRY RUN] Would create: %s\n", versionDir)
			continue
		}

		if err := os.MkdirAll(versionDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
			continue
		}

		data := ManifestData{
			Version:    manifestVersion,
			Installers: installers,
		}

		writeManifest(filepath.Join(versionDir, "Mendix.MendixStudioPro.yaml"), packageManifestTemplate, data)
		writeManifest(filepath.Join(versionDir, "Mendix.MendixStudioPro.installer.yaml"), installerManifestTemplate, data)
		writeManifest(filepath.Join(versionDir, "Mendix.MendixStudioPro.locale.en-US.yaml"), localeManifestTemplate, data)

		fmt.Printf("  Created manifests in %s\n", versionDir)
	}
}

func manifestVersionFor(r Release) string {
	if r.Build > 0 {
		return fmt.Sprintf("%d.%d.%d", r.Major, r.Minor, r.Patch)
	}
	return fmt.Sprintf("%d.%d.%d", r.Major, r.Minor, r.Patch)
}

func installerURL(r Release, arch string) string {
	version := r.VersionFull
	archSuffix := ""
	if arch == "arm64" {
		archSuffix = "-ARM64"
	}
	return fmt.Sprintf("https://artifacts.rnd.mendix.com/modelers/Mendix-%s-Setup%s.exe", version, archSuffix)
}

func urlExists(url string) bool {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func computeSHA256(url string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	h := sha256.New()
	if _, err := io.Copy(h, resp.Body); err != nil {
		return "", err
	}

	return fmt.Sprintf("%X", h.Sum(nil)), nil
}

func downloadToTemp(url string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tempFile, err := os.CreateTemp("", "mendix-*.exe")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

func writeManifest(path string, tmpl *template.Template, data ManifestData) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}
```

### Phase 4: GitHub Workflows

#### .github/workflows/update-manifests.yml
```yaml
name: Update Manifests
on:
  schedule:
    - cron: '0 6 * * *'
  workflow_dispatch:

jobs:
  update:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          
      - name: Install 7-Zip
        run: choco install 7zip -y
      
      - name: Build generator
        run: |
          cd tools/manifest-generator
          go build -o manifest-generator.exe .
      
      - name: Generate manifests
        run: |
          cd tools/manifest-generator
          ./manifest-generator.exe -manifest-dir ../../manifests
      
      - name: Commit and push
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git add manifests/
          $newFiles = @(git diff --cached --name-only --diff-filter=A)
          if ($newFiles.Count -gt 0) {
            $versions = $newFiles | ForEach-Object { ($_ -split '/')[3] } | Select-Object -Unique
            $message = "Add manifests for: $($versions -join ', ')"
            git commit -m $message
            git push
          } else {
            Write-Host "No new manifests to commit"
          }
```

#### .github/workflows/ci.yml
```yaml
name: CI
on: [push, pull_request]

jobs:
  validate:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Validate manifests
        run: |
          $manifests = Get-ChildItem manifests -Recurse -Filter *.installer.yaml
          foreach ($manifest in $manifests) {
            Write-Host "Validating $($manifest.FullName)"
            winget validate $manifest.FullName
            if ($LASTEXITCODE -ne 0) {
              Write-Error "Validation failed for $($manifest.FullName)"
              exit 1
            }
          }
```

### Phase 5: Documentation

#### README.md
```markdown
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
```

#### CLAUDE.md
```markdown
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
```

## Testing Checklist

After setting up the repo:

1. **Local build**
   ```powershell
   cd tools/manifest-generator
   go build
   ./manifest-generator.exe -dry-run -manifest-dir ../../manifests
   ```

2. **Generate one manifest**
   ```powershell
   ./manifest-generator.exe -manifest-dir ../../manifests -skip-guid
   ```

3. **Validate manifest**
   ```powershell
   winget validate manifests/Mendix/MendixStudioPro/11.9.1/Mendix.MendixStudioPro.installer.yaml
   ```

4. **Test installation** (if manifest created)
   ```powershell
   winget source add mendix file:///{repo-path}/manifests
   winget search Mendix.MendixStudioPro --source mendix
   # Only install if you want to test
   ```

5. **Push and test workflow**
   ```bash
   git push origin initial-setup
   # Create PR, merge to main
   # Go to Actions → Update Manifests → Run workflow
   ```

## ARM64 Notes

Mendix may not provide ARM64 installers for all versions. The generator will:
- Check both x64 and ARM64 URLs
- Include only available installers in the manifest
- Support multi-architecture manifests (both in one file)

## Next Steps After Setup

1. Run generator manually to create first manifests
2. Test installation on Windows machine (both x64 and ARM64 if available)
3. Enable scheduled workflow for daily updates
4. Monitor for new Mendix releases
5. Consider submitting to microsoft/winget-pkgs when stable
