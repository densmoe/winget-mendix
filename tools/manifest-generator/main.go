package main

import (
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

type installerVariant struct {
	name                 string
	arch                 string
	scope                string
	elevationRequirement string
	urlFunc              func(Release) string
}

var variants = []installerVariant{
	{
		name:                 "machine x64",
		arch:                 "x64",
		scope:                "machine",
		elevationRequirement: "elevatesSelf",
		urlFunc: func(r Release) string {
			return fmt.Sprintf("https://artifacts.rnd.mendix.com/modelers/Mendix-%s-Setup.exe", r.VersionFull)
		},
	},
	{
		name:  "user x64",
		arch:  "x64",
		scope: "user",
		urlFunc: func(r Release) string {
			return fmt.Sprintf("https://artifacts.rnd.mendix.com/modelers/Mendix-%s-User-x64-Setup.exe", r.VersionFull)
		},
	},
	{
		name:  "user arm64",
		arch:  "arm64",
		scope: "user",
		urlFunc: func(r Release) string {
			return fmt.Sprintf("https://artifacts.rnd.mendix.com/modelers/Mendix-%s-User-arm64-Setup.exe", r.VersionFull)
		},
	},
}

func main() {
	manifestDir := flag.String("manifest-dir", "../../manifests", "Directory for manifests")
	skipSHA := flag.Bool("skip-sha", false, "Skip SHA256 computation")
	skipGUID := flag.Bool("skip-guid", false, "Skip Product GUID extraction")
	dryRun := flag.Bool("dry-run", false, "Preview only")
	versionTypes := flag.String("version-types", "LTS,MTS,Stable", "Comma-separated version types")
	minMajor := flag.Int("min-major", 10, "Minimum major version")
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
		for _, v := range variants {
			url := v.urlFunc(release)

			if !urlExists(url) {
				fmt.Printf("  %s: not available\n", v.name)
				continue
			}

			var sha string
			if *skipSHA {
				sha = "SHA256_PLACEHOLDER"
			} else {
				sha, err = fetchSHA256FromCDN(url)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  %s: SHA256 fetch failed: %v\n", v.name, err)
					continue
				}
			}

			var guid string
			if *skipGUID {
				guid = GUIDPlaceholder(manifestVersion)
			} else if v.scope == "machine" {
				tempFile, dlErr := downloadToTemp(url)
				if dlErr != nil {
					fmt.Fprintf(os.Stderr, "  %s: download failed: %v\n", v.name, dlErr)
					guid = GUIDPlaceholder(manifestVersion)
				} else {
					guid, err = ExtractProductGUID(tempFile)
					if err != nil {
						fmt.Printf("  %s: GUID extraction failed: %v\n", v.name, err)
						guid = GUIDPlaceholder(manifestVersion)
					}
					os.Remove(tempFile)
				}
			} else {
				guid = GUIDPlaceholder(manifestVersion)
			}

			installers = append(installers, InstallerData{
				Arch:                 v.arch,
				Scope:                v.scope,
				URL:                  url,
				SHA256:               sha,
				GUID:                 guid,
				ElevationRequirement: v.elevationRequirement,
			})
			fmt.Printf("  %s: OK\n", v.name)
		}

		if len(installers) == 0 {
			fmt.Printf("  No installers found, skipping\n")
			continue
		}

		if *dryRun {
			fmt.Printf("  [DRY RUN] Would create: %s (%d installers)\n", versionDir, len(installers))
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

func urlExists(url string) bool {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func fetchSHA256FromCDN(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url + ".sha256")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("SHA256 file not found (HTTP %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Format: "hash *filename" or just "hash"
	parts := strings.Fields(strings.TrimSpace(string(body)))
	if len(parts) == 0 {
		return "", fmt.Errorf("empty SHA256 file")
	}

	return strings.ToUpper(parts[0]), nil
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
