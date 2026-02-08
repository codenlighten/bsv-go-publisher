# 🎮 Admin Quick Reference - Tier Management

## Register New Clients

### Pilot Tier (Testing, Zero Friction)
```bash
curl -X POST https://api.govhash.org/admin/clients/register \
  -H "X-Admin-Password: ***" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "AKUA Pilot",
    "tier": "pilot",
    "max_daily_tx": 10000
  }'
```
**Result:** Client can use API with API key only (no signature)

---

### Enterprise Tier (Commercial, Full Security)
```bash
curl -X POST https://api.govhash.org/admin/clients/register \
  -H "X-Admin-Password: ***" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "NotaryHash Enterprise",
    "tier": "enterprise",
    "public_key": "04a1b2c3...",
    "max_daily_tx": 100000
  }'
```
**Result:** Client MUST sign all requests with ECDSA

---

### Government Tier (Institutional, Maximum Security)
```bash
curl -X POST https://api.govhash.org/admin/clients/register \
  -H "X-Admin-Password: ***" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GovHash Agency",
    "tier": "government",
    "public_key": "04d4e5f6...",
    "allowed_ips": ["203.0.113.0/24", "198.51.100.5"],
    "max_daily_tx": 0
  }'
```
**Result:** Client MUST sign + requests only from whitelisted IPs

---

## Upgrade/Downgrade Tiers

### Upgrade Pilot → Enterprise
```bash
# Get client ID first
curl https://api.govhash.org/admin/clients/list \
  -H "X-Admin-Password: ***"

# Then upgrade
curl -X PATCH https://api.govhash.org/admin/clients/507f1f77bcf86cd799439011/security \
  -H "X-Admin-Password: ***" \
  -H "Content-Type: application/json" \
  -d '{
    "tier": "enterprise",
    "require_signature": true,
    "grace_period_hours": 48
  }'
```

---

### Downgrade Enterprise → Pilot (Emergency Fallback)
```bash
curl -X PATCH https://api.govhash.org/admin/clients/507f1f77bcf86cd799439011/security \
  -H "X-Admin-Password: ***" \
  -H "Content-Type: application/json" \
  -d '{
    "tier": "pilot",
    "require_signature": false
  }'
```

---

## Update Security Settings

### Add IP Whitelist (Pilot Tier)
```bash
curl -X PATCH https://api.govhash.org/admin/clients/:id/security \
  -H "X-Admin-Password: ***" \
  -H "Content-Type: application/json" \
  -d '{
    "allowed_ips": ["10.0.0.0/8", "172.16.0.0/12"]
  }'
```

---

### Extend Grace Period (Key Rotation Support)
```bash
curl -X PATCH https://api.govhash.org/admin/clients/:id/security \
  -H "X-Admin-Password: ***" \
  -H "Content-Type: application/json" \
  -d '{
    "grace_period_hours": 72
  }'
```
**Use Case:** Client has distributed systems, needs more time to roll out new keys

---

## List All Clients
```bash
curl https://api.govhash.org/admin/clients/list \
  -H "X-Admin-Password: ***"
```

---

## Tier Matrix Quick Reference

| Tier | Auth | Rate Limit | Grace Period | Best For |
|------|------|------------|--------------|----------|
| **Pilot** | Key Only | 10/min | 0h | Testing, MVPs |
| **Enterprise** | Key + ECDSA | 100/min | 24h | Production Apps |
| **Government** | Key + ECDSA + IP | ∞ | 168h | Agencies, Banks |

---

## Common Workflows

### New Pilot Client (AKUA Pattern)
1. **Register:** `POST /admin/clients/register` with `"tier": "pilot"`
2. **Test:** Client uses API with just `X-API-Key` header
3. **Upgrade:** When ready, `PATCH /.../security` to enterprise
4. **Register Key:** Client calls `POST /auth/register-public-key`
5. **Production:** Client now signs all requests

---

### Immediate Enterprise (NotaryHash Pattern)
1. **Client generates keys locally** (never share private key)
2. **Client sends public key** to admin via secure channel
3. **Register:** `POST /admin/clients/register` with tier=enterprise + public_key
4. **Production:** Client signs requests from day one

---

### Emergency Signature Bypass
If client loses private key and needs immediate access:
```bash
# Temporarily disable signature requirement
PATCH /admin/clients/:id/security
  {"require_signature": false}

# Client can now access API with key only
# Have client generate new keys ASAP
# Then re-enable signature requirement
```

---

## Monitoring Tier Usage

Check server logs for tier-based authentication:
```bash
# Pilot tier requests
grep "\[PILOT\]" /var/log/bsv-broadcaster.log

# Enterprise/Government tier (signature verified)
grep "✅ Signature verified" /var/log/bsv-broadcaster.log

# Grace period usage (key rotation)
grep "🔄 Trying old key" /var/log/bsv-broadcaster.log
```

---

## Security Best Practices

### DO:
✅ Start new clients on pilot tier for testing  
✅ Upgrade to enterprise when ready for production  
✅ Use government tier for institutions with compliance requirements  
✅ Set appropriate grace periods (24h standard, 168h for large systems)  
✅ Use IP whitelists for pilot clients with static IPs  

### DON'T:
❌ Store client private keys on the server  
❌ Downgrade production clients to pilot without reason  
❌ Set grace periods < 1 hour (too short for distributed systems)  
❌ Use pilot tier for production without IP whitelisting  
❌ Share admin password via insecure channels  

---

**Need Help?** Check [PHASE_5_COMPLETE.md](./PHASE_5_COMPLETE.md) for detailed examples
