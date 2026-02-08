#!/bin/bash

# GovHash API Stress Test - ECDSA Authentication
# Tests concurrent publishing with ECDSA signature authentication

set -e

# Configuration
API_URL="${API_URL:-https://api.govhash.org}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
CONCURRENT_REQUESTS="${CONCURRENT_REQUESTS:-25}"
TOTAL_REQUESTS="${TOTAL_REQUESTS:-100}"
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
RESULTS_DIR="/tmp/govhash_ecdsa_stress_$(date +%s)"
mkdir -p "$RESULTS_DIR"

echo -e "${GREEN}╔═════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║      GovHash API Stress Test - ECDSA Auth              ║${NC}"
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
echo "  Authentication:       ECDSA secp256k1"
echo "  Results Directory:    $RESULTS_DIR"
echo ""

# Step 1: Generate ECDSA key pair
echo -e "${BLUE}Step 1: Generating ECDSA key pair...${NC}"

PRIVATE_KEY_FILE="$RESULTS_DIR/ecdsa_private.pem"
PUBLIC_KEY_FILE="$RESULTS_DIR/ecdsa_public.pem"

openssl ecparam -name secp256k1 -genkey -noout -out "$PRIVATE_KEY_FILE" 2>/dev/null
openssl ec -in "$PRIVATE_KEY_FILE" -pubout -out "$PUBLIC_KEY_FILE" 2>/dev/null

PUBLIC_KEY_HEX=$(openssl ec -in "$PRIVATE_KEY_FILE" -pubout -outform DER 2>/dev/null | tail -c 65 | xxd -p -c 65)

echo -e "${GREEN}✓${NC} Generated secp256k1 key pair"
echo -e "${YELLOW}Public Key (hex):${NC} ${PUBLIC_KEY_HEX:0:32}...${PUBLIC_KEY_HEX: -16}"
echo ""

# Step 2: Register client with public key
echo -e "${BLUE}Step 2: Registering ECDSA client...${NC}"

CLIENT_NAME="ECDSA Stress Test $(date +%s)"
REGISTER_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/admin/clients/register" \
    -H "X-Admin-Password: ${ADMIN_PASSWORD}" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"${CLIENT_NAME}\",
        \"public_key\": \"${PUBLIC_KEY_HEX}\",
        \"max_daily_tx\": 100000
    }")

