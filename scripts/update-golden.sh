#!/bin/bash
# Update golden test output files.
# Run this after changing templates or test fixtures.
set -e

WRITE_GOLDEN_OUTPUT=1 go test ./internal/backend/ -run TestGolden -count=1 -v
