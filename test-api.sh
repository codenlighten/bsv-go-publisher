#!/bin/bash

# BSV AKUA Broadcaster - API Test Suite
# Tests all endpoints with the production API

set -e

# Configuration
API_URL="https://api.govhash.org"
API_KEY="gh_KqxxVawkirYuNvyzXEELUzUAA3-_20nzRAm-QWF2P-M="
ADMIN_PASSWORD="${ADMIN_PASSWORD:-your_admin_password_here}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
PASSED=0
FAILED=0

# Helper functions
print_test() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}TEST: $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((PASSED++))
}

print_fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((FAILED++))
}

print_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Generate random hex data
random_hex() {
    echo -n "Test message $(date +%s)" | xxd -p | tr -d '\n'
}

# Start tests
echo -e "${GREEN}╔════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  BSV AKUA Broadcaster - API Test Suite    ║${NC}"
echo -e "${GREEN}╔════════════════════════════════════════════╝${NC}"
echo -e "${GREEN}║${NC} API: ${API_URL}"
echo -e "${GREEN}╚════════════════════════════════════════════${NC}\n"

# Test 1: Health Check
print_test "GET /health - Health check endpoint"
RESPONSE=$(curl -s -w "\n%{http_code}" "${API_URL}/health")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ]; then
    print_pass "Health check returned 200 OK"
    print_info "Response: $BODY"
    
    # Check for expected fields
    if echo "$BODY" | grep -q "utxoStats"; then
        print_pass "Response contains utxoStats"
    else
        print_fail "Response missing utxoStats"
    fi
else
    print_fail "Health check failed with HTTP $HTTP_CODE"
fi

# Test 2: Publish - Async Mode (default)
print_test "POST /publish - Async mode (default)"
DATA=$(random_hex)
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/publish" \
    -H "X-API-Key: ${API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"data\":\"${DATA}\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "202" ]; then
    print_pass "Async publish returned 202 Accepted"
    UUID=$(echo "$BODY" | grep -o '"uuid":"[^"]*"' | cut -d'"' -f4)
    print_info "UUID: $UUID"
    
    # Wait for broadcast
    sleep 4
    
    # Test 3: Status Check
    print_test "GET /status/:uuid - Check broadcast status"
    STATUS_RESPONSE=$(curl -s -w "\n%{http_code}" "${API_URL}/status/${UUID}")
    STATUS_HTTP_CODE=$(echo "$STATUS_RESPONSE" | tail -n1)
    STATUS_BODY=$(echo "$STATUS_RESPONSE" | head -n-1)
    
    if [ "$STATUS_HTTP_CODE" = "200" ]; then
        print_pass "Status check returned 200 OK"
        print_info "Status: $STATUS_BODY"
        
        if echo "$STATUS_BODY" | grep -q "success"; then
            TXID=$(echo "$STATUS_BODY" | grep -o '"txid":"[^"]*"' | cut -d'"' -f4)
            print_pass "Transaction broadcasted successfully"
            print_info "TXID: $TXID"
        fi
    else
        print_fail "Status check failed with HTTP $STATUS_HTTP_CODE"
    fi
else
    print_fail "Async publish failed with HTTP $HTTP_CODE: $BODY"
fi

# Test 4: Publish - Synchronous Mode
print_test "POST /publish?wait=true - Synchronous mode"
DATA=$(random_hex)
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/publish?wait=true" \
    -H "X-API-Key: ${API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"data\":\"${DATA}\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "201" ]; then
    print_pass "Synchronous publish returned 201 Created"
    TXID=$(echo "$BODY" | grep -o '"txid":"[^"]*"' | cut -d'"' -f4)
    ARC_STATUS=$(echo "$BODY" | grep -o '"arc_status":"[^"]*"' | cut -d'"' -f4)
    print_pass "Got immediate TXID (no polling needed)"
    print_info "TXID: $TXID"
    print_info "ARC Status: $ARC_STATUS"
elif [ "$HTTP_CODE" = "202" ]; then
    print_pass "Queue busy, fell back to async mode (expected behavior)"
    print_info "Response: $BODY"
else
    print_fail "Synchronous publish failed with HTTP $HTTP_CODE: $BODY"
fi

# Test 5: Invalid API Key
print_test "POST /publish - Invalid API key"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/publish" \
    -H "X-API-Key: invalid_key" \
    -H "Content-Type: application/json" \
    -d "{\"data\":\"$(random_hex)\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "403" ]; then
    print_pass "Invalid API key rejected with HTTP $HTTP_CODE"
else
    print_fail "Expected 401/403 for invalid API key, got HTTP $HTTP_CODE"
fi

# Test 6: Missing data field
print_test "POST /publish - Missing data field"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/publish" \
    -H "X-API-Key: ${API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{}")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "400" ]; then
    print_pass "Missing data field rejected with 400 Bad Request"
    print_info "Error: $BODY"
else
    print_fail "Expected 400 for missing data, got HTTP $HTTP_CODE"
fi

# Test 7: Admin Stats (if password provided)
if [ "$ADMIN_PASSWORD" != "your_admin_password_here" ]; then
    print_test "GET /admin/stats - Admin statistics"
    RESPONSE=$(curl -s -w "\n%{http_code}" "${API_URL}/admin/stats" \
        -H "X-Admin-Password: ${ADMIN_PASSWORD}")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | head -n-1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        print_pass "Admin stats returned 200 OK"
        print_info "Stats: $BODY"
    else
        print_fail "Admin stats failed with HTTP $HTTP_CODE"
    fi
else
    print_info "Skipping admin tests (set ADMIN_PASSWORD environment variable to test)"
fi

# Test 8: Invalid endpoint
print_test "GET /invalid - Non-existent endpoint"
RESPONSE=$(curl -s -w "\n%{http_code}" "${API_URL}/invalid")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "404" ]; then
    print_pass "Invalid endpoint returned 404 Not Found"
else
    print_fail "Expected 404 for invalid endpoint, got HTTP $HTTP_CODE"
fi

# Summary
echo -e "\n${GREEN}╔════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║              TEST SUMMARY                  ║${NC}"
echo -e "${GREEN}╔════════════════════════════════════════════╝${NC}"
echo -e "${GREEN}║${NC} Total Tests: $((PASSED + FAILED))"
echo -e "${GREEN}║${NC} ${GREEN}Passed: ${PASSED}${NC}"
echo -e "${GREEN}║${NC} ${RED}Failed: ${FAILED}${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════${NC}\n"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}\n"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}\n"
    exit 1
fi
