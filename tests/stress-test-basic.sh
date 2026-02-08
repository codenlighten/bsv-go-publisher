#!/bin/bash

# GovHash API Stress Test - Basic Authentication
# Tests concurrent publishing with API key authentication

set -e

# Configuration
API_URL="${API_URL:-https://api.govhash.org}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
CONCURRENT_REQUESTS="${CONCURRENT_REQUESTS:-50}"
TOTAL_REQUESTS="${TOTAL_REQUESTS:-200}"
USE_WAIT_PARAM="${USE_WAIT_PARAM:-true}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

# Results tracking
RESULTS_DIR="/tmp/govhash_stress_$(date +%s)"
mkdir -p "$RESULTS_DIR"

echo -e "${GREEN}╔═════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║       GovHash API Stress Test - Basic Auth             ║${NC}"
echo -e "${GREEN}╚═════════════════════════════════════════════════════════╝${NC}\n"

# Check for admin password
if [ -z "$ADMIN_PASSWORD" ]; then
    echo -e "${RED}Error: ADMIN_PASSWORD not set${NC}"
    echo "Usage: ADMIN_PASSWORD='your_password' $0"
    exit 1
fi

# Display test configuration
echo -e "${CYAN}Test Configuration:${NC}"
echo "  API URL:              $API_URL"
echo "  Total Requests:       $TOTAL_REQUESTS"
echo "  Concurrent Requests:  $CONCURRENT_REQUESTS"
echo "  Wait Parameter:       $USE_WAIT_PARAM"
echo "  Results Directory:    $RESULTS_DIR"
echo ""

# Step 1: Register test client
echo -e "${BLUE}Step 1: Registering test client...${NC}"

CLIENT_NAME="Stress Test Client $(date +%s)"
REGISTER_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/admin/clients/register" \
    -H "X-Admin-Password: ${ADMIN_PASSWORD}" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"${CLIENT_NAME}\",
        \"max_daily_tx\": 100000,
        \"tier\": \"pilot\"
    }")

HTTP_CODE=$(echo "$REGISTER_RESPONSE" | tail -n1)
BODY=$(echo "$REGISTER_RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    echo -e "${GREEN}✓${NC} Client registered successfully"
    CLIENT_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    API_KEY=$(echo "$BODY" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4)
    echo -e "${YELLOW}Client ID:${NC} $CLIENT_ID"
    echo -e "${YELLOW}API Key:${NC} $API_KEY"
    echo ""
else
    echo -e "${RED}✗ Registration failed with HTTP $HTTP_CODE${NC}"
    echo "$BODY"
    exit 1
fi

# Save API key for reuse
echo "$API_KEY" > "$RESULTS_DIR/api_key.txt"

# Step 2: Check initial system health
echo -e "${BLUE}Step 2: Checking initial system health...${NC}"

HEALTH_RESPONSE=$(curl -s "${API_URL}/health")
echo "$HEALTH_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$HEALTH_RESPONSE"

INITIAL_QUEUE=$(echo "$HEALTH_RESPONSE" | grep -o '"queueDepth":[0-9]*' | cut -d':' -f2)
INITIAL_UTXOS=$(echo "$HEALTH_RESPONSE" | grep -o '"publishing_available":[0-9]*' | cut -d':' -f2)

echo ""
echo -e "${YELLOW}Initial State:${NC}"
echo "  Queue Depth:      $INITIAL_QUEUE"
echo "  Available UTXOs:  $INITIAL_UTXOS"
echo ""

# Step 3: Define publish function
publish_transaction() {
    local request_id=$1
    local api_key=$2
    local use_wait=$3
    local start_time=$(date +%s%3N)
    
    # Generate unique data
    local data_text="Stress test request #${request_id} at $(date +%s%N)"
    local data_hex=$(echo -n "$data_text" | xxd -p | tr -d '\n')
    
    # Construct URL with wait parameter
    local url="${API_URL}/publish"
    if [ "$use_wait" = "true" ]; then
        url="${url}?wait=true"
    fi
    
    # Make request
    local response=$(curl -s -w "\n%{http_code}\n%{time_total}" -X POST "$url" \
        -H "X-API-Key: ${api_key}" \
        -H "Content-Type: application/json" \
        -d "{\"data\":\"${data_hex}\"}" \
        --max-time 30)
    
    local http_code=$(echo "$response" | tail -n2 | head -n1)
    local time_total=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n-2)
    
    local end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))
    
    # Extract TXID or UUID
    local txid=$(echo "$body" | grep -o '"txid":"[^"]*"' | cut -d'"' -f4)
    local uuid=$(echo "$body" | grep -o '"uuid":"[^"]*"' | cut -d'"' -f4)
    local arc_status=$(echo "$body" | grep -o '"arc_status":"[^"]*"' | cut -d'"' -f4)
    
    # Log result
    echo "$request_id,$http_code,$duration,$time_total,$txid,$uuid,$arc_status" >> "$RESULTS_DIR/results.csv"
    
    # Print status
    if [ "$http_code" = "201" ] && [ -n "$txid" ]; then
        echo -e "${GREEN}✓${NC} Request #${request_id}: ${GREEN}SUCCESS${NC} (${duration}ms, TXID: ${txid:0:16}...)"
    elif [ "$http_code" = "202" ] || [ "$http_code" = "201" ]; then
        echo -e "${YELLOW}⊙${NC} Request #${request_id}: ${YELLOW}QUEUED${NC} (${duration}ms, UUID: ${uuid:0:16}...)"
    else
        echo -e "${RED}✗${NC} Request #${request_id}: ${RED}FAILED${NC} (HTTP $http_code)"
    fi
}

