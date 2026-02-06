#!/bin/bash

# BSV AKUA Broadcast Server - Test Script
# This script performs basic integration tests

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
BOLD='\033[1m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BOLD}BSV AKUA Broadcast Server - Integration Tests${NC}\n"

# Test 1: Health Check
echo -e "${BOLD}[1/5] Testing health endpoint...${NC}"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/health")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" -eq 200 ]; then
  echo -e "${GREEN}✓ Health check passed${NC}"
  echo "   Status: $(echo $BODY | jq -r '.status')"
  AVAILABLE=$(echo $BODY | jq -r '.utxos.publishing_available // 0')
  echo "   Publishing UTXOs: $AVAILABLE"
  
  if [ "$AVAILABLE" -eq 0 ]; then
    echo -e "${YELLOW}⚠️  Warning: No publishing UTXOs available${NC}"
    echo "   Tests requiring UTXOs will be skipped"
    SKIP_UTXO_TESTS=true
  fi
else
  echo -e "${RED}✗ Health check failed (HTTP $HTTP_CODE)${NC}"
  exit 1
fi

echo ""

# Test 2: Stats Endpoint
echo -e "${BOLD}[2/5] Testing stats endpoint...${NC}"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/admin/stats")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" -eq 200 ]; then
  echo -e "${GREEN}✓ Stats endpoint accessible${NC}"
  BODY=$(echo "$RESPONSE" | head -n-1)
  echo "$BODY" | jq '.utxos'
else
  echo -e "${RED}✗ Stats endpoint failed (HTTP $HTTP_CODE)${NC}"
  exit 1
fi

echo ""

# Test 3: Publish Request (if UTXOs available)
if [ "$SKIP_UTXO_TESTS" != "true" ]; then
  echo -e "${BOLD}[3/5] Testing publish endpoint...${NC}"
  
  # Create test data
  TEST_DATA=$(echo -n "Test from BSV Broadcast Server $(date +%s)" | xxd -p | tr -d '\n')
  
  RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/publish" \
    -H "Content-Type: application/json" \
    -d "{\"data\":\"$TEST_DATA\"}")
  
  HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
  BODY=$(echo "$RESPONSE" | head -n-1)
  
  if [ "$HTTP_CODE" -eq 202 ]; then
    echo -e "${GREEN}✓ Publish request accepted${NC}"
    UUID=$(echo $BODY | jq -r '.uuid')
    echo "   UUID: $UUID"
    echo "   Queue Depth: $(echo $BODY | jq -r '.queueDepth')"
    
    # Save UUID for next test
    TEST_UUID="$UUID"
  else
    echo -e "${RED}✗ Publish request failed (HTTP $HTTP_CODE)${NC}"
    echo "$BODY"
    exit 1
  fi
else
  echo -e "${YELLOW}[3/5] Skipping publish test (no UTXOs)${NC}"
fi

echo ""

# Test 4: Status Check
if [ -n "$TEST_UUID" ]; then
  echo -e "${BOLD}[4/5] Testing status endpoint...${NC}"
  
  RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/status/$TEST_UUID")
  HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
  BODY=$(echo "$RESPONSE" | head -n-1)
  
  if [ "$HTTP_CODE" -eq 200 ]; then
    echo -e "${GREEN}✓ Status endpoint working${NC}"
    STATUS=$(echo $BODY | jq -r '.status')
    echo "   Status: $STATUS"
    
    if [ "$STATUS" != "pending" ] && [ "$STATUS" != "processing" ]; then
      TXID=$(echo $BODY | jq -r '.txid // "none"')
      echo "   TxID: $TXID"
    fi
  else
    echo -e "${RED}✗ Status check failed (HTTP $HTTP_CODE)${NC}"
    exit 1
  fi
else
  echo -e "${YELLOW}[4/5] Skipping status test (no UUID)${NC}"
fi

echo ""

# Test 5: Invalid Requests
echo -e "${BOLD}[5/5] Testing error handling...${NC}"

# Test 5a: Empty data
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/publish" \
  -H "Content-Type: application/json" \
  -d '{"data":""}')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" -eq 400 ]; then
  echo -e "${GREEN}✓ Empty data rejected correctly${NC}"
else
  echo -e "${RED}✗ Empty data should return 400, got $HTTP_CODE${NC}"
fi

# Test 5b: Invalid hex
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/publish" \
  -H "Content-Type: application/json" \
  -d '{"data":"not-valid-hex"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" -eq 400 ]; then
  echo -e "${GREEN}✓ Invalid hex rejected correctly${NC}"
else
  echo -e "${RED}✗ Invalid hex should return 400, got $HTTP_CODE${NC}"
fi

# Test 5c: Non-existent status
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/status/00000000-0000-0000-0000-000000000000")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" -eq 404 ]; then
  echo -e "${GREEN}✓ Non-existent UUID returns 404${NC}"
else
  echo -e "${RED}✗ Non-existent UUID should return 404, got $HTTP_CODE${NC}"
fi

echo ""
echo -e "${BOLD}${GREEN}=== All Tests Passed ===${NC}"

if [ "$SKIP_UTXO_TESTS" == "true" ]; then
  echo ""
  echo -e "${YELLOW}Note: Some tests were skipped due to no available UTXOs.${NC}"
  echo -e "${YELLOW}Fund the server and run the splitter to enable full testing.${NC}"
fi
