#!/bin/bash

# Production Migration - AKUA Pilot Tier
# Migrates existing AKUA client to pilot tier (no signature required)

set -e

echo "🔄 Migrating AKUA client to pilot tier..."

# Connect to MongoDB and update client
docker exec -it bsv_akua_db mongosh \
  -u root \
  -p "${MONGO_PASSWORD}" \
  --authenticationDatabase admin \
  bsv_broadcaster \
  --eval '
    db.clients.updateOne(
      { name: "AKUA Production" },
      {
        $set: {
          tier: "pilot",
          require_signature: false,
          grace_period_hours: 0,
          allowed_ips: [],
          updated_at: new Date()
        }
      }
    );
    
    print("\n✅ Migration result:");
    printjson(db.clients.findOne(
      { name: "AKUA Production" },
      { name: 1, tier: 1, require_signature: 1, api_key_hash: 1, _id: 1 }
    ));
  '

echo ""
echo "✅ AKUA client migrated to pilot tier"
echo "   • Tier: pilot"
echo "   • Require Signature: false"
echo "   • Grace Period: 0 hours"
echo "   • API key remains unchanged"
echo ""
echo "🔥 AKUA can now use API with API key only (no ECDSA signature required)"
