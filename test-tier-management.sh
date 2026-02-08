#!/bin/bash

# BSV AKUA Broadcaster - Adaptive Security Tier Management Test
# Tests admin endpoints for tier-based security configuration

set -e

# Configuration
API_URL="https://api.govhash.org"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-your_admin_password_here}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

echo -e "${GREEN}╔═══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  Adaptive Security Tier Management Test      ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════╝${NC}\n"

# Check for admin password
if [ "$ADMIN_PASSWORD" = "your_admin_password_here" ]; then
    echo -e "${RED}Error: ADMIN_PASSWORD not set${NC}"
    echo "Usage: ADMIN_PASSWORD='your_password' $0"
    exit 1
fi

# Test 1: Register PILOT tier client (no public key required)
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}Test 1: Register PILOT Tier Client${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

PILOT_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/admin/clients/register" \
    -H "X-Admin-Password: ${ADMIN_PASSWORD}" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"AKUA Pilot Client $(date +%s)\",
        \"tier\": \"pilot\",
        \"max_daily_tx\": 10000,
        \"allowed_ips\": [\"127.0.0.1\", \"::1\"]
    }")

PILOT_HTTP_CODE=$(echo "$PILOT_RESPONSE" | tail -n1)
PILOT_BODY=$(echo "$PILOT_RESPONSE" | head -n-1)

if [ "$PILOT_HTTP_CODE" = "200" ] || [ "$PILOT_HTTP_CODE" = "201" ]; then
    echo -e "${GREEN}✓ Pilot client registered successfully${NC}"
    PILOT_CLIENT_ID=$(echo "$PILOT_BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    PILOT_API_KEY=$(echo "$PILOT_BODY" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4)
    PILOT_TIER=$(echo "$PILOT_BODY" | grep -o '"tier":"[^"]*"' | cut -d'"' -f4)
    PILOT_REQ_SIG=$(echo "$PILOT_BODY" | grep -o '"require_signature":[^,}]*' | cut -d':' -f2)
    
    echo -e "${YELLOW}Client ID:${NC} $PILOT_CLIENT_ID"
    echo -e "${YELLOW}API Key:${NC} $PILOT_API_KEY"
    echo -e "${YELLOW}Tier:${NC} $PILOT_TIER"
    echo -e "${YELLOW}Require Signature:${NC} $PILOT_REQ_SIG"
    echo -e "${YELLOW}Allowed IPs:${NC} 127.0.0.1, ::1"
    echo ""
else
    echo -e "${RED}✗ Registration failed with HTTP $PILOT_HTTP_CODE${NC}"
    echo "$PILOT_BODY"
    exit 1
fi

# Test 2: Test pilot client with API key only (no signature)
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}Test 2: Pilot Tier - API Key Only (No Signature)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

DATA_HEX=$(echo -n "PILOT tier test - no signature required" | xxd -p | tr -d '\n')

PILOT_PUBLISH=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/publish" \
    -H "X-API-Key: ${PILOT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"data\":\"${DATA_HEX}\"}")

PILOT_PUB_CODE=$(echo "$PILOT_PUBLISH" | tail -n1)
PILOT_PUB_BODY=$(echo "$PILOT_PUBLISH" | head -n-1)

if [ "$PILOT_PUB_CODE" = "201" ] || [ "$PILOT_PUB_CODE" = "202" ]; then
    echo -e "${GREEN}✓ Pilot client request succeeded WITHOUT signature${NC}"
    echo -e "${YELLOW}HTTP Status:${NC} $PILOT_PUB_CODE"
    echo "$PILOT_PUB_BODY"
    echo ""
else
    echo -e "${RED}✗ Request failed with HTTP $PILOT_PUB_CODE${NC}"
    echo "$PILOT_PUB_BODY"
fi

echo ""

