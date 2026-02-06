# GovHash Quick Reference

**Essential commands and procedures for daily operations**

---

## 🔍 Health Monitoring

```bash
# Check system health
curl -s https://api.govhash.org/health | jq

# Watch health continuously
watch -n 10 'curl -s https://api.govhash.org/health | jq'

# Check current UTXO count
curl -s https://api.govhash.org/health | jq '.utxos.publishing_available'
```

---

## 👥 Client Management

### Register New Client
```bash
curl -X POST https://api.govhash.org/admin/clients/register \
  -H "Content-Type: application/json" \
  -H "X-Admin-Password: $ADMIN_PASSWORD" \
  -d '{
    "name": "Acme Corp",
    "public_key": "02abc123...",
    "site_origin": "acme.com",
    "max_daily_tx": 1000
  }' | jq

# ⚠️ IMPORTANT: Save the returned API key - shown only once!
```

### List All Clients
```bash
curl -s -X GET https://api.govhash.org/admin/clients/list \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq
```

### Deactivate Client
```bash
curl -X POST https://api.govhash.org/admin/clients/<client_id>/deactivate \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq
```

### Activate Client
```bash
curl -X POST https://api.govhash.org/admin/clients/<client_id>/activate \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq
```

---

## 🧹 UTXO Maintenance

### Estimate Consolidation Value
```bash
# Check how many UTXOs can be consolidated
curl -s -X GET "https://api.govhash.org/admin/maintenance/estimate-sweep?utxo_type=publishing&max_inputs=100" \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq
```

### Consolidate Publishing UTXOs
```bash
curl -X POST https://api.govhash.org/admin/maintenance/sweep \
  -H "Content-Type: application/json" \
  -H "X-Admin-Password: $ADMIN_PASSWORD" \
  -d '{
    "dest_address": "1YourPublishingAddress...",
    "max_inputs": 100,
    "utxo_type": "publishing"
  }' | jq

# Returns txid - check on WhatsOnChain: https://whatsonchain.com/tx/<txid>
```

### Consolidate Change UTXOs (Dust)
```bash
curl -X POST https://api.govhash.org/admin/maintenance/consolidate-dust \
  -H "Content-Type: application/json" \
  -H "X-Admin-Password: $ADMIN_PASSWORD" \
  -d '{
    "funding_address": "1YourFundingAddress...",
    "max_inputs": 100
  }' | jq
```

---

## 🚨 Emergency Procedures

### Stop Train Worker
```bash
# Stops processing new batches (current batch completes)
curl -X POST https://api.govhash.org/admin/emergency/stop-train \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq

# To resume: restart the server
docker-compose restart
```

### Check Train Status
```bash
curl -s -X GET https://api.govhash.org/admin/emergency/status \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq '.running'
```

---

## 💾 Backup & Restore

### Manual Backup
```bash
cd /home/greg/dev/go-bsv-akua-broadcast
./scripts/backup-mongodb.sh

# Backup saved to: /backups/govhash/govhash_YYYY-MM-DD_HH-MM-SS.archive.gz
```

### Restore from Backup
```bash
# ⚠️ THIS WILL REPLACE ALL DATA!
cd /home/greg/dev/go-bsv-akua-broadcast
./scripts/restore-mongodb.sh /backups/govhash/govhash_2026-02-06.archive.gz

# Or restore latest:
./scripts/restore-mongodb.sh /backups/govhash/latest.archive.gz
```

---

## 📊 Database Queries

### Check Client Transaction Counts
```bash
docker exec bsv_akua_db mongosh --quiet --eval '
  db.getSiblingDB("go-bsv").clients.find(
    {},
    {name:1, tx_count:1, max_daily_tx:1, last_reset_date:1, is_active:1}
  ).pretty()
'
```

### Check Recent Broadcasts
```bash
docker exec bsv_akua_db mongosh --quiet --eval '
  db.getSiblingDB("go-bsv").broadcast_requests.find().sort({created_at:-1}).limit(10).pretty()
'
```

### UTXO Statistics
```bash
docker exec bsv_akua_db mongosh --quiet --eval '
  db = db.getSiblingDB("go-bsv");
  print("Publishing Available: " + db.utxos.countDocuments({status: "available", type: "publishing"}));
  print("Publishing Spent: " + db.utxos.countDocuments({status: "spent", type: "publishing"}));
  print("Funding Available: " + db.utxos.countDocuments({status: "available", type: "funding"}));
'
```

---

## 🐳 Docker Operations

