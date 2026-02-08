#!/bin/bash

# Pilot Migration Script for Existing AKUA Client
# Migrates the production AKUA key to pilot tier with signature disabled

set -e

# Configuration
MONGO_URI="${MONGO_URI:-mongodb://localhost:27017}"
DATABASE="bsv_broadcaster"
COLLECTION="clients"

# AKUA Production API Key Hash (gh_KqxxVawkirYuNvyzXEELUzUAA3-_20nzRAm-QWF2P-M=)
# Note: This is the HASHED version stored in MongoDB
AKUA_API_KEY_HASH="your_akua_key_hash_here"  # Replace with actual hash from DB

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}╔═══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     AKUA Pilot Tier Migration                ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════╝${NC}\n"

echo -e "${BLUE}Migrating AKUA production client to pilot tier...${NC}\n"

# MongoDB update command
UPDATE_COMMAND="db.${COLLECTION}.updateOne(
    {name: 'AKUA Production'},
    {\$set: {
        tier: 'pilot',
        require_signature: false,
        grace_period_hours: 0,
        allowed_ips: [],
        updated_at: new Date()
    }}
)"

echo -e "${YELLOW}MongoDB Update Command:${NC}"
echo "$UPDATE_COMMAND"
echo ""

# Execute migration
echo -e "${BLUE}Executing migration...${NC}"

mongo "$MONGO_URI/$DATABASE" --eval "$UPDATE_COMMAND"

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}╔═══════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║         ✓ Migration Complete                 ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${YELLOW}AKUA Client Security Status:${NC}"
    echo "  • Tier: pilot"
    echo "  • Require Signature: false"
    echo "  • Grace Period: 0 hours (not applicable)"
    echo "  • Allowed IPs: [] (all IPs allowed)"
    echo ""
    echo -e "${GREEN}✓ AKUA can now use the API with API key only${NC}"
    echo -e "${GREEN}✓ No ECDSA signature required${NC}"
    echo -e "${GREEN}✓ Existing API key remains valid${NC}"
    echo ""
else
    echo ""
    echo -e "${RED}✗ Migration failed${NC}"
    exit 1
fi

# Verification query
echo -e "${BLUE}Verifying migration...${NC}"

VERIFY_COMMAND="db.${COLLECTION}.findOne(
    {name: 'AKUA Production'},
    {name: 1, tier: 1, require_signature: 1, grace_period_hours: 1}
)"

mongo "$MONGO_URI/$DATABASE" --eval "$VERIFY_COMMAND"

echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo "  1. Deploy updated server code to production"
echo "  2. Test AKUA client with existing API key (no signature)"
echo "  3. When ready to upgrade, use: PATCH /admin/clients/:id/security"
echo "  4. Monitor logs for [PILOT] tier requests"
echo ""
