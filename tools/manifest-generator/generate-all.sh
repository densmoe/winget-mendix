#!/usr/bin/env bash
#
# Generate manifests for all Mendix Studio Pro versions listed on the marketplace.
# Designed to run overnight — streams installers for SHA256 without writing to disk.
#
# Usage:
#   cd tools/manifest-generator
#   ./generate-all.sh
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MANIFEST_DIR="$SCRIPT_DIR/../../manifests"
LOG_FILE="$SCRIPT_DIR/generate-all.log"

cd "$SCRIPT_DIR"

echo "=== Mendix manifest generation started at $(date) ===" | tee "$LOG_FILE"

# Build once
echo "Building generator..." | tee -a "$LOG_FILE"
go build -o manifest-generator . 2>&1 | tee -a "$LOG_FILE"

# Run with 2 workers — keeps downloads sequential enough to not saturate bandwidth,
# but overlaps network I/O with hash computation. Each version downloads ~1.3GB
# (3 installer variants), so 2 workers ≈ 2.6GB in flight at any time.
echo "Starting generation (this will take several hours)..." | tee -a "$LOG_FILE"
./manifest-generator \
    -manifest-dir "$MANIFEST_DIR" \
    -min-major 9 \
    -version-types "LTS,MTS,Stable" \
    -workers 2 \
    2>&1 | tee -a "$LOG_FILE"

echo "" | tee -a "$LOG_FILE"
echo "=== Generation finished at $(date) ===" | tee -a "$LOG_FILE"

# Count results
TOTAL=$(ls -d "$MANIFEST_DIR"/Mendix/MendixStudioPro/*/ 2>/dev/null | wc -l | tr -d ' ')
echo "Total manifest versions: $TOTAL" | tee -a "$LOG_FILE"

# Commit and push
cd "$SCRIPT_DIR/../.."
if git diff --quiet HEAD -- manifests/ tools/manifest-generator/; then
    echo "No changes to commit." | tee -a "$LOG_FILE"
else
    echo "Committing and pushing..." | tee -a "$LOG_FILE"
    git add manifests/ tools/manifest-generator/guid_extractor.go tools/manifest-generator/guid_extractor_test.go tools/manifest-generator/main.go
    git commit -m "Add manifests for all Mendix Studio Pro versions

Generated $(git diff --cached --stat | grep 'files changed' || echo 'manifests') with
computed ProductCodes (SHA1 of UTF-16LE AppId) and SHA256 hashes.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
    git push origin main
    echo "Pushed to GitHub." | tee -a "$LOG_FILE"
fi

echo "=== Done at $(date) ===" | tee -a "$LOG_FILE"
