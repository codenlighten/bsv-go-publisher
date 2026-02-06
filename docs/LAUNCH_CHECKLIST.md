# GovHash Launch Checklist

**Pre-Launch Verification for Enterprise Security Features**

---

## 🔒 Security Configuration

### 1. Environment Variables

- [ ] **ADMIN_PASSWORD set in .env**
  ```bash
  # Generate strong password:
  openssl rand -base64 32
  # Add to .env:
  ADMIN_PASSWORD=your_generated_password_here
  ```

- [ ] **Verify .env not in git**
  ```bash
  grep "^\.env$" .gitignore
  # Should return: .env
  ```

- [ ] **Production secrets secured**
  - [ ] ARC_TOKEN not committed to git
  - [ ] MONGO_URI password is strong
  - [ ] Private keys never logged

### 2. Database Indexes

- [ ] **Client collection indexed**
  ```bash
  docker exec bsv_akua_db mongosh --eval '
    db.getSiblingDB("go-bsv").clients.getIndexes()
  '
  # Should show index on api_key_hash (unique)
  ```

- [ ] **UTXO indexes optimal**
  ```bash
  docker exec bsv_akua_db mongosh --eval '
    db.getSiblingDB("go-bsv").utxos.getIndexes()
  '
  # Should show (status, type) compound index
  ```

---

## 🧪 Integration Testing

### 3. Health Endpoints

- [ ] **Basic health check**
  ```bash
  curl http://localhost:8080/health | jq
  # Should return: status="healthy", utxos stats
  ```

- [ ] **Admin stats endpoint**
  ```bash
  curl http://localhost:8080/admin/stats | jq
  # Should return: database stats, UTXO counts
  ```

### 4. Authentication Flow

- [ ] **Register test client**
  ```bash
  curl -X POST http://localhost:8080/admin/clients/register \
    -H "Content-Type: application/json" \
    -H "X-Admin-Password: $ADMIN_PASSWORD" \
    -d '{
      "name": "Test Client",
      "public_key": "02...",
      "max_daily_tx": 100
    }' | jq
  
  # Save the returned api_key - shown only once!
  ```

- [ ] **Test API key lookup**
  ```bash
  # Try publishing with X-API-Key header (will fail without signature)
  curl -X POST http://localhost:8080/publish \
    -H "Content-Type: application/json" \
    -H "X-API-Key: gh_your_test_key" \
    -d '{"data": "48656c6c6f"}' | jq
  
  # Should return: 401 "Missing X-Signature header"
  ```

- [ ] **Test signature verification** (use client example code)
  - [ ] Valid signature returns 200
  - [ ] Invalid signature returns 401
  - [ ] Wrong public key returns 401

### 5. Rate Limiting

- [ ] **Daily quota enforced**
  ```bash
  # Send max_daily_tx + 1 requests
  # Last request should return: 429 "Daily transaction limit exceeded"
  ```

- [ ] **Counter resets at midnight**
  ```bash
  # Check tx_count field in database
  docker exec bsv_akua_db mongosh --eval '
    db.getSiblingDB("go-bsv").clients.find({}, {name:1, tx_count:1, last_reset_date:1})
  '
  ```

### 6. Admin Endpoints

- [ ] **Client management works**
  ```bash
  # List clients
  curl -X GET http://localhost:8080/admin/clients/list \
    -H "X-Admin-Password: $ADMIN_PASSWORD" | jq
  
  # Deactivate client
  curl -X POST http://localhost:8080/admin/clients/<id>/deactivate \
    -H "X-Admin-Password: $ADMIN_PASSWORD" | jq
  
  # Verify client cannot publish when inactive
  ```

- [ ] **UTXO consolidation tested**
  ```bash
  # Estimate sweep
  curl -X GET "http://localhost:8080/admin/maintenance/estimate-sweep?utxo_type=publishing&max_inputs=10" \
    -H "X-Admin-Password: $ADMIN_PASSWORD" | jq
  
  # Execute sweep (test with small max_inputs first!)
  curl -X POST http://localhost:8080/admin/maintenance/sweep \
    -H "Content-Type: application/json" \
    -H "X-Admin-Password: $ADMIN_PASSWORD" \
    -d '{
      "dest_address": "1YourAddress...",
      "max_inputs": 10,
      "utxo_type": "publishing"
    }' | jq
  ```

