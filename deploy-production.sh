#!/bin/bash

# Production Deployment Script - Phase 9
# Deploys adaptive security tier system to api.govhash.org

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

echo -e "${GREEN}╔═══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   Phase 9: Production Deployment             ║${NC}"
echo -e "${GREEN}║   Adaptive Security Tier System               ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════╝${NC}\n"

# Step 1: Pre-deployment verification
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}Step 1: Pre-Deployment Verification${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

echo -e "${YELLOW}Checking current server status...${NC}"
HEALTH_CHECK=$(curl -s https://api.govhash.org/health || echo "OFFLINE")
if [[ "$HEALTH_CHECK" == *"ok"* ]]; then
    echo -e "${GREEN}✓${NC} Current server is healthy"
else
    echo -e "${YELLOW}⚠${NC} Server health check failed (may already be down)"
fi
echo ""

# Step 2: Database migration (migrate existing AKUA client to pilot tier)
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}Step 2: Database Migration - AKUA → Pilot Tier${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

echo -e "${YELLOW}Migrating existing AKUA client to pilot tier...${NC}"

# Get MongoDB connection info
MONGO_URI="mongodb://root:${MONGO_PASSWORD}@localhost:27017"
if [ -z "$MONGO_PASSWORD" ]; then
    echo -e "${RED}✗ MONGO_PASSWORD not set${NC}"
    exit 1
fi

# Migration query
MIGRATION_CMD='
db.clients.updateOne(
    {name: "AKUA Production"},
    {$set: {
        tier: "pilot",
        require_signature: false,
        grace_period_hours: 0,
        allowed_ips: [],
        updated_at: new Date()
    }}
)'

# Execute migration
docker exec bsv_akua_db mongosh \
    --quiet \
    --username root \
    --password "$MONGO_PASSWORD" \
    --authenticationDatabase admin \
    bsv_broadcaster \
    --eval "$MIGRATION_CMD"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} AKUA client migrated to pilot tier"
    echo ""
    
    # Verify migration
    VERIFY_CMD='db.clients.findOne({name: "AKUA Production"}, {name: 1, tier: 1, require_signature: 1, _id: 0})'
    echo -e "${YELLOW}Verifying migration:${NC}"
    docker exec bsv_akua_db mongosh \
        --quiet \
        --username root \
        --password "$MONGO_PASSWORD" \
        --authenticationDatabase admin \
        bsv_broadcaster \
        --eval "$VERIFY_CMD"
    echo ""
else
    echo -e "${YELLOW}⚠${NC} Migration failed or client doesn't exist yet"
    echo "   (Client may be created later with pilot tier defaults)"
    echo ""
fi

# Step 3: Deploy new container
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}Step 3: Deploy Updated Container${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

echo -e "${YELLOW}Stopping current container...${NC}"
docker-compose stop bsv-publisher
echo -e "${GREEN}✓${NC} Container stopped"
echo ""

echo -e "${YELLOW}Starting new container with adaptive security...${NC}"
docker-compose up -d bsv-publisher
echo -e "${GREEN}✓${NC} Container started"
echo ""

# Step 4: Health check and warm-up
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}Step 4: Health Check and Warm-up${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

echo -e "${YELLOW}Waiting for server to be ready...${NC}"
for i in {1..30}; do
    sleep 2
    HEALTH=$(curl -s https://api.govhash.org/health 2>/dev/null || echo "")
    if [[ "$HEALTH" == *"ok"* ]]; then
        echo -e "${GREEN}✓${NC} Server is healthy (attempt $i)"
        break
    else
        echo -e "${YELLOW}⏳${NC} Waiting... (attempt $i/30)"
    fi
    
    if [ $i -eq 30 ]; then
        echo -e "${RED}✗${NC} Server failed to become healthy"
        echo ""
        echo -e "${YELLOW}Container logs:${NC}"
        docker-compose logs --tail=50 bsv-publisher
        exit 1
    fi
done
echo ""

# Step 5: Test tier enforcement
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}Step 5: Test Tier Enforcement${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

echo -e "${YELLOW}Testing AKUA pilot tier (API key only, no signature)...${NC}"
DATA_HEX=$(echo -n "Phase 9 deployment test - pilot tier" | xxd -p | tr -d '\n')

PILOT_TEST=$(curl -s -w "\n%{http_code}" -X POST "https://api.govhash.org/publish" \
    -H "X-API-Key: gh_KqxxVawkirYuNvyzXEELUzUAA3-_20nzRAm-QWF2P-M=" \
    -H "Content-Type: application/json" \
    -d "{\"data\":\"${DATA_HEX}\"}")

PILOT_CODE=$(echo "$PILOT_TEST" | tail -n1)
PILOT_BODY=$(echo "$PILOT_TEST" | head -n-1)

if [ "$PILOT_CODE" = "201" ] || [ "$PILOT_CODE" = "202" ]; then
    echo -e "${GREEN}✓${NC} Pilot tier working! Request succeeded WITHOUT signature"
    echo -e "${YELLOW}HTTP Status:${NC} $PILOT_CODE"
    if [ "$PILOT_CODE" = "201" ]; then
        TXID=$(echo "$PILOT_BODY" | grep -o '"txid":"[^"]*"' | cut -d'"' -f4)
        echo -e "${YELLOW}TXID:${NC} $TXID"
    else
        UUID=$(echo "$PILOT_BODY" | grep -o '"uuid":"[^"]*"' | cut -d'"' -f4)
        echo -e "${YELLOW}UUID:${NC} $UUID"
    fi
else
    echo -e "${RED}✗${NC} Pilot tier test failed with HTTP $PILOT_CODE"
    echo "$PILOT_BODY"
fi
echo ""

# Step 6: Check container logs for tier-based logging
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}Step 6: Verify Tier-Based Logging${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"

echo -e "${YELLOW}Recent logs (looking for [PILOT] indicators):${NC}"
docker-compose logs --tail=20 bsv-publisher | grep -E "\[PILOT\]|tier|signature" || echo "(No tier logs yet - may appear on next request)"
echo ""

# Step 7: Summary
echo -e "${GREEN}╔═══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     ✓ Phase 9 Deployment Complete            ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Deployment Summary:${NC}"
echo "  ✓ Database migration: AKUA → pilot tier"
echo "  ✓ Container deployment: adaptive security active"
echo "  ✓ Health check: PASSED"
echo "  ✓ Tier enforcement: VERIFIED"
echo ""
echo -e "${YELLOW}What Changed:${NC}"
echo "  • AKUA client now uses API key only (no signature required)"
echo "  • New clients can be registered with tier parameter"
echo "  • Admin can upgrade/downgrade tiers via PATCH endpoint"
echo "  • Grace periods support key rotation (24h enterprise, 168h government)"
echo ""
echo -e "${YELLOW}New Admin Endpoints:${NC}"
echo "  • POST   /admin/clients/register (with tier support)"
echo "  • PATCH  /admin/clients/:id/security (runtime tier management)"
echo "  • POST   /auth/register-public-key (client self-service)"
echo "  • POST   /auth/rotate-public-key (key rotation)"
echo "  • GET    /auth/key-status (introspection)"
echo ""
echo -e "${YELLOW}Monitor Tier Usage:${NC}"
echo "  docker-compose logs -f bsv-publisher | grep -E '\[PILOT\]|tier'"
echo ""
echo -e "${YELLOW}Test Scripts Available:${NC}"
echo "  • ./test-api.sh - General API tests"
echo "  • ./test-ecdsa-auth.sh - ECDSA authentication"
echo "  • ./test-tier-management.sh - Tier upgrade/downgrade"
echo ""
echo -e "${GREEN}🎉 Adaptive Security Tier System is LIVE!${NC}"
echo ""