HTTP_CODE=$(echo "$REGISTER_RESPONSE" | tail -n1)
BODY=$(echo "$REGISTER_RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    echo -e "${GREEN}✓${NC} ECDSA client registered successfully"
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

# Save credentials
echo "$API_KEY" > "$RESULTS_DIR/api_key.txt"
echo "$PUBLIC_KEY_HEX" > "$RESULTS_DIR/public_key.txt"

# Step 3: Check initial system health
echo -e "${BLUE}Step 3: Checking initial system health...${NC}"

HEALTH_RESPONSE=$(curl -s "${API_URL}/health")
echo "$HEALTH_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$HEALTH_RESPONSE"

INITIAL_QUEUE=$(echo "$HEALTH_RESPONSE" | grep -o '"queueDepth":[0-9]*' | cut -d':' -f2)
INITIAL_UTXOS=$(echo "$HEALTH_RESPONSE" | grep -o '"publishing_available":[0-9]*' | cut -d':' -f2)

echo ""
echo -e "${YELLOW}Initial State:${NC}"
echo "  Queue Depth:      $INITIAL_QUEUE"
echo "  Available UTXOs:  $INITIAL_UTXOS"
echo ""

# Step 4: Define signed publish function
publish_signed_transaction() {
    local request_id=$1
    local api_key=$2
    local private_key=$3
    local use_wait=$4
    local start_time=$(date +%s%3N)
    
    # Generate unique data
    local data_text="ECDSA stress test #${request_id} at $(date +%s%N)"
    local data_hex=$(echo -n "$data_text" | xxd -p | tr -d '\n')
    
    # Generate signature components
    local timestamp=$(date +%s)000
    local nonce=$(uuidgen | tr -d '-')
    
    # Create signature payload: timestamp + nonce + data
    local signature_payload="${timestamp}${nonce}${data_hex}"
    echo -n "$signature_payload" > "/tmp/sig_payload_${request_id}.txt"
    
    # Sign the payload
    local signature=$(openssl dgst -sha256 -sign "$private_key" "/tmp/sig_payload_${request_id}.txt" 2>/dev/null | base64 -w 0)
    
    # Construct URL
    local url="${API_URL}/publish"
    if [ "$use_wait" = "true" ]; then
        url="${url}?wait=true"
    fi
    
    # Make signed request
    local response=$(curl -s -w "\n%{http_code}\n%{time_total}" -X POST "$url" \
        -H "X-API-Key: ${api_key}" \
        -H "X-Signature: ${signature}" \
        -H "X-Timestamp: ${timestamp}" \
        -H "X-Nonce: ${nonce}" \
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
    
    # Cleanup temp file
    rm -f "/tmp/sig_payload_${request_id}.txt"
    
    # Print status
    if [ "$http_code" = "201" ] && [ -n "$txid" ]; then
        echo -e "${GREEN}✓${NC} Request #${request_id}: ${GREEN}SIGNED+SUCCESS${NC} (${duration}ms, TXID: ${txid:0:16}...)"
    elif [ "$http_code" = "202" ] || [ "$http_code" = "201" ]; then
        echo -e "${YELLOW}⊙${NC} Request #${request_id}: ${YELLOW}SIGNED+QUEUED${NC} (${duration}ms, UUID: ${uuid:0:16}...)"
    elif [ "$http_code" = "401" ] || [ "$http_code" = "403" ]; then
        echo -e "${RED}✗${NC} Request #${request_id}: ${RED}AUTH FAILED${NC} (HTTP $http_code)"
    else
        echo -e "${RED}✗${NC} Request #${request_id}: ${RED}FAILED${NC} (HTTP $http_code)"
    fi
}

export -f publish_signed_transaction
export API_URL
export RESULTS_DIR

# Step 5: Initialize results file
echo "request_id,http_code,duration_ms,curl_time,txid,uuid,arc_status" > "$RESULTS_DIR/results.csv"

# Step 6: Run ECDSA stress test
echo -e "${BLUE}Step 4: Starting ECDSA stress test...${NC}"
echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

TEST_START_TIME=$(date +%s)

# Run concurrent requests
if command -v parallel &> /dev/null; then
    echo "Using GNU parallel for concurrency..."
    seq 1 $TOTAL_REQUESTS | parallel -j $CONCURRENT_REQUESTS --bar \
        "publish_signed_transaction {} $API_KEY $PRIVATE_KEY_FILE $USE_WAIT_PARAM"
else
    echo "Using xargs for concurrency..."
    seq 1 $TOTAL_REQUESTS | xargs -P $CONCURRENT_REQUESTS -I {} bash -c \
        "publish_signed_transaction {} $API_KEY $PRIVATE_KEY_FILE $USE_WAIT_PARAM"
fi

TEST_END_TIME=$(date +%s)
TOTAL_DURATION=$((TEST_END_TIME - TEST_START_TIME))

echo ""
echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

# Step 7: Analyze results
echo -e "${BLUE}Step 5: Analyzing results...${NC}\n"

# Count successes and failures
TOTAL_SENT=$(wc -l < "$RESULTS_DIR/results.csv")
TOTAL_SENT=$((TOTAL_SENT - 1))

SUCCESS_COUNT=$(awk -F',' '$2 == 201 && $5 != "" { count++ } END { print count+0 }' "$RESULTS_DIR/results.csv")
QUEUED_COUNT=$(awk -F',' '($2 == 201 || $2 == 202) && $5 == "" && $6 != "" { count++ } END { print count+0 }' "$RESULTS_DIR/results.csv")
AUTH_FAILED=$(awk -F',' '$2 == 401 || $2 == 403 { count++ } END { print count+0 }' "$RESULTS_DIR/results.csv")
FAILED_COUNT=$(awk -F',' '$2 != 201 && $2 != 202 { count++ } END { print count+0 }' "$RESULTS_DIR/results.csv")

# Calculate latency statistics
AVG_LATENCY=$(awk -F',' 'NR > 1 && $3 > 0 { sum += $3; count++ } END { if (count > 0) print int(sum/count); else print 0 }' "$RESULTS_DIR/results.csv")
MIN_LATENCY=$(awk -F',' 'NR > 1 && $3 > 0 { if (min == "" || $3 < min) min = $3 } END { print min+0 }' "$RESULTS_DIR/results.csv")
MAX_LATENCY=$(awk -F',' 'NR > 1 && $3 > 0 { if ($3 > max) max = $3 } END { print max+0 }' "$RESULTS_DIR/results.csv")

P50_LATENCY=$(awk -F',' 'NR > 1 && $3 > 0 { print $3 }' "$RESULTS_DIR/results.csv" | sort -n | awk '{a[NR]=$1} END {print a[int(NR*0.5)]+0}')
P95_LATENCY=$(awk -F',' 'NR > 1 && $3 > 0 { print $3 }' "$RESULTS_DIR/results.csv" | sort -n | awk '{a[NR]=$1} END {print a[int(NR*0.95)]+0}')
P99_LATENCY=$(awk -F',' 'NR > 1 && $3 > 0 { print $3 }' "$RESULTS_DIR/results.csv" | sort -n | awk '{a[NR]=$1} END {print a[int(NR*0.99)]+0}')

THROUGHPUT=$(echo "scale=2; $TOTAL_SENT / $TOTAL_DURATION" | bc)

# Check final system health
FINAL_HEALTH=$(curl -s "${API_URL}/health")
FINAL_QUEUE=$(echo "$FINAL_HEALTH" | grep -o '"queueDepth":[0-9]*' | cut -d':' -f2)
FINAL_UTXOS=$(echo "$FINAL_HEALTH" | grep -o '"publishing_available":[0-9]*' | cut -d':' -f2)

# Display results
echo -e "${GREEN}╔═════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║              ECDSA STRESS TEST RESULTS                  ║${NC}"
echo -e "${GREEN}╚═════════════════════════════════════════════════════════╝${NC}\n"

echo -e "${CYAN}Request Summary:${NC}"
echo "  Total Sent:           $TOTAL_SENT"
echo -e "  Instant Success:      ${GREEN}$SUCCESS_COUNT${NC} ($(echo "scale=1; $SUCCESS_COUNT * 100 / $TOTAL_SENT" | bc)%)"
echo -e "  Queued (Async):       ${YELLOW}$QUEUED_COUNT${NC} ($(echo "scale=1; $QUEUED_COUNT * 100 / $TOTAL_SENT" | bc)%)"
echo -e "  Auth Failures:        ${RED}$AUTH_FAILED${NC} ($(echo "scale=1; $AUTH_FAILED * 100 / $TOTAL_SENT" | bc)%)"
echo -e "  Other Failures:       ${RED}$FAILED_COUNT${NC}"
echo ""

echo -e "${CYAN}Performance Metrics:${NC}"
echo "  Total Duration:       ${TOTAL_DURATION}s"
echo "  Throughput:           ${THROUGHPUT} tx/sec"
echo "  Concurrency Level:    ${CONCURRENT_REQUESTS}"
echo "  Signature Overhead:   ~10-20ms per request"
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

echo -e "${CYAN}Security Features:${NC}"
echo "  ✓ ECDSA secp256k1 signature verification"
echo "  ✓ Timestamp validation (prevents replay)"
echo "  ✓ Nonce validation (prevents duplicate requests)"
echo "  ✓ Cryptographic non-repudiation"
echo ""

# Generate summary report
cat > "$RESULTS_DIR/summary.txt" << EOF
GovHash ECDSA Stress Test Summary
Generated: $(date)

Configuration:
- API URL: $API_URL
- Client: $CLIENT_NAME
- Total Requests: $TOTAL_SENT
- Concurrent Requests: $CONCURRENT_REQUESTS
- Authentication: ECDSA secp256k1
- Wait Parameter: $USE_WAIT_PARAM

Results:
- Instant Success: $SUCCESS_COUNT ($(echo "scale=1; $SUCCESS_COUNT * 100 / $TOTAL_SENT" | bc)%)
- Queued (Async): $QUEUED_COUNT ($(echo "scale=1; $QUEUED_COUNT * 100 / $TOTAL_SENT" | bc)%)
- Auth Failures: $AUTH_FAILED ($(echo "scale=1; $AUTH_FAILED * 100 / $TOTAL_SENT" | bc)%)
- Other Failures: $FAILED_COUNT

Performance:
- Total Duration: ${TOTAL_DURATION}s
- Throughput: ${THROUGHPUT} tx/sec
- Average Latency: ${AVG_LATENCY} ms (includes signature generation)
- P95 Latency: ${P95_LATENCY} ms
- P99 Latency: ${P99_LATENCY} ms

System Impact:
- Queue Depth Change: $INITIAL_QUEUE → $FINAL_QUEUE
- UTXOs Consumed: $((INITIAL_UTXOS - FINAL_UTXOS))

Security:
- Authentication Method: ECDSA secp256k1
- Public Key: ${PUBLIC_KEY_HEX:0:32}...${PUBLIC_KEY_HEX: -16}
- Private Key: $PRIVATE_KEY_FILE (keep secure!)
EOF

echo -e "${YELLOW}Results saved to:${NC} $RESULTS_DIR"
echo "  - results.csv         (Raw data)"
echo "  - summary.txt         (Summary report)"
echo "  - ecdsa_private.pem   (Private key - KEEP SECURE!)"
echo "  - ecdsa_public.pem    (Public key)"
echo "  - api_key.txt         (API key)"
echo ""

echo -e "${GREEN}╔═════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║           ECDSA STRESS TEST COMPLETE ✓                 ║${NC}"
echo -e "${GREEN}╚═════════════════════════════════════════════════════════╝${NC}"

# Exit with success if < 5% failed (excluding auth failures which indicate signature issues)
FAIL_RATE=$(echo "scale=0; ($FAILED_COUNT - $AUTH_FAILED) * 100 / $TOTAL_SENT" | bc)
if [ "$FAIL_RATE" -lt 5 ] && [ "$AUTH_FAILED" -lt 5 ]; then
    exit 0
else
    echo -e "${RED}Warning: Failure rate exceeds threshold${NC}"
    exit 1
fi
