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
	"text/template"
	"time"
)

func main() {
	manifestDir := flag.String("manifest-dir", "../../manifests", "Directory for manifests")
	skipSHA := flag.Bool("skip-sha", false, "Skip SHA256 computation")
	skipGUID := flag.Bool("skip-guid", false, "Skip Product GUID extraction")
	dryRun := flag.Bool("dry-run", false, "Preview only")
	versionTypes := flag.String("version-types", "LTS,MTS,Stable", "Comma-separated version types")
	minMajor := flag.Int("min-major", 10, "Minimum major version")
	_ = flag.Int("workers", 3, "Parallel workers for downloads")
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

		var installers []InstallerData
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
				tempFile, dlErr := downloadToTemp(url)
				if dlErr != nil {
					fmt.Fprintf(os.Stderr, "  Error downloading: %v\n", dlErr)
					guid = GUIDPlaceholder(manifestVersion)
				} else {
					guid, err = ExtractProductGUID(tempFile)
					if err != nil {
						fmt.Printf("  Warning: GUID extraction failed: %v\n", err)
						guid = GUIDPlaceholder(manifestVersion)
					}
					os.Remove(tempFile)
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
