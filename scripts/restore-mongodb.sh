#!/bin/bash
#
# MongoDB Restore Script for GovHash/NotaryHash
# Restores from backup archive
#
# Usage: ./restore-mongodb.sh <backup-file>
# Example: ./restore-mongodb.sh /backups/govhash/govhash_2026-02-06.archive.gz

set -e

CONTAINER_NAME="${CONTAINER_NAME:-bsv_akua_db}"

if [ -z "$1" ]; then
    echo "Usage: $0 <backup-file>"
    echo ""
    echo "Available backups:"
    ls -lh /backups/govhash/*.archive.gz 2>/dev/null || echo "  No backups found in /backups/govhash"
    exit 1
fi

BACKUP_FILE="$1"

if [ ! -f "$BACKUP_FILE" ]; then
    echo "Error: Backup file not found: $BACKUP_FILE"
    exit 1
fi

echo "=================================================="
echo "MongoDB Restore"
echo "=================================================="
echo "WARNING: This will REPLACE all data in the database!"
echo "Backup file: $BACKUP_FILE"
echo "Container: $CONTAINER_NAME"
echo ""
read -p "Are you sure you want to continue? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "Restore cancelled."
    exit 0
fi

echo ""
echo "Starting restore..."

# Restore backup
cat "$BACKUP_FILE" | docker exec -i "$CONTAINER_NAME" sh -c 'mongorestore --archive --gzip --drop'

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ Restore completed successfully!"
    echo ""
    echo "Verification:"
    docker exec "$CONTAINER_NAME" mongosh --quiet --eval 'db.getSiblingDB("go-bsv").stats()' | head -20
else
    echo ""
    echo "✗ Restore failed!"
    exit 1
fi