export -f publish_transaction
export API_URL
export RESULTS_DIR

# Step 4: Initialize results file
echo "request_id,http_code,duration_ms,curl_time,txid,uuid,arc_status" > "$RESULTS_DIR/results.csv"

# Step 5: Run stress test
echo -e "${BLUE}Step 3: Starting stress test...${NC}"
echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

TEST_START_TIME=$(date +%s)

# Run concurrent requests using GNU parallel or xargs
if command -v parallel &> /dev/null; then
    echo "Using GNU parallel for concurrency..."
    seq 1 $TOTAL_REQUESTS | parallel -j $CONCURRENT_REQUESTS --bar \
        "publish_transaction {} $API_KEY $USE_WAIT_PARAM"
else
    echo "Using xargs for concurrency (install 'parallel' for better performance)..."
    seq 1 $TOTAL_REQUESTS | xargs -P $CONCURRENT_REQUESTS -I {} bash -c \
        "publish_transaction {} $API_KEY $USE_WAIT_PARAM"
fi

TEST_END_TIME=$(date +%s)
TOTAL_DURATION=$((TEST_END_TIME - TEST_START_TIME))

echo ""
echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

# Step 6: Analyze results
echo -e "${BLUE}Step 4: Analyzing results...${NC}\n"

# Count successes and failures
TOTAL_SENT=$(wc -l < "$RESULTS_DIR/results.csv")
TOTAL_SENT=$((TOTAL_SENT - 1))  # Subtract header line

SUCCESS_COUNT=$(awk -F',' '$2 == 201 && $5 != "" { count++ } END { print count+0 }' "$RESULTS_DIR/results.csv")
QUEUED_COUNT=$(awk -F',' '($2 == 201 || $2 == 202) && $5 == "" && $6 != "" { count++ } END { print count+0 }' "$RESULTS_DIR/results.csv")
FAILED_COUNT=$(awk -F',' '$2 != 201 && $2 != 202 { count++ } END { print count+0 }' "$RESULTS_DIR/results.csv")

# Calculate latency statistics
AVG_LATENCY=$(awk -F',' 'NR > 1 && $3 > 0 { sum += $3; count++ } END { if (count > 0) print int(sum/count); else print 0 }' "$RESULTS_DIR/results.csv")
MIN_LATENCY=$(awk -F',' 'NR > 1 && $3 > 0 { if (min == "" || $3 < min) min = $3 } END { print min+0 }' "$RESULTS_DIR/results.csv")
MAX_LATENCY=$(awk -F',' 'NR > 1 && $3 > 0 { if ($3 > max) max = $3 } END { print max+0 }' "$RESULTS_DIR/results.csv")

# Calculate percentiles (approximate)
P50_LATENCY=$(awk -F',' 'NR > 1 && $3 > 0 { print $3 }' "$RESULTS_DIR/results.csv" | sort -n | awk '{a[NR]=$1} END {print a[int(NR*0.5)]+0}')
P95_LATENCY=$(awk -F',' 'NR > 1 && $3 > 0 { print $3 }' "$RESULTS_DIR/results.csv" | sort -n | awk '{a[NR]=$1} END {print a[int(NR*0.95)]+0}')
P99_LATENCY=$(awk -F',' 'NR > 1 && $3 > 0 { print $3 }' "$RESULTS_DIR/results.csv" | sort -n | awk '{a[NR]=$1} END {print a[int(NR*0.99)]+0}')

# Calculate throughput
THROUGHPUT=$(echo "scale=2; $TOTAL_SENT / $TOTAL_DURATION" | bc)

# Step 7: Check final system health
FINAL_HEALTH=$(curl -s "${API_URL}/health")
FINAL_QUEUE=$(echo "$FINAL_HEALTH" | grep -o '"queueDepth":[0-9]*' | cut -d':' -f2)
FINAL_UTXOS=$(echo "$FINAL_HEALTH" | grep -o '"publishing_available":[0-9]*' | cut -d':' -f2)

# Display results
echo -e "${GREEN}╔═════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                    STRESS TEST RESULTS                  ║${NC}"
echo -e "${GREEN}╚═════════════════════════════════════════════════════════╝${NC}\n"

