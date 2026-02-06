#!/bin/bash
#
# Weekly Maintenance Script for GovHash
# Consolidates UTXOs, checks system health, generates reports
#
# Usage: ./weekly-maintenance.sh
# Cron: 0 3 * * 0 /path/to/weekly-maintenance.sh >> /var/log/govhash-maintenance.log 2>&1

set -e

API_URL="${API_URL:-https://api.govhash.org}"
ADMIN_PASSWORD="${ADMIN_PASSWORD}"
PUBLISHING_ADDRESS="${PUBLISHING_ADDRESS}"

if [ -z "$ADMIN_PASSWORD" ]; then
    echo "Error: ADMIN_PASSWORD not set"
    exit 1
fi

if [ -z "$PUBLISHING_ADDRESS" ]; then
    echo "Error: PUBLISHING_ADDRESS not set"
    exit 1
fi

echo "=================================================="
echo "GovHash Weekly Maintenance"
echo "Date: $(date '+%Y-%m-%d %H:%M:%S')"
echo "=================================================="
echo ""

# 1. Check system health
echo "1. Checking system health..."
HEALTH=$(curl -s "$API_URL/health")
echo "$HEALTH" | jq '.'

PUBLISHING_AVAILABLE=$(echo "$HEALTH" | jq -r '.utxos.publishing_available')
PUBLISHING_SPENT=$(echo "$HEALTH" | jq -r '.utxos.publishing_spent')
QUEUE_DEPTH=$(echo "$HEALTH" | jq -r '.queueDepth')

echo ""
echo "Current Status:"
echo "  Publishing UTXOs Available: $PUBLISHING_AVAILABLE"
echo "  Publishing UTXOs Spent: $PUBLISHING_SPENT"
echo "  Queue Depth: $QUEUE_DEPTH"
echo ""

# 2. Check if consolidation is needed (if spent UTXOs > 100)
if [ "$PUBLISHING_SPENT" -gt 100 ]; then
    echo "2. Consolidating spent UTXOs (count: $PUBLISHING_SPENT)..."
    
    # Estimate sweep value first
    ESTIMATE=$(curl -s -X GET "$API_URL/admin/maintenance/estimate-sweep?utxo_type=publishing&max_inputs=100" \
        -H "X-Admin-Password: $ADMIN_PASSWORD")
    
    echo "Estimate:"
    echo "$ESTIMATE" | jq '.'
    
    CONSOLIDATE_COUNT=$(echo "$ESTIMATE" | jq -r '.count')
    CONSOLIDATE_SATS=$(echo "$ESTIMATE" | jq -r '.total_sats')
    
    if [ "$CONSOLIDATE_COUNT" -gt 50 ]; then
        echo "Proceeding with consolidation ($CONSOLIDATE_COUNT UTXOs, $CONSOLIDATE_SATS sats)..."
        
        RESULT=$(curl -s -X POST "$API_URL/admin/maintenance/sweep" \
            -H "Content-Type: application/json" \
            -H "X-Admin-Password: $ADMIN_PASSWORD" \
            -d "{
                \"dest_address\": \"$PUBLISHING_ADDRESS\",
                \"max_inputs\": 100,
                \"utxo_type\": \"publishing\"
            }")
        
        echo "Result:"
        echo "$RESULT" | jq '.'
        
        TXID=$(echo "$RESULT" | jq -r '.txid')
        if [ "$TXID" != "null" ]; then
            echo "✓ Consolidation successful: $TXID"
            echo "  View on WhatsOnChain: https://whatsonchain.com/tx/$TXID"
        else
            echo "✗ Consolidation failed"
        fi
    else
        echo "Skipping consolidation (only $CONSOLIDATE_COUNT UTXOs)"
    fi
else
    echo "2. Skipping consolidation (only $PUBLISHING_SPENT spent UTXOs)"
fi

echo ""

# 3. List active clients
echo "3. Active clients report..."
CLIENTS=$(curl -s -X GET "$API_URL/admin/clients/list" \
    -H "X-Admin-Password: $ADMIN_PASSWORD")

CLIENT_COUNT=$(echo "$CLIENTS" | jq -r '.clients | length')
echo "Total clients: $CLIENT_COUNT"

if [ "$CLIENT_COUNT" -gt 0 ]; then
    echo ""
    echo "Client Summary:"
    echo "$CLIENTS" | jq -r '.clients[] | "  \(.name): \(.tx_count)/\(.max_daily_tx) tx today (\(if .is_active then "active" else "inactive" end))"'
fi

echo ""

# 4. Check train status
echo "4. Checking train status..."
TRAIN_STATUS=$(curl -s -X GET "$API_URL/admin/emergency/status" \
    -H "X-Admin-Password: $ADMIN_PASSWORD")
echo "$TRAIN_STATUS" | jq '.'

RUNNING=$(echo "$TRAIN_STATUS" | jq -r '.running')
if [ "$RUNNING" = "true" ]; then
    echo "✓ Train is running"
else
    echo "⚠ Train is stopped!"
fi

echo ""

# 5. Database stats
echo "5. Database statistics..."
docker exec bsv_akua_db mongosh --quiet --eval '
    db = db.getSiblingDB("go-bsv");
    print("Collections:");
    print("  UTXOs: " + db.utxos.countDocuments());
    print("  Broadcast Requests: " + db.broadcast_requests.countDocuments());
    print("  Clients: " + db.clients.countDocuments());
    print("");
    print("Database size: " + (db.stats().dataSize / 1024 / 1024).toFixed(2) + " MB");
'

echo ""
echo "=================================================="
echo "Maintenance Complete: $(date '+%Y-%m-%d %H:%M:%S')"
echo "=================================================="
