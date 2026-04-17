#!/bin/bash
# Extract GUIDs for all existing manifests with parallel processing
# Requires: 7z (brew install p7zip)

set -e

MANIFEST_DIR="${1:-../manifests/Mendix/MendixStudioPro}"
WORKERS="${2:-5}"  # Number of parallel workers

echo "Extracting Product GUIDs for all manifests (${WORKERS} parallel workers)..."
echo "This will download machine installers (~600MB each) to extract GUIDs."
echo ""

if ! command -v 7z &> /dev/null; then
    echo "Error: 7z not found. Install with: brew install p7zip"
    exit 1
fi

extract_guid() {
    local version_dir="$1"
    local version=$(basename "$version_dir")
    local installer_yaml="$version_dir/Mendix.MendixStudioPro.installer.yaml"

    if [ ! -f "$installer_yaml" ]; then
        echo "[$version] No installer manifest found, skipping"
        return
    fi

    # Check if already has real GUID
    if ! grep -q "PLACEHOLDER" "$installer_yaml"; then
        echo "[$version] Already has real GUID, skipping"
        return
    fi

    # Extract machine installer URL
    local url=$(awk '/Scope: machine/,/InstallerUrl:/ {if (/InstallerUrl:/) print $2}' "$installer_yaml" | head -1)

    if [ -z "$url" ]; then
        echo "[$version] Could not find machine installer URL, skipping"
        return
    fi

    echo "[$version] Downloading $url..."

    local temp_file=$(mktemp "/tmp/mendix-${version}-XXXXXX.exe")
    local extract_dir=$(mktemp -d "/tmp/mendix-extract-${version}-XXXXXX")

    if ! curl -sL "$url" -o "$temp_file"; then
        echo "[$version] Download failed"
        rm -f "$temp_file"
        rm -rf "$extract_dir"
        return
    fi

    # Extract with 7z
    if ! 7z x "$temp_file" -o"$extract_dir" -y > /dev/null 2>&1; then
        echo "[$version] Extraction failed"
        rm -f "$temp_file"
        rm -rf "$extract_dir"
        return
    fi

    # Find MSI
    local msi_file=$(find "$extract_dir" -name "*.msi" -type f | head -1)

    if [ -z "$msi_file" ]; then
        echo "[$version] No MSI found in installer"
        rm -f "$temp_file"
        rm -rf "$extract_dir"
        return
    fi

    # Extract GUID from MSI
    local guid=$(7z l "$msi_file" | grep -Eo '\{[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}\}' | head -1 | tr '[:lower:]' '[:upper:]')

    if [ -n "$guid" ]; then
        echo "[$version] Found GUID: $guid"

        # Replace placeholder in YAML
        local placeholder="{MENDIX-STUDIO-PRO-${version//./-}-PLACEHOLDER}"

        # Use a lock file to prevent concurrent writes
        (
            flock -x 200
            sed -i.bak "s/$placeholder/$guid/g" "$installer_yaml"
            rm -f "$installer_yaml.bak"
        ) 200>/tmp/guid-lock-"$version"

        echo "[$version] Updated manifest"
    else
        echo "[$version] Could not extract GUID from MSI"
    fi

    # Cleanup
    rm -f "$temp_file"
    rm -rf "$extract_dir"
    rm -f /tmp/guid-lock-"$version"
}

export -f extract_guid

# Process all versions in parallel
find "$MANIFEST_DIR" -maxdepth 1 -type d | tail -n +2 | xargs -P "$WORKERS" -I {} bash -c 'extract_guid "$@"' _ {}

echo ""
echo "Done! Commit and push the updated manifests."