# Test 3: Register ENTERPRISE tier client (requires public key)
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}Test 3: Register ENTERPRISE Tier Client${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

# Generate test key
ENTERPRISE_PRIVATE_KEY="/tmp/test_enterprise_private.pem"
ENTERPRISE_PUBLIC_KEY="/tmp/test_enterprise_public.pem"
openssl ecparam -name secp256k1 -genkey -noout -out "$ENTERPRISE_PRIVATE_KEY" 2>/dev/null
openssl ec -in "$ENTERPRISE_PRIVATE_KEY" -pubout -out "$ENTERPRISE_PUBLIC_KEY" 2>/dev/null
ENTERPRISE_PUBLIC_KEY_HEX=$(openssl ec -in "$ENTERPRISE_PRIVATE_KEY" -pubout -outform DER 2>/dev/null | tail -c 65 | xxd -p -c 65)

ENTERPRISE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/admin/clients/register" \
    -H "X-Admin-Password: ${ADMIN_PASSWORD}" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"NotaryHash Enterprise $(date +%s)\",
        \"public_key\": \"${ENTERPRISE_PUBLIC_KEY_HEX}\",
        \"tier\": \"enterprise\",
        \"max_daily_tx\": 100000
    }")

ENTERPRISE_HTTP_CODE=$(echo "$ENTERPRISE_RESPONSE" | tail -n1)
ENTERPRISE_BODY=$(echo "$ENTERPRISE_RESPONSE" | head -n-1)

if [ "$ENTERPRISE_HTTP_CODE" = "200" ] || [ "$ENTERPRISE_HTTP_CODE" = "201" ]; then
    echo -e "${GREEN}✓ Enterprise client registered successfully${NC}"
    ENTERPRISE_CLIENT_ID=$(echo "$ENTERPRISE_BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    ENTERPRISE_API_KEY=$(echo "$ENTERPRISE_BODY" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4)
    ENTERPRISE_TIER=$(echo "$ENTERPRISE_BODY" | grep -o '"tier":"[^"]*"' | cut -d'"' -f4)
    ENTERPRISE_REQ_SIG=$(echo "$ENTERPRISE_BODY" | grep -o '"require_signature":[^,}]*' | cut -d':' -f2)
    ENTERPRISE_GRACE=$(echo "$ENTERPRISE_BODY" | grep -o '"grace_period_hours":[^,}]*' | cut -d':' -f2)
    
    echo -e "${YELLOW}Client ID:${NC} $ENTERPRISE_CLIENT_ID"
    echo -e "${YELLOW}API Key:${NC} $ENTERPRISE_API_KEY"
    echo -e "${YELLOW}Tier:${NC} $ENTERPRISE_TIER"
    echo -e "${YELLOW}Require Signature:${NC} $ENTERPRISE_REQ_SIG"
    echo -e "${YELLOW}Grace Period:${NC} $ENTERPRISE_GRACE hours"
    echo ""
else
    echo -e "${RED}✗ Registration failed with HTTP $ENTERPRISE_HTTP_CODE${NC}"
    echo "$ENTERPRISE_BODY"
fi

echo ""

# Test 4: Test enterprise client WITHOUT signature (should fail)
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}Test 4: Enterprise Tier - Reject Without Signature${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

DATA_HEX=$(echo -n "Enterprise test - should fail without signature" | xxd -p | tr -d '\n')

ENTERPRISE_NO_SIG=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/publish" \
    -H "X-API-Key: ${ENTERPRISE_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"data\":\"${DATA_HEX}\"}")

ENTERPRISE_NO_SIG_CODE=$(echo "$ENTERPRISE_NO_SIG" | tail -n1)
ENTERPRISE_NO_SIG_BODY=$(echo "$ENTERPRISE_NO_SIG" | head -n-1)

if [ "$ENTERPRISE_NO_SIG_CODE" = "401" ] || [ "$ENTERPRISE_NO_SIG_CODE" = "403" ]; then
    echo -e "${GREEN}✓ Enterprise tier correctly rejected request without signature${NC}"
    echo -e "${YELLOW}HTTP Status:${NC} $ENTERPRISE_NO_SIG_CODE"
    echo "$ENTERPRISE_NO_SIG_BODY"
else
    echo -e "${RED}✗ Expected 401/403, got HTTP $ENTERPRISE_NO_SIG_CODE${NC}"
    echo "$ENTERPRISE_NO_SIG_BODY"
fi

echo ""

# Test 5: Upgrade pilot client to enterprise tier
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}Test 5: Runtime Tier Upgrade (Pilot → Enterprise)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

