package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	maxVersions := flag.Int("max-versions", 0, "Limit to first N versions (0 = all)")
	workers := flag.Int("workers", 5, "Number of parallel workers")
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

	if *maxVersions > 0 && len(releases) > *maxVersions {
		releases = releases[:*maxVersions]
	}

	fmt.Printf("Found %d releases, processing with %d workers\n", len(releases), *workers)

	var wg sync.WaitGroup
	releaseChan := make(chan Release, len(releases))
	resultChan := make(chan string, len(releases))

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for release := range releaseChan {
				result := processRelease(release, *manifestDir, *skipSHA, *skipGUID, *dryRun)
				resultChan <- result
			}
		}()
	}

	go func() {
		for _, release := range releases {
			releaseChan <- release
		}
		close(releaseChan)
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		fmt.Println(result)
	}
}

func processRelease(release Release, manifestDir string, skipSHA, skipGUID, dryRun bool) string {
	manifestVersion := manifestVersionFor(release)
	versionDir := filepath.Join(manifestDir, "Mendix", "MendixStudioPro", manifestVersion)

	if _, err := os.Stat(versionDir); err == nil {
		return fmt.Sprintf("Skipping %s (already exists)", manifestVersion)
	}

	var installers []InstallerData
	for _, v := range variants {
		url := v.urlFunc(release)

		if !urlExists(url) {
			continue
		}

		var sha string
		if skipSHA {
			sha = "SHA256_PLACEHOLDER"
		} else {
			var err error
			sha, err = fetchSHA256FromCDN(url)
			if err != nil {
				continue
			}
		}

		guid := GUIDPlaceholder(manifestVersion)
		if !skipGUID && v.scope == "machine" {
			tempFile, dlErr := downloadToTemp(url)
			if dlErr == nil {
				extracted, err := ExtractProductGUID(tempFile)
				if err == nil {
					guid = extracted
				}
				os.Remove(tempFile)
			}
		}

		installers = append(installers, InstallerData{
			Arch:                 v.arch,
			Scope:                v.scope,
			URL:                  url,
			SHA256:               sha,
			GUID:                 guid,
			ElevationRequirement: v.elevationRequirement,
		})
	}

	if len(installers) == 0 {
		return fmt.Sprintf("%s: no installers found", manifestVersion)
	}

	if dryRun {
		return fmt.Sprintf("%s: would create %d installers", manifestVersion, len(installers))
	}

	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Sprintf("%s: failed to create directory: %v", manifestVersion, err)
	}

	data := ManifestData{
		Version:    manifestVersion,
		Installers: installers,
	}

	writeManifest(filepath.Join(versionDir, "Mendix.MendixStudioPro.yaml"), packageManifestTemplate, data)
	writeManifest(filepath.Join(versionDir, "Mendix.MendixStudioPro.installer.yaml"), installerManifestTemplate, data)
	writeManifest(filepath.Join(versionDir, "Mendix.MendixStudioPro.locale.en-US.yaml"), localeManifestTemplate, data)

	return fmt.Sprintf("%s: created (%d installers)", manifestVersion, len(installers))
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
