#!/bin/bash
#
# Performance Regression Testing Script
#
# This script runs benchmarks and compares them against a baseline to detect
# performance regressions. It fails the build if performance degrades > 10%.
#
# Usage:
#   ./scripts/perf-regression-test.sh [baseline-file]
#
# If no baseline file is provided, it creates one first.

set -e

# Configuration
BASELINE_FILE="${1:-perf-baseline.txt}"
CURRENT_FILE="perf-current.txt"
REPORT_FILE="perf-regression-report.txt"
REGRESSION_THRESHOLD=10  # Percentage degradation allowed
BENCH_COUNT=5  # Number of benchmark runs for statistical significance

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Performance Regression Testing"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo

# Check if benchstat is installed
if ! command -v benchstat &> /dev/null; then
    echo -e "${YELLOW}⚠${NC}  benchstat not found. Installing..."
    go install golang.org/x/perf/cmd/benchstat@latest
fi

# Check if baseline exists
if [ ! -f "$BASELINE_FILE" ]; then
    echo -e "${YELLOW}⚠${NC}  No baseline found at $BASELINE_FILE"
    echo "   Creating baseline..."
    echo

    echo "Running baseline benchmarks ($BENCH_COUNT iterations)..."
    go test -bench=. -benchmem -count=$BENCH_COUNT ./parser ./decoder ./encoder ./validator > "$BASELINE_FILE"

    echo -e "${GREEN}✓${NC} Baseline created: $BASELINE_FILE"
    echo
    echo "Run this script again to compare against baseline."
    exit 0
fi

echo "📊 Running current benchmarks ($BENCH_COUNT iterations)..."
echo "   This may take a minute..."
go test -bench=. -benchmem -count=$BENCH_COUNT ./parser ./decoder ./encoder ./validator > "$CURRENT_FILE"
echo -e "${GREEN}✓${NC} Benchmarks complete"
echo

echo "📈 Comparing with baseline..."
echo
benchstat "$BASELINE_FILE" "$CURRENT_FILE" | tee "$REPORT_FILE"
echo

# Parse benchstat output for regressions
# benchstat shows changes like "+10.5%" or "-5.2%"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Regression Analysis"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo

REGRESSIONS_FOUND=0

# Extract time/op changes from benchstat output
# Look for lines with percentage changes in the "delta" column
while IFS= read -r line; do
    # Skip header lines and separator lines
    if [[ "$line" =~ ^name || "$line" =~ ^-+ || -z "$line" ]]; then
        continue
    fi

    # Look for time/op lines with delta column
    if [[ "$line" =~ time/op ]]; then
        # Extract the delta percentage (e.g., "+15.2%" or "-5.1%")
        delta=$(echo "$line" | awk '{print $NF}')

        # Check if it's a significant regression (positive percentage > threshold)
        if [[ "$delta" =~ ^\+([0-9]+\.[0-9]+)% ]]; then
            percentage="${BASH_REMATCH[1]}"
            benchmark=$(echo "$line" | awk '{print $1}')

            # Use bc for floating point comparison
            if (( $(echo "$percentage > $REGRESSION_THRESHOLD" | bc -l) )); then
                echo -e "${RED}✗ REGRESSION:${NC} $benchmark slowdown: +$percentage%"
                REGRESSIONS_FOUND=$((REGRESSIONS_FOUND + 1))
            fi
        fi
    fi

    # Look for alloc/op lines with delta column (memory regressions)
    if [[ "$line" =~ alloc/op ]]; then
        delta=$(echo "$line" | awk '{print $NF}')

        if [[ "$delta" =~ ^\+([0-9]+\.[0-9]+)% ]]; then
            percentage="${BASH_REMATCH[1]}"
            benchmark=$(echo "$line" | awk '{print $1}')

            # Allow more tolerance for memory (20%)
            if (( $(echo "$percentage > 20.0" | bc -l) )); then
                echo -e "${YELLOW}⚠  MEMORY:${NC} $benchmark memory increase: +$percentage%"
            fi
        fi
    fi
done < "$REPORT_FILE"

echo

if [ $REGRESSIONS_FOUND -eq 0 ]; then
    echo -e "${GREEN}✓ No performance regressions detected!${NC}"
    echo
    echo "Summary:"
    echo "  Threshold: ${REGRESSION_THRESHOLD}% slowdown"
    echo "  Baseline: $BASELINE_FILE"
    echo "  Current:  $CURRENT_FILE"
    echo "  Report:   $REPORT_FILE"
    echo
    exit 0
else
    echo -e "${RED}✗ Found $REGRESSIONS_FOUND performance regression(s)${NC}"
    echo
    echo "Performance has degraded by more than ${REGRESSION_THRESHOLD}%."
    echo
    echo "Next steps:"
    echo "  1. Review the regression report: $REPORT_FILE"
    echo "  2. Profile the slow benchmarks:"
    echo "     go test -bench=<benchmark> -cpuprofile=cpu.prof ./..."
    echo "     go tool pprof cpu.prof"
    echo "  3. Fix the regression or update baseline if intentional:"
    echo "     cp $CURRENT_FILE $BASELINE_FILE"
    echo
    exit 1
fi
