# GovHash Operations Guide - Internal Team

**Production URL:** https://api.govhash.org  
**Admin Portal:** https://api.govhash.org/admin  
**Server:** 134.209.4.149 (DigitalOcean)  
**Last Updated:** February 8, 2026

---

## Quick Access

### Admin Portal Login
- **URL:** https://api.govhash.org/admin
- **Password:** Check `.env` file (ADMIN_PASSWORD)
- **Features:** Live metrics, client management, UTXO monitoring, emergency controls

### SSH Access
```bash
ssh root@134.209.4.149
```

### Container Management
```bash
# View running containers
docker ps | grep bsv

# Check logs
docker logs bsv_akua_server -f
docker logs bsv_admin_portal -f
docker logs bsv_akua_db -f

# Restart services
docker-compose restart
```

---

## Daily Operations

### 1. Morning Health Check (5 minutes)

```bash
# Check system health
curl https://api.govhash.org/health | jq

# Expected output:
# - status: "healthy"
# - publishing_available: > 40,000
# - queueDepth: < 100

# Check admin stats
curl https://api.govhash.org/admin/stats \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq

# Review metrics:
# - broadcasts24h: normal range 0-10,000
# - avgLatencyMs: < 5000
# - throughput: distributed across time buckets
```

### 2. Client Management

**Register New Client:**
```bash
curl -X POST https://api.govhash.org/admin/clients/register \
  -H "Content-Type: application/json" \
  -H "X-Admin-Password: $ADMIN_PASSWORD" \
  -d '{
    "name": "Acme Corp",
    "tier": "enterprise",
    "public_key": "02abc123...",
    "site_origin": "https://acmecorp.com",
    "max_daily_tx": 10000
  }' | jq

# IMPORTANT: Save the returned API key immediately!
# It is shown only once and cannot be recovered.
```

**List All Clients:**
```bash
curl https://api.govhash.org/admin/clients/list \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq
```

**Update Client Security Tier:**
```bash
curl -X POST https://api.govhash.org/admin/clients/{id}/update-security \
  -H "Content-Type: application/json" \
  -H "X-Admin-Password: $ADMIN_PASSWORD" \
  -d '{
    "tier": "government",
    "max_daily_tx": 100000,
    "require_signature": true
  }' | jq
```

**Deactivate Client:**
```bash
curl -X POST https://api.govhash.org/admin/clients/{id}/deactivate \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq
```

### 3. UTXO Pool Monitoring

**Check UTXO Status:**
```bash
curl https://api.govhash.org/admin/stats \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq '.utxos'
```

**Alert Thresholds:**
- **Critical:** publishing_available < 10,000 (immediate action)
- **Warning:** publishing_available < 25,000 (plan consolidation)
- **Healthy:** publishing_available > 40,000

**Consolidate UTXOs (if < 10,000 available):**
```bash
# Estimate consolidation
curl https://api.govhash.org/admin/maintenance/estimate-sweep \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq

# Execute sweep (consolidate all publishing UTXOs)
curl -X POST https://api.govhash.org/admin/maintenance/sweep \
  -H "Content-Type: application/json" \
  -H "X-Admin-Password: $ADMIN_PASSWORD" \
  -d '{
    "dest_address": "12w4BoPtqCt7EFLmUPi9GLmpbZ1CHdPvzj",
    "max_inputs": 1000,
    "utxo_type": "publishing"
  }' | jq
```

---

## Weekly Maintenance (Sundays 3:00 AM UTC)

### Automated UTXO Consolidation

**Cron Job (already configured on server):**
```bash
0 3 * * 0 /home/greg/dev/go-bsv-akua-broadcast/scripts/weekly-maintenance.sh >> /var/log/govhash-maintenance.log 2>&1
```

**Manual Execution (if needed):**
```bash
ssh root@134.209.4.149
cd /home/greg/dev/go-bsv-akua-broadcast
ADMIN_PASSWORD="..." PUBLISHING_ADDRESS="..." ./scripts/weekly-maintenance.sh
```

**What It Does:**
1. Checks UTXO pool health
2. Consolidates change UTXOs if > 1,000
3. Splits large UTXOs if publishing pool < 40,000
4. Logs all operations to MongoDB

---

## Emergency Procedures

### 1. Service Degradation (High Queue Depth)

**Symptoms:** `/health` shows queueDepth > 500

**Actions:**
```bash
# Check train status
curl https://api.govhash.org/admin/emergency/status \
  -H "X-Admin-Password: $ADMIN_PASSWORD"

# If train stopped, restart containers
ssh root@134.209.4.149
docker-compose restart

# Monitor queue drain
watch -n 5 'curl -s https://api.govhash.org/health | jq .queueDepth'
```

### 2. UTXO Pool Depletion

**Symptoms:** publishing_available < 1,000