- [ ] **Emergency stop works**
  ```bash
  # Stop train
  curl -X POST http://localhost:8080/admin/emergency/stop-train \
    -H "X-Admin-Password: $ADMIN_PASSWORD" | jq
  
  # Check status
  curl -X GET http://localhost:8080/admin/emergency/status \
    -H "X-Admin-Password: $ADMIN_PASSWORD" | jq
  # Should return: running=false
  
  # Restart server to resume
  docker-compose restart
  ```

---

## 📊 UTXO Pool Status

### 7. Publishing UTXO Verification

- [ ] **Current pool size adequate**
  ```bash
  curl http://localhost:8080/health | jq '.utxos.publishing_available'
  # Should be > 10,000 for production launch
  ```

- [ ] **Run Phase 2 split if needed** (if < 10k UTXOs)
  ```bash
  curl -X POST http://localhost:8080/admin/split-phase2 \
    -H "Content-Type: application/json" \
    -d '{
      "branch_count": 50,
      "leaf_count": 1000,
      "amount_per_leaf": 100
    }' | jq
  ```

### 8. Funding UTXO Balance

- [ ] **Funding address has sufficient balance**
  ```bash
  # Check funding address balance
  # Should have > 0.01 BSV for ongoing operations
  ```

---

## 🗂️ Backup & Recovery

### 9. Automated Backups

- [ ] **Backup script installed**
  ```bash
  chmod +x scripts/backup-mongodb.sh
  ./scripts/backup-mongodb.sh
  # Verify backup created in /backups/govhash/
  ```

- [ ] **Cron job configured**
  ```bash
  # Add to crontab:
  crontab -e
  
  # Every night at 2:00 AM
  0 2 * * * /home/greg/dev/go-bsv-akua-broadcast/scripts/backup-mongodb.sh >> /var/log/mongodb-backup.log 2>&1
  ```

- [ ] **Test restore procedure**
  ```bash
  chmod +x scripts/restore-mongodb.sh
  # Test on development environment first!
  ```

### 10. Weekly Maintenance

- [ ] **Maintenance script configured**
  ```bash
  chmod +x scripts/weekly-maintenance.sh
  
  # Set environment variables
  export ADMIN_PASSWORD="your_admin_password"
  export PUBLISHING_ADDRESS="your_publishing_address"
  export API_URL="https://api.govhash.org"
  
  # Test run
  ./scripts/weekly-maintenance.sh
  ```

- [ ] **Cron job for Sunday maintenance**
  ```bash
  # Add to crontab:
  crontab -e
  
  # Every Sunday at 3:00 AM
  0 3 * * 0 ADMIN_PASSWORD="xxx" PUBLISHING_ADDRESS="1xxx" API_URL="https://api.govhash.org" /home/greg/dev/go-bsv-akua-broadcast/scripts/weekly-maintenance.sh >> /var/log/govhash-maintenance.log 2>&1
  ```

---

## 🌐 Production Deployment

### 11. SSL/HTTPS Configuration

- [ ] **Let's Encrypt certificate valid**
  ```bash
  openssl s_client -connect api.govhash.org:443 -servername api.govhash.org < /dev/null 2>/dev/null | openssl x509 -noout -dates
  # Check expiry date
  ```

- [ ] **Auto-renewal configured**
  ```bash
  # Certbot renewal cron should exist
  cat /etc/cron.d/certbot
  ```

### 12. Nginx Configuration

- [ ] **Reverse proxy working**
  ```bash
  curl -I https://api.govhash.org/health
  # Should return: 200 OK
  ```

- [ ] **Rate limiting configured** (optional additional layer)
  ```nginx
  # In nginx config:
  limit_req_zone $binary_remote_addr zone=api_limit:10m rate=100r/s;
  ```

