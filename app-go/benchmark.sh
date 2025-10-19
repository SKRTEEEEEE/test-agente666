#!/bin/bash

# Performance Benchmark Script for app-go
# Tests cache performance and gzip compression

echo "🚀 app-go Performance Benchmark"
echo "================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if server is running
echo -n "Checking if server is running... "
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${RED}FAILED${NC}"
    echo "Please start the server first:"
    echo "  docker run -d -p 8080:8080 --name hello-world-go hello-world-go:latest"
    exit 1
fi
echo -e "${GREEN}OK${NC}"
echo ""

# Test 1: Cache Performance (Cold vs Warm)
echo -e "${BLUE}Test 1: Cache Performance${NC}"
echo "------------------------"

USER="SKRTEEEEEE"
ENDPOINT="http://localhost:8080/issues/$USER"

echo -n "Cold request (no cache): "
COLD_TIME=$(curl -w "%{time_total}" -s -o /dev/null "$ENDPOINT")
echo -e "${YELLOW}${COLD_TIME}s${NC}"

sleep 1

echo -n "Warm request (cached):   "
WARM_TIME=$(curl -w "%{time_total}" -s -o /dev/null "$ENDPOINT")
echo -e "${GREEN}${WARM_TIME}s${NC}"

# Calculate speedup
SPEEDUP=$(echo "scale=2; $COLD_TIME / $WARM_TIME" | bc)
echo -e "Speedup: ${GREEN}${SPEEDUP}x faster${NC}"
echo ""

# Test 2: Gzip Compression
echo -e "${BLUE}Test 2: Gzip Compression${NC}"
echo "------------------------"

echo -n "Without gzip: "
SIZE_UNCOMPRESSED=$(curl -s "$ENDPOINT" | wc -c)
echo -e "${YELLOW}${SIZE_UNCOMPRESSED} bytes${NC}"

echo -n "With gzip:    "
SIZE_COMPRESSED=$(curl -s -H "Accept-Encoding: gzip" "$ENDPOINT" --compressed | wc -c)
echo -e "${GREEN}${SIZE_COMPRESSED} bytes${NC}"

# Calculate savings
SAVINGS=$(echo "scale=2; 100 - ($SIZE_COMPRESSED * 100 / $SIZE_UNCOMPRESSED)" | bc)
echo -e "Savings: ${GREEN}${SAVINGS}% reduction${NC}"
echo ""

# Test 3: Multiple Requests (Cache Hit Rate)
echo -e "${BLUE}Test 3: Cache Hit Rate (10 requests)${NC}"
echo "-------------------------------------"

TOTAL_TIME=0
for i in {1..10}; do
    TIME=$(curl -w "%{time_total}" -s -o /dev/null "$ENDPOINT")
    TOTAL_TIME=$(echo "$TOTAL_TIME + $TIME" | bc)
    printf "Request %2d: %ss\n" $i $TIME
done

AVG_TIME=$(echo "scale=4; $TOTAL_TIME / 10" | bc)
echo -e "Average: ${GREEN}${AVG_TIME}s${NC}"
echo ""

# Test 4: Health Check Performance
echo -e "${BLUE}Test 4: Health Check Performance${NC}"
echo "---------------------------------"

HEALTH_TOTAL=0
for i in {1..5}; do
    TIME=$(curl -w "%{time_total}" -s -o /dev/null "http://localhost:8080/health")
    HEALTH_TOTAL=$(echo "$HEALTH_TOTAL + $TIME" | bc)
done

HEALTH_AVG=$(echo "scale=4; $HEALTH_TOTAL / 5" | bc)
echo -e "Average health check time: ${GREEN}${HEALTH_AVG}s${NC}"
echo ""

# Summary
echo "================================"
echo -e "${GREEN}✓ Benchmark Complete!${NC}"
echo ""
echo "Key Findings:"
echo "  • Cache provides ${SPEEDUP}x speedup"
echo "  • Gzip reduces response size by ${SAVINGS}%"
echo "  • Average cached response time: ${AVG_TIME}s"
echo "  • Health check latency: ${HEALTH_AVG}s"
echo ""
echo "For detailed performance documentation, see PERFORMANCE.md"
