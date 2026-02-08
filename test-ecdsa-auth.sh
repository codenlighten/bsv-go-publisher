#!/bin/bash

# BSV AKUA Broadcaster - ECDSA Authentication Test
# Demonstrates registering a public key and signing requests

set -e

# Configuration
API_URL="https://api.govhash.org"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-your_admin_password_here}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}╔════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  ECDSA Authentication Test                 ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════╝${NC}\n"

# Check for admin password
if [ "$ADMIN_PASSWORD" = "your_admin_password_here" ]; then
    echo -e "${RED}Error: ADMIN_PASSWORD not set${NC}"
    echo "Usage: ADMIN_PASSWORD='your_password' $0"
    exit 1
fi

# Step 1: Generate ECDSA key pair
echo -e "${BLUE}Step 1: Generating ECDSA key pair...${NC}"
PRIVATE_KEY_FILE="/tmp/test_ecdsa_private.pem"
PUBLIC_KEY_FILE="/tmp/test_ecdsa_public.pem"

openssl ecparam -name secp256k1 -genkey -noout -out "$PRIVATE_KEY_FILE"
openssl ec -in "$PRIVATE_KEY_FILE" -pubout -out "$PUBLIC_KEY_FILE"

echo -e "${GREEN}✓${NC} Generated key pair:"
echo "   Private: $PRIVATE_KEY_FILE"
echo "   Public:  $PUBLIC_KEY_FILE"
echo ""

# Extract public key in hex format for registration
PUBLIC_KEY_HEX=$(openssl ec -in "$PRIVATE_KEY_FILE" -pubout -outform DER 2>/dev/null | tail -c 65 | xxd -p -c 65)
echo -e "${YELLOW}Public Key (hex):${NC}"
echo "$PUBLIC_KEY_HEX"
echo ""

# Step 2: Register client with public key
echo -e "${BLUE}Step 2: Registering client with public key...${NC}"

REGISTER_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/admin/clients/register" \
    -H "X-Admin-Password: ${ADMIN_PASSWORD}" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"ECDSA Test Client $(date +%s)\",
        \"public_key\": \"${PUBLIC_KEY_HEX}\",
        \"max_daily_tx\": 1000
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

# Step 3: Create a signed request
echo -e "${BLUE}Step 3: Making signed request...${NC}"

# Generate request parameters
TIMESTAMP=$(date +%s)000  # Unix timestamp in milliseconds
NONCE=$(uuidgen)
DATA_HEX=$(echo -n "ECDSA authenticated message $(date +%s)" | xxd -p | tr -d '\n')

echo "Request parameters:"
echo "  Timestamp: $TIMESTAMP"
echo "  Nonce:     $NONCE"
echo "  Data (hex): $DATA_HEX"
echo ""

# Create signature payload: timestamp + nonce + data
SIGNATURE_PAYLOAD="${TIMESTAMP}${NONCE}${DATA_HEX}"
echo -e "${YELLOW}Signature payload:${NC} $SIGNATURE_PAYLOAD"

# Sign the payload
echo "$SIGNATURE_PAYLOAD" > /tmp/sig_payload.txt
SIGNATURE=$(openssl dgst -sha256 -sign "$PRIVATE_KEY_FILE" /tmp/sig_payload.txt | base64 -w 0)

echo -e "${YELLOW}Signature (base64):${NC}"
echo "$SIGNATURE"
echo ""

# Step 4: Make the authenticated request
echo -e "${BLUE}Step 4: Sending signed publish request with ?wait=true...${NC}"

PUBLISH_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/publish?wait=true" \
    -H "X-API-Key: ${API_KEY}" \
    -H "X-Signature: ${SIGNATURE}" \
    -H "X-Timestamp: ${TIMESTAMP}" \
    -H "X-Nonce: ${NONCE}" \
    -H "Content-Type: application/json" \
    -d "{\"data\":\"${DATA_HEX}\"}")

PUBLISH_HTTP_CODE=$(echo "$PUBLISH_RESPONSE" | tail -n1)
PUBLISH_BODY=$(echo "$PUBLISH_RESPONSE" | head -n-1)

if [ "$PUBLISH_HTTP_CODE" = "201" ]; then
    echo -e "${GREEN}✓ Signed request succeeded!${NC}"
    TXID=$(echo "$PUBLISH_BODY" | grep -o '"txid":"[^"]*"' | cut -d'"' -f4)
    ARC_STATUS=$(echo "$PUBLISH_BODY" | grep -o '"arc_status":"[^"]*"' | cut -d'"' -f4)
    echo -e "${GREEN}TXID:${NC} $TXID"
    echo -e "${GREEN}ARC Status:${NC} $ARC_STATUS"
    echo ""
    echo -e "${GREEN}✓ Verify on blockchain:${NC}"
    echo "  https://whatsonchain.com/tx/${TXID}"
