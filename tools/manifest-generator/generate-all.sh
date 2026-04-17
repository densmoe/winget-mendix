#!/usr/bin/env bash
#
# Generate manifests for all Mendix Studio Pro versions listed on the marketplace.
# Resumable — skips versions that already have complete manifests (all 3 files,
# real SHA256 hashes, real ProductCodes). Safe to kill and restart.
#
# Usage:
#   cd tools/manifest-generator
#   nohup ./generate-all.sh &
#
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MANIFEST_DIR="$SCRIPT_DIR/../../manifests"
REPO_DIR="$SCRIPT_DIR/../.."
LOG_FILE="$SCRIPT_DIR/generate-all.log"

log() { echo "[$(date '+%H:%M:%S')] $*" | tee -a "$LOG_FILE"; }

cd "$SCRIPT_DIR"
log "=== Started ==="

# Build
log "Building..."
if ! go build -o manifest-generator . 2>&1 | tee -a "$LOG_FILE"; then
    log "FATAL: build failed"
    exit 1
fi

# Run generator — processes one version at a time so we can commit in batches.
# The Go tool skips complete manifests and regenerates incomplete ones.
log "Generating manifests (single worker for reliability)..."
./manifest-generator \
    -manifest-dir "$MANIFEST_DIR" \
    -min-major 9 \
    -version-types "LTS,MTS,Stable" \
    -workers 1 \
    2>&1 | while IFS= read -r line; do
    log "$line"

    # Commit every 10 created versions to save progress
    if echo "$line" | grep -q "created"; then
        CREATED=$((${CREATED:-0} + 1))
        if (( CREATED % 10 == 0 )); then
            log "Checkpoint: committing $CREATED versions so far..."
            cd "$REPO_DIR"
            git add manifests/ tools/manifest-generator/main.go tools/manifest-generator/guid_extractor.go tools/manifest-generator/guid_extractor_test.go 2>/dev/null
            git diff --cached --quiet || git commit -m "Add Mendix Studio Pro manifests (batch $((CREATED / 10)))

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
            cd "$SCRIPT_DIR"
        fi
    fi
done

# Final commit and push
log "Generation complete. Committing remaining changes..."
cd "$REPO_DIR"
git add manifests/ tools/manifest-generator/main.go tools/manifest-generator/guid_extractor.go tools/manifest-generator/guid_extractor_test.go 2>/dev/null
if git diff --cached --quiet 2>/dev/null; then
    log "No new changes to commit."
else
    TOTAL=$(ls -d "$MANIFEST_DIR"/Mendix/MendixStudioPro/*/ 2>/dev/null | wc -l | tr -d ' ')
    git commit -m "Add Mendix Studio Pro manifests (final, $TOTAL total versions)

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
fi

log "Pushing to GitHub..."
git push origin main 2>&1 | tee -a "$LOG_FILE"

log "=== Done ==="