**Immediate Actions:**
```bash
# Stop accepting new requests temporarily
curl -X POST https://api.govhash.org/admin/emergency/stop-train \
  -H "X-Admin-Password: $ADMIN_PASSWORD"

# Emergency UTXO split from funding pool
# (Contact Gregory Ward immediately)

# Resume after UTXOs replenished
docker-compose restart
```

### 3. Total System Failure

**Symptoms:** `/health` returns 500 or timeouts

**Recovery Steps:**
```bash
ssh root@134.209.4.149

# Check container status
docker ps -a | grep bsv

# Check logs for errors
docker logs bsv_akua_server --tail 100

# Restart all services
cd /home/greg/dev/go-bsv-akua-broadcast
docker-compose down
docker-compose up -d

# Wait 30 seconds for startup
sleep 30

# Verify recovery
curl https://api.govhash.org/health
```

### 4. Database Corruption

**Recovery from Backup:**
```bash
ssh root@134.209.4.149
cd /home/greg/dev/go-bsv-akua-broadcast/backups

# List available backups
ls -lh govhash-backup-*.tar.gz

# Restore latest backup
BACKUP_FILE="govhash-backup-YYYY-MM-DD.tar.gz"
tar xzf $BACKUP_FILE
mongorestore --host localhost:27017 --db go-bsv dump/go-bsv/

# Restart server to clear cache
docker-compose restart
```

---

## Monitoring & Alerts

### Key Metrics to Track

**System Health:**
- `/health` status (should always be "healthy")
- Queue depth (normal: 0-100, alert: > 500)
- Publishing UTXOs (critical: < 10,000)

**Performance:**
- Average latency (normal: 2-5 seconds)
- Broadcasts per 24h (trend analysis)
- Throughput distribution (detect traffic patterns)

**Clients:**
- Active client count
- Daily transaction usage per client
- Clients approaching rate limits (txCount/maxDailyTx > 80%)

### Recommended Monitoring Tools

**1. Uptime Monitoring (External):**
- Service: UptimeRobot or Pingdom
- Check: `https://api.govhash.org/health` every 5 minutes
- Alert: Email + SMS if down for > 2 minutes

**2. Log Aggregation:**
```bash
# Real-time log monitoring
ssh root@134.209.4.149
docker logs bsv_akua_server -f | grep -E "ERROR|WARN|Emergency"
```

**3. Resource Monitoring:**
```bash
# Server resources
ssh root@134.209.4.149
htop  # CPU/Memory usage
df -h  # Disk space
docker stats  # Container resources
```

---

## Security Best Practices

### 1. Admin Password Rotation (Quarterly)

```bash
# Generate new password
openssl rand -base64 32

# Update .env on server
ssh root@134.209.4.149
nano /home/greg/dev/go-bsv-akua-broadcast/.env
# Update ADMIN_PASSWORD=...

# Restart containers
docker-compose restart

# Update team password manager
# Update admin portal login credentials
```

### 2. Client API Key Rotation (Annually)

```bash
# For each client:
# 1. Generate new API key (register new client temporarily)
# 2. Notify client with 30-day migration window
# 3. Deactivate old key after migration
# 4. Remove old client record
```

### 3. SSL Certificate Renewal

```bash
# Certificates auto-renew via Let's Encrypt
# Manual renewal if needed:
ssh root@134.209.4.149
certbot renew --nginx
nginx -s reload
```

### 4. Database Backups

**Automated Daily Backups:**
```bash
# Cron job (already configured):
0 2 * * * /home/greg/dev/go-bsv-akua-broadcast/scripts/backup-database.sh

# Verify backups exist
ssh root@134.209.4.149
ls -lh /home/greg/dev/go-bsv-akua-broadcast/backups/

# Retention: Keep 7 daily, 4 weekly, 3 monthly
```

---

## Performance Optimization

### 1. UTXO Pool Management

**Target Pool Sizes:**
- Funding UTXOs: 50-100 (large amounts for splitting)
- Publishing UTXOs: 40,000-50,000 (100 sats each)
- Change UTXOs: < 500 (consolidate weekly)

**Splitting Strategy:**
```bash
# When publishing pool drops below 40,000:
# 1. Identify largest funding UTXO
# 2. Split into 50,000 x 100-sat outputs
# 3. Takes ~5 minutes to process
# 4. Monitor via /admin/stats
```

### 2. Train Configuration

**Current Settings (optimal):**
- Interval: 3 seconds (balance latency vs. ARC rate limits)
- Batch size: 1,000 tx max (ARC V1 limit)
- Queue capacity: 10,000 items

**Tuning (if needed):**
```bash
# Edit .env
TRAIN_INTERVAL=3s  # Lower = faster but more ARC calls
TRAIN_MAX_BATCH=1000  # Don't exceed ARC limits

# Restart to apply
docker-compose restart
```

### 3. Database Indexing