echo -e "${CYAN}Request Summary:${NC}"
echo "  Total Sent:           $TOTAL_SENT"
echo -e "  Instant Success:      ${GREEN}$SUCCESS_COUNT${NC} ($(echo "scale=1; $SUCCESS_COUNT * 100 / $TOTAL_SENT" | bc)%)"
echo -e "  Queued (Async):       ${YELLOW}$QUEUED_COUNT${NC} ($(echo "scale=1; $QUEUED_COUNT * 100 / $TOTAL_SENT" | bc)%)"
echo -e "  Failed:               ${RED}$FAILED_COUNT${NC} ($(echo "scale=1; $FAILED_COUNT * 100 / $TOTAL_SENT" | bc)%)"
echo ""

echo -e "${CYAN}Performance Metrics:${NC}"
echo "  Total Duration:       ${TOTAL_DURATION}s"
echo "  Throughput:           ${THROUGHPUT} tx/sec"
echo "  Concurrency Level:    ${CONCURRENT_REQUESTS}"
echo ""

echo -e "${CYAN}Latency Distribution (ms):${NC}"
echo "  Average (Mean):       ${AVG_LATENCY} ms"
echo "  Minimum:              ${MIN_LATENCY} ms"
echo "  Maximum:              ${MAX_LATENCY} ms"
echo "  Median (p50):         ${P50_LATENCY} ms"
echo "  95th Percentile:      ${P95_LATENCY} ms"
echo "  99th Percentile:      ${P99_LATENCY} ms"
echo ""

echo -e "${CYAN}System State:${NC}"
echo "  Initial Queue:        $INITIAL_QUEUE"
echo "  Final Queue:          $FINAL_QUEUE (Δ $((FINAL_QUEUE - INITIAL_QUEUE)))"
echo "  Initial UTXOs:        $INITIAL_UTXOS"
echo "  Final UTXOs:          $FINAL_UTXOS (Used: $((INITIAL_UTXOS - FINAL_UTXOS)))"
echo ""

# Generate summary report
cat > "$RESULTS_DIR/summary.txt" << EOF
GovHash Stress Test Summary
Generated: $(date)

Configuration:
- API URL: $API_URL
- Client: $CLIENT_NAME
- Total Requests: $TOTAL_SENT
- Concurrent Requests: $CONCURRENT_REQUESTS
- Wait Parameter: $USE_WAIT_PARAM

Results:
- Instant Success: $SUCCESS_COUNT ($(echo "scale=1; $SUCCESS_COUNT * 100 / $TOTAL_SENT" | bc)%)
- Queued (Async): $QUEUED_COUNT ($(echo "scale=1; $QUEUED_COUNT * 100 / $TOTAL_SENT" | bc)%)
- Failed: $FAILED_COUNT ($(echo "scale=1; $FAILED_COUNT * 100 / $TOTAL_SENT" | bc)%)

Performance:
- Total Duration: ${TOTAL_DURATION}s
- Throughput: ${THROUGHPUT} tx/sec
- Average Latency: ${AVG_LATENCY} ms
- P95 Latency: ${P95_LATENCY} ms
- P99 Latency: ${P99_LATENCY} ms

System Impact:
- Queue Depth Change: $INITIAL_QUEUE → $FINAL_QUEUE
- UTXOs Consumed: $((INITIAL_UTXOS - FINAL_UTXOS))
EOF

echo -e "${YELLOW}Results saved to:${NC} $RESULTS_DIR"
echo "  - results.csv        (Raw data)"
echo "  - summary.txt        (Summary report)"
echo "  - api_key.txt        (Test client API key)"
echo ""

# Optional: Wait for queue to clear and check TXIDs
if [ "$QUEUED_COUNT" -gt 0 ]; then
    echo -e "${BLUE}Step 5: Checking queued transactions...${NC}"
    echo "Waiting 10 seconds for queue to process..."
    sleep 10
    
    QUEUED_UUIDS=$(awk -F',' '$6 != "" && $5 == "" { print $6 }' "$RESULTS_DIR/results.csv" | head -5)
    
    echo "Sampling 5 queued transactions:"
    for uuid in $QUEUED_UUIDS; do
        STATUS_RESP=$(curl -s "${API_URL}/status/${uuid}")
        TXID=$(echo "$STATUS_RESP" | grep -o '"txid":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$TXID" ]; then
            echo -e "  ${GREEN}✓${NC} UUID ${uuid:0:8}... → TXID ${TXID:0:16}..."
        else
            echo -e "  ${YELLOW}⊙${NC} UUID ${uuid:0:8}... → Still processing"
        fi
    done
    echo ""
fi

echo -e "${GREEN}╔═════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║              STRESS TEST COMPLETE ✓                     ║${NC}"
echo -e "${GREEN}╚═════════════════════════════════════════════════════════╝${NC}"

# Exit with success if < 5% failed
FAIL_RATE=$(echo "scale=0; $FAILED_COUNT * 100 / $TOTAL_SENT" | bc)
if [ "$FAIL_RATE" -lt 5 ]; then
    exit 0
else
    echo -e "${RED}Warning: Failure rate ${FAIL_RATE}% exceeds 5% threshold${NC}"
    exit 1
fi