### 13. Firewall Rules

- [ ] **Port 8080 NOT exposed externally** (only via nginx)
  ```bash
  sudo ufw status
  # Should NOT show: 8080/tcp ALLOW
  ```

- [ ] **MongoDB port 27017 NOT exposed**
  ```bash
  sudo ufw status
  # Should NOT show: 27017/tcp ALLOW
  ```

- [ ] **Only ports 80, 443, 22 (SSH) exposed**
  ```bash
  sudo ufw status numbered
  ```

---

## 📝 Documentation

### 14. Client Onboarding Materials

- [ ] **Client Integration Guide ready**
  - [ ] examples/CLIENT_EXAMPLES.md reviewed
  - [ ] JavaScript example tested
  - [ ] Python example tested
  - [ ] Go example tested

- [ ] **"Client Kit" prepared** for first users:
  - [ ] Link to CLIENT_EXAMPLES.md
  - [ ] API endpoint URLs
  - [ ] Sample code snippets
  - [ ] Rate limit information
  - [ ] Support contact

### 15. Admin Documentation

- [ ] **docs/SECURITY.md reviewed**
- [ ] **docs/IMPLEMENTATION_SUMMARY.md reviewed**
- [ ] **docs/LAUNCH_CHECKLIST.md (this file) completed**

---

## 🚀 Go-Live Steps

### 16. Final Pre-Launch

- [ ] **All tests passed** (above checklist complete)
- [ ] **UTXO pool > 10,000** publishing UTXOs
- [ ] **ADMIN_PASSWORD set** and secure
- [ ] **Backups automated** and tested
- [ ] **Monitoring enabled** (health checks, logs)

### 17. Launch Day

1. [ ] **Create first production client**
   ```bash
   curl -X POST https://api.govhash.org/admin/clients/register \
     -H "Content-Type: application/json" \
     -H "X-Admin-Password: $ADMIN_PASSWORD" \
     -d '{
       "name": "Production Client 1",
       "public_key": "02...",
       "site_origin": "govhash.org",
       "max_daily_tx": 1000
     }'
   ```

2. [ ] **Send test transaction** from production client
3. [ ] **Verify on blockchain** (WhatsOnChain)
4. [ ] **Monitor logs** for errors
   ```bash
   docker logs -f bsv_akua_server
   ```

5. [ ] **Update STATUS.md** to "LIVE WITH SECURITY"
6. [ ] **Announce launch** on GovHash.org

### 18. Post-Launch Monitoring (First 24 Hours)

- [ ] **Check health every hour**
  ```bash
  watch -n 3600 'curl -s https://api.govhash.org/health | jq'
  ```

- [ ] **Monitor UTXO pool depletion rate**
- [ ] **Review client transaction counts**
- [ ] **Check for authentication failures** in logs
- [ ] **Verify train is processing batches**

---

## 📞 Emergency Contacts

**If issues arise:**

1. **Stop train immediately:**
   ```bash
   curl -X POST https://api.govhash.org/admin/emergency/stop-train \
     -H "X-Admin-Password: $ADMIN_PASSWORD"
   ```

2. **Check logs:**
   ```bash
   docker logs --tail=100 bsv_akua_server
   ```

3. **Disable problematic client:**
   ```bash
   curl -X POST https://api.govhash.org/admin/clients/<id>/deactivate \
     -H "X-Admin-Password: $ADMIN_PASSWORD"
   ```

4. **Rollback if needed:**
   ```bash
   docker-compose down
   ./scripts/restore-mongodb.sh /backups/govhash/latest.archive.gz
   docker-compose up -d
   ```

---

## ✅ Sign-Off

**Completed by:** ___________________________  
**Date:** ___________________________  
**Production URL:** https://api.govhash.org  
**Launch Status:** ⬜ Ready for Production

---

**Notes:**

_Use this space to document any launch-specific notes, deviations from the checklist, or special considerations for your deployment._
