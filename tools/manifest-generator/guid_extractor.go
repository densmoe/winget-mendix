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

	cmd := exec.Command("7z", "x", exePath, fmt.Sprintf("-o%s", tempDir), "-y")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("7z extract failed: %w", err)
	}

	msiFiles, err := filepath.Glob(filepath.Join(tempDir, "*.msi"))
	if err != nil || len(msiFiles) == 0 {
		msiFiles, _ = filepath.Glob(filepath.Join(tempDir, "**", "*.msi"))
	}

	if len(msiFiles) == 0 {
		return "", fmt.Errorf("no MSI found in installer")
	}

	cmd = exec.Command("7z", "l", msiFiles[0])
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("7z list MSI failed: %w", err)
	}

	matches := guidPattern.FindAllString(string(output), -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no GUID found in MSI")
	}

	return strings.ToUpper(matches[0]), nil
}

func GUIDPlaceholder(version string) string {
	return fmt.Sprintf("{MENDIX-STUDIO-PRO-%s-PLACEHOLDER}", strings.ReplaceAll(version, ".", "-"))
}
