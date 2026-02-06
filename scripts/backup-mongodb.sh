#!/bin/bash
#
# MongoDB Backup Script for GovHash/NotaryHash
# Creates compressed backup archive with timestamp
#
# Usage: ./backup-mongodb.sh
# Cron: 0 2 * * * /path/to/backup-mongodb.sh >> /var/log/mongodb-backup.log 2>&1

set -e

# Configuration
BACKUP_DIR="${BACKUP_DIR:-$HOME/backups/govhash}"
CONTAINER_NAME="${CONTAINER_NAME:-bsv_akua_db}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
MONGO_PASSWORD="${MONGO_PASSWORD}"
DATE=$(date +%F)
TIMESTAMP=$(date +%F_%H-%M-%S)

# Get password from .env if not set
if [ -z "$MONGO_PASSWORD" ] && [ -f .env ]; then
    MONGO_PASSWORD=$(grep "^MONGO_PASSWORD=" .env | cut -d'=' -f2)
fi

if [ -z "$MONGO_PASSWORD" ]; then
    echo "Error: MONGO_PASSWORD not set"
    echo "Set via environment variable or ensure .env file exists"
    exit 1
fi

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

echo "=================================================="
echo "MongoDB Backup Started: $TIMESTAMP"
echo "=================================================="

# Create backup
BACKUP_FILE="$BACKUP_DIR/govhash_$TIMESTAMP.archive.gz"

echo "Creating backup: $BACKUP_FILE"
docker exec -e MONGO_PASSWORD="$MONGO_PASSWORD" "$CONTAINER_NAME" sh -c 'mongodump --uri="mongodb://root:$MONGO_PASSWORD@localhost:27017/go-bsv?authSource=admin" --archive --gzip' > "$BACKUP_FILE"

if [ $? -eq 0 ]; then
    BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
    echo "✓ Backup created successfully: $BACKUP_SIZE"
    
    # Create symlink to latest backup
    ln -sf "$(basename "$BACKUP_FILE")" "$BACKUP_DIR/latest.archive.gz"
    echo "✓ Updated latest backup symlink"
else
    echo "✗ Backup failed!"
    exit 1
fi

# Cleanup old backups (older than RETENTION_DAYS)
echo "Cleaning up backups older than $RETENTION_DAYS days..."
find "$BACKUP_DIR" -name "govhash_*.archive.gz" -type f -mtime +$RETENTION_DAYS -delete
REMAINING=$(find "$BACKUP_DIR" -name "govhash_*.archive.gz" -type f | wc -l)
echo "✓ Cleanup complete. $REMAINING backups retained."

# Statistics
echo ""
echo "Backup Statistics:"
echo "  Location: $BACKUP_DIR"
echo "  Size: $BACKUP_SIZE"
echo "  Total Backups: $REMAINING"
df -h "$BACKUP_DIR" | tail -1 | awk '{print "  Disk Usage: " $3 "/" $2 " (" $5 ")"}'

echo ""
echo "=================================================="
echo "MongoDB Backup Completed: $(date +%F_%H-%M-%S)"
echo "=================================================="

# Optional: Upload to S3 or remote storage
# Uncomment and configure if needed:
# if command -v aws &> /dev/null; then
#     aws s3 cp "$BACKUP_FILE" "s3://your-bucket/govhash-backups/" --storage-class STANDARD_IA
#     echo "✓ Backup uploaded to S3"
# fi