### View Server Logs
```bash
docker logs -f $(docker ps --filter "name=server" -q)

# Or with docker-compose:
docker-compose logs -f
```

### Restart Services
```bash
cd /home/greg/dev/go-bsv-akua-broadcast
docker-compose restart
```

### Rebuild After Code Changes
```bash
cd /home/greg/dev/go-bsv-akua-broadcast
docker-compose down
make build
docker-compose up -d
```

### Check Container Status
```bash
docker-compose ps
```

---

## 📈 Performance Monitoring

### Check Train Queue Depth
```bash
curl -s https://api.govhash.org/health | jq '.queueDepth'
# Should be 0 or low (<10) normally
```

### Monitor Train Batches (Live)
```bash
docker logs -f $(docker ps --filter "name=server" -q) | grep "Train departing"
```

### Check Broadcast Success Rate
```bash
docker exec bsv_akua_db mongosh --quiet --eval '
  db = db.getSiblingDB("go-bsv");
  total = db.broadcast_requests.countDocuments();
  success = db.broadcast_requests.countDocuments({status: "success"});
  print("Total: " + total);
  print("Success: " + success);
  print("Rate: " + (success/total*100).toFixed(2) + "%");
'
```

---

## 🔐 Security Checks

### Verify Admin Password Not in Git
```bash
grep -r "ADMIN_PASSWORD" /home/greg/dev/go-bsv-akua-broadcast/.git/ 2>/dev/null
# Should return nothing
```

### Check SSL Certificate Expiry
```bash
echo | openssl s_client -connect api.govhash.org:443 -servername api.govhash.org 2>/dev/null | openssl x509 -noout -dates
```

### Verify Firewall Rules
```bash
sudo ufw status numbered
# Should NOT expose port 8080 or 27017 externally
```

---

## 🛠️ Weekly Maintenance

Run every Sunday at 3 AM (or manually):

```bash
cd /home/greg/dev/go-bsv-akua-broadcast

export ADMIN_PASSWORD="your_admin_password"
export PUBLISHING_ADDRESS="your_publishing_address"
export API_URL="https://api.govhash.org"

./scripts/weekly-maintenance.sh
```

This script:
- Checks system health
- Consolidates UTXOs if needed (>100 spent)
- Lists active clients
- Verifies train status
- Shows database statistics

---

## 📝 Useful Aliases

Add to `~/.bashrc`:

```bash
# GovHash aliases
alias govhash-health='curl -s https://api.govhash.org/health | jq'
alias govhash-logs='docker logs -f $(docker ps --filter "name=server" -q)'
alias govhash-clients='curl -s https://api.govhash.org/admin/clients/list -H "X-Admin-Password: $ADMIN_PASSWORD" | jq'
alias govhash-backup='cd /home/greg/dev/go-bsv-akua-broadcast && ./scripts/backup-mongodb.sh'
alias govhash-utxos='curl -s https://api.govhash.org/health | jq ".utxos"'

# Then run: source ~/.bashrc
```

---

## 🆘 Common Issues

### Issue: Health shows "unhealthy"
**Solution:** Check if MongoDB is running
```bash
docker-compose ps
# If mongo is down:
docker-compose up -d
```

### Issue: Publishing UTXO pool depleted
**Solution:** Run Phase 2 split
```bash
curl -X POST https://api.govhash.org/admin/split-phase2 \
  -H "Content-Type: application/json" \
  -d '{"branch_count": 50, "leaf_count": 1000, "amount_per_leaf": 100}'
```

### Issue: Client hitting rate limit
**Solution:** Increase their daily quota
```bash
# Update in MongoDB directly:
docker exec bsv_akua_db mongosh --quiet --eval '
  db.getSiblingDB("go-bsv").clients.updateOne(
    {name: "Client Name"},
    {$set: {max_daily_tx: 5000}}
  )
'
```

### Issue: Train stopped
**Solution:** Check status and restart
```bash
curl -s https://api.govhash.org/admin/emergency/status -H "X-Admin-Password: $ADMIN_PASSWORD"
docker-compose restart
```

---

## 📞 Support

- **Documentation:** [docs/SECURITY.md](docs/SECURITY.md)
- **Client Examples:** [examples/CLIENT_EXAMPLES.md](examples/CLIENT_EXAMPLES.md)
- **Launch Checklist:** [docs/LAUNCH_CHECKLIST.md](docs/LAUNCH_CHECKLIST.md)
- **Email:** support@govhash.org