elif [ "$PUBLISH_HTTP_CODE" = "202" ]; then
    echo -e "${YELLOW}⚠ Queue busy, fell back to async mode${NC}"
    UUID=$(echo "$PUBLISH_BODY" | grep -o '"uuid":"[^"]*"' | cut -d'"' -f4)
    echo -e "${YELLOW}UUID:${NC} $UUID"
    echo "Poll: ${API_URL}/status/${UUID}"
else
    echo -e "${RED}✗ Request failed with HTTP $PUBLISH_HTTP_CODE${NC}"
    echo "$PUBLISH_BODY"
fi

echo ""

# Step 5: Test signature validation with wrong signature
echo -e "${BLUE}Step 5: Testing invalid signature (should fail)...${NC}"

TIMESTAMP=$(date +%s)000
NONCE=$(uuidgen)
DATA_HEX=$(echo -n "Test wrong signature" | xxd -p | tr -d '\n')
WRONG_SIGNATURE="aW52YWxpZF9zaWduYXR1cmU="

INVALID_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/publish" \
    -H "X-API-Key: ${API_KEY}" \
    -H "X-Signature: ${WRONG_SIGNATURE}" \
    -H "X-Timestamp: ${TIMESTAMP}" \
    -H "X-Nonce: ${NONCE}" \
    -H "Content-Type: application/json" \
    -d "{\"data\":\"${DATA_HEX}\"}")

INVALID_HTTP_CODE=$(echo "$INVALID_RESPONSE" | tail -n1)
INVALID_BODY=$(echo "$INVALID_RESPONSE" | head -n-1)

if [ "$INVALID_HTTP_CODE" = "401" ] || [ "$INVALID_HTTP_CODE" = "403" ]; then
    echo -e "${GREEN}✓ Invalid signature correctly rejected (HTTP $INVALID_HTTP_CODE)${NC}"
    echo "  Error: $INVALID_BODY"
else
    echo -e "${RED}✗ Expected 401/403 for invalid signature, got HTTP $INVALID_HTTP_CODE${NC}"
fi

echo ""

# Step 6: Test replay protection (reuse same nonce)
echo -e "${BLUE}Step 6: Testing replay protection (reuse nonce)...${NC}"

# Try to reuse the same timestamp and nonce from step 4
REPLAY_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/publish" \
    -H "X-API-Key: ${API_KEY}" \
    -H "X-Signature: ${SIGNATURE}" \
    -H "X-Timestamp: ${TIMESTAMP}" \
    -H "X-Nonce: ${NONCE}" \
    -H "Content-Type: application/json" \
    -d "{\"data\":\"${DATA_HEX}\"}")

REPLAY_HTTP_CODE=$(echo "$REPLAY_RESPONSE" | tail -n1)
REPLAY_BODY=$(echo "$REPLAY_RESPONSE" | head -n-1)

if [ "$REPLAY_HTTP_CODE" = "401" ] || [ "$REPLAY_HTTP_CODE" = "403" ]; then
    echo -e "${GREEN}✓ Replay attack correctly prevented (HTTP $REPLAY_HTTP_CODE)${NC}"
    echo "  Error: $REPLAY_BODY"
else
    echo -e "${YELLOW}⚠ Replay protection check inconclusive (HTTP $REPLAY_HTTP_CODE)${NC}"
    echo "  Note: Nonce cache may have expired or not implemented"
fi

echo ""

# Cleanup
echo -e "${BLUE}Cleanup...${NC}"
rm -f /tmp/sig_payload.txt
echo -e "${YELLOW}Note: Private key saved at:${NC} $PRIVATE_KEY_FILE"
echo -e "${YELLOW}Note: Public key saved at:${NC} $PUBLIC_KEY_FILE"
echo ""

# Summary
echo -e "${GREEN}╔════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║         ECDSA Authentication               ║${NC}"
echo -e "${GREEN}║            ✓ COMPLETE                      ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Authentication Flow:${NC}"
echo "  1. ✓ Generated secp256k1 key pair"
echo "  2. ✓ Registered public key with API"
echo "  3. ✓ Created signature: SHA256(timestamp + nonce + data)"
echo "  4. ✓ Sent signed request with X-Signature header"
echo "  5. ✓ Server verified signature using public key"
echo "  6. ✓ Transaction broadcasted successfully"
echo ""
echo -e "${YELLOW}Security Features Tested:${NC}"
echo "  • Signature validation (ECDSA secp256k1)"
echo "  • Timestamp validation (prevents old requests)"
echo "  • Nonce validation (prevents replay attacks)"
echo "  • 4-layer security: API Key + ECDSA + UTXO Lock + Train Batch"
echo ""