echo -e "${PURPLE}Upgrading client $PILOT_CLIENT_ID to enterprise tier...${NC}"

UPGRADE_RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH "${API_URL}/admin/clients/${PILOT_CLIENT_ID}/security" \
    -H "X-Admin-Password: ${ADMIN_PASSWORD}" \
    -H "Content-Type: application/json" \
    -d "{
        \"tier\": \"enterprise\",
        \"require_signature\": true,
        \"grace_period_hours\": 48
    }")

UPGRADE_HTTP_CODE=$(echo "$UPGRADE_RESPONSE" | tail -n1)
UPGRADE_BODY=$(echo "$UPGRADE_RESPONSE" | head -n-1)

if [ "$UPGRADE_HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ Client upgraded to enterprise tier${NC}"
    echo "$UPGRADE_BODY"
    echo ""
    
    # Test that pilot API key now requires signature
    echo -e "${PURPLE}Testing that upgraded client now requires signature...${NC}"
    
    UPGRADED_TEST=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/publish" \
        -H "X-API-Key: ${PILOT_API_KEY}" \
        -H "Content-Type: application/json" \
        -d "{\"data\":\"${DATA_HEX}\"}")
    
    UPGRADED_TEST_CODE=$(echo "$UPGRADED_TEST" | tail -n1)
    
    if [ "$UPGRADED_TEST_CODE" = "401" ] || [ "$UPGRADED_TEST_CODE" = "403" ]; then
        echo -e "${GREEN}✓ Upgraded client now correctly requires signature${NC}"
    else
        echo -e "${RED}✗ Expected 401/403, got HTTP $UPGRADED_TEST_CODE${NC}"
    fi
else
    echo -e "${RED}✗ Upgrade failed with HTTP $UPGRADE_HTTP_CODE${NC}"
    echo "$UPGRADE_BODY"
fi

echo ""

# Test 6: Downgrade enterprise back to pilot
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}Test 6: Runtime Tier Downgrade (Enterprise → Pilot)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

echo -e "${PURPLE}Downgrading client $ENTERPRISE_CLIENT_ID to pilot tier...${NC}"

DOWNGRADE_RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH "${API_URL}/admin/clients/${ENTERPRISE_CLIENT_ID}/security" \
    -H "X-Admin-Password: ${ADMIN_PASSWORD}" \
    -H "Content-Type: application/json" \
    -d "{
        \"tier\": \"pilot\",
        \"require_signature\": false,
        \"allowed_ips\": [\"127.0.0.1\"]
    }")

DOWNGRADE_HTTP_CODE=$(echo "$DOWNGRADE_RESPONSE" | tail -n1)
DOWNGRADE_BODY=$(echo "$DOWNGRADE_RESPONSE" | head -n-1)

if [ "$DOWNGRADE_HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ Client downgraded to pilot tier${NC}"
    echo "$DOWNGRADE_BODY"
else
    echo -e "${RED}✗ Downgrade failed with HTTP $DOWNGRADE_HTTP_CODE${NC}"
    echo "$DOWNGRADE_BODY"
fi

echo ""

# Summary
echo -e "${GREEN}╔═══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║    Adaptive Tier Management Test Complete    ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Tier Management Features Tested:${NC}"
echo "  ✓ Pilot tier registration (no public key required)"
echo "  ✓ Pilot tier authentication (API key only)"
echo "  ✓ Enterprise tier registration (public key required)"
echo "  ✓ Enterprise tier signature enforcement"
echo "  ✓ Runtime tier upgrade (pilot → enterprise)"
echo "  ✓ Runtime tier downgrade (enterprise → pilot)"
echo ""
echo -e "${YELLOW}Security Tier Matrix:${NC}"
echo "  • Pilot:      API Key Only       | 10 req/min   | Testing"
echo "  • Enterprise: API Key + ECDSA    | 100 req/min  | Commercial"
echo "  • Government: API Key + ECDSA + IP | Unlimited  | Institutional"
echo ""
echo -e "${YELLOW}Test Clients Created:${NC}"
echo "  • Pilot:      $PILOT_CLIENT_ID"
echo "  • Enterprise: $ENTERPRISE_CLIENT_ID"
echo ""