**Existing Indexes (do not modify):**
- `utxos`: status, type, locked_at
- `broadcast_requests`: uuid, status, created_at
- `clients`: api_key_hash, is_active

---

## Troubleshooting Common Issues

### Issue: "Request queued but never broadcasts"

**Diagnosis:**
```bash
curl https://api.govhash.org/admin/emergency/status \
  -H "X-Admin-Password: $ADMIN_PASSWORD"
```

**If `running: false`:**
- Train worker crashed, restart containers
- Check logs for panic/error

**If `running: true` but queue not draining:**
- Check ARC connectivity: `curl https://arc.gorillapool.io`
- Verify ARC_TOKEN in .env is valid
- Check server logs for ARC errors

---

### Issue: "High latency (>10 seconds average)"

**Possible Causes:**
1. **ARC network congestion** - Check arc.gorillapool.io status
2. **UTXO locking contention** - If broadcasting > 1,000 tx/second
3. **Train interval too high** - Reduce from 3s to 2s temporarily

**Actions:**
```bash
# Check current queue depth
curl https://api.govhash.org/health | jq .queueDepth

# If > 500, temporarily reduce train interval
ssh root@134.209.4.149
nano .env  # Set TRAIN_INTERVAL=2s
docker-compose restart
```

---

### Issue: "Client signature verification failing"

**Client-Side Checklist:**
1. Using double SHA-256 (not single)
2. Signing raw hex data (not JSON)
3. DER-encoded signature output
4. Timestamp within 5 minutes of server time
5. Public key matches registered key

**Server-Side Verification:**
```bash
# Check client's registered public key
curl https://api.govhash.org/admin/clients/list \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | \
  jq '.clients[] | select(.name=="ClientName") | .publicKey'

# Review recent failed requests in logs
ssh root@134.209.4.149
docker logs bsv_akua_server | grep "Signature verification failed"
```

---

## Deployment & Updates

### Rolling Update (Zero Downtime)

```bash
ssh root@134.209.4.149
cd /home/greg/dev/go-bsv-akua-broadcast

# Pull latest code
git pull origin main

# Rebuild containers
docker-compose build

# Restart with health check
docker-compose up -d

# Verify health
sleep 10
curl https://api.govhash.org/health
```

### Frontend Admin Portal Update

```bash
# Build frontend locally
cd /home/greg/dev/go-bsv-akua-broadcast/frontend
npm run build

# Deploy to server
tar czf /tmp/frontend.tar.gz dist/
scp /tmp/frontend.tar.gz root@134.209.4.149:/tmp/

# Extract on server
ssh root@134.209.4.149
cd /tmp && tar xzf frontend.tar.gz
docker cp dist/. bsv_admin_portal:/usr/share/nginx/html/
```

---

## Contact Information

### Technical Team
- **Lead Developer:** Gregory Ward
- **On-Call Rotation:** Check team calendar
- **Emergency Escalation:** +1 (555) 123-4567

### Service Providers
- **Hosting:** DigitalOcean (support@digitalocean.com)
- **ARC Provider:** GorillaPool (support@gorillapool.io)
- **DNS/SSL:** Cloudflare (via Let's Encrypt)

---

## Useful Commands Reference

```bash
# Quick health check
curl https://api.govhash.org/health | jq

# Check active clients
curl https://api.govhash.org/admin/clients/list \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq '.clients | length'

# Monitor queue in real-time
watch -n 2 'curl -s https://api.govhash.org/health | jq .queueDepth'

# Check UTXO pool
curl https://api.govhash.org/admin/stats \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | \
  jq '.utxos.publishing_available'

# Emergency stop train
curl -X POST https://api.govhash.org/admin/emergency/stop-train \
  -H "X-Admin-Password: $ADMIN_PASSWORD"

# View server logs
ssh root@134.209.4.149 'docker logs bsv_akua_server --tail 50 -f'

# Database backup
ssh root@134.209.4.149
cd /home/greg/dev/go-bsv-akua-broadcast
./scripts/backup-database.sh

# UTXO consolidation
ssh root@134.209.4.149
cd /home/greg/dev/go-bsv-akua-broadcast
ADMIN_PASSWORD="..." ./scripts/weekly-maintenance.sh
```

---

## Runbook Summary

| Scenario | Action | Time to Resolve |
|----------|--------|-----------------|
| Queue depth > 500 | Check train status, restart if needed | 2 minutes |
| Publishing UTXOs < 10K | Emergency consolidation + split | 15 minutes |
| System down | Docker restart + verify health | 3 minutes |
| High latency | Check ARC, reduce train interval | 5 minutes |
| Client signature failing | Verify public key, check timestamp | 10 minutes |
| Database corruption | Restore from backup | 20 minutes |

---

**Questions?** Contact Gregory Ward or check internal Slack #govhash-ops channel.
