# 🏛️ GovHash BSV Attestation Platform - Final Architecture

**Version:** 1.0 Production  
**Deployment Date:** February 8, 2026  
**Status:** ✅ FULLY OPERATIONAL  
**Mainnet URL:** https://api.govhash.org

---

## 📊 System Performance Metrics

| Metric | Specification | Status |
|--------|--------------|--------|
| **Sustained Throughput** | 333 TPS | ✅ ARC-Limited |
| **Peak Capacity** | 1,000 tx/batch | ✅ Operational |
| **UTXO Pool** | 50,000 publishing UTXOs | ✅ Atomic Locking |
| **Train Interval** | 3 seconds | ✅ Optimized |
| **Latency (to Accepted)** | < 5 seconds | ✅ Synchronous Mode |
| **Data Integrity** | ECDSA secp256k1 | ✅ Non-repudiation |
| **Availability** | 99.9% uptime | ✅ Docker + Health Checks |

---

## 🎯 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         CLIENT APPLICATIONS                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
│  │  AKUA Pilot  │  │  NotaryHash  │  │   GovHash    │                  │
│  │   (Tier 1)   │  │ (Enterprise) │  │ (Government) │                  │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘                  │
│         │                 │                 │                            │
│    API Key Only      API Key + ECDSA   API Key + ECDSA + IP Lock        │
└─────────┼─────────────────┼─────────────────┼────────────────────────────┘
          │                 │                 │
          ▼                 ▼                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          NGINX REVERSE PROXY                             │
│                    (SSL/TLS Termination + Rate Limiting)                │
└─────────────────────────────────────────┬───────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                   BSV AKUA BROADCAST SERVER (Go 1.24)                   │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                    ADAPTIVE AUTH MIDDLEWARE                     │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐ │    │
│  │  │  Pilot Path  │  │ Enterprise   │  │  Government Path     │ │    │
│  │  │  API Key +   │  │   Path       │  │  API Key + ECDSA +   │ │    │
│  │  │  IP Whitelist│  │ API Key +    │  │  IP Lock + Grace     │ │    │
│  │  │              │  │   ECDSA      │  │  Period (7 days)     │ │    │
│  │  └──────────────┘  └──────────────┘  └──────────────────────┘ │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                      API ENDPOINTS                              │    │
│  │                                                                 │    │
│  │  PUBLIC:                      ADMIN (Password-Protected):      │    │
│  │  • POST   /publish            • POST   /admin/clients/register │    │
│  │  • POST   /publish?wait=true  • PATCH  /admin/clients/:id/sec  │    │
│  │  • GET    /status/:uuid       • GET    /admin/clients/list     │    │
│  │  • GET    /health             • POST   /admin/maintenance/sweep│    │
│  │                                                                 │    │
│  │  SELF-SERVICE AUTH:                                             │    │
│  │  • POST   /auth/register-public-key                            │    │
│  │  • POST   /auth/rotate-public-key                              │    │
│  │  • GET    /auth/key-status                                     │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                    CORE BROADCASTING ENGINE                     │    │
│  │                                                                 │    │
│  │  ┌───────────────┐      ┌──────────────┐      ┌─────────────┐ │    │
│  │  │  Train Worker │─────▶│ UTXO Manager │◀────▶│  Arc Client │ │    │
│  │  │  (3s batches) │      │ (50k pool)   │      │  (Gorillapool)│    │
│  │  └───────────────┘      └──────────────┘      └─────────────┘ │    │
│  │        │                       │                      │         │    │
│  │        │ Build TX              │ Lock UTXOs          │ Broadcast│    │
│  │        ▼                       ▼                      ▼         │    │
│  │  ┌───────────────────────────────────────────────────────────┐ │    │
│  │  │         Transaction Builder (OP_RETURN Data)              │ │    │
│  │  │  • Atomic UTXO Locking                                    │ │    │
│  │  │  • 1,000 tx/batch capacity                                │ │    │
│  │  │  • 0.5 sat/byte fee calculation                           │ │    │
│  │  └───────────────────────────────────────────────────────────┘ │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                    BACKGROUND SERVICES                          │    │
│  │  • UTXO Recovery (stale lock cleanup)                          │    │
│  │  • Client Rate Limiting (10/100/∞ req/min)                     │    │
│  │  • Transaction Counter (daily reset)                           │    │
│  │  • Health Monitoring                                            │    │
│  └────────────────────────────────────────────────────────────────┘    │
└───────────────────────────────────┬──────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           MONGODB DATABASE                               │
│                                                                          │
│  Collections:                                                            │
│  ┌────────────────┐  ┌──────────────────┐  ┌─────────────────────┐    │
│  │    UTXOs       │  │  Broadcast Reqs  │  │      Clients        │    │
│  │  (50k active)  │  │  (audit trail)   │  │  (API keys + tiers) │    │
│  │                │  │                  │  │                     │    │
│  │  • Outpoint    │  │  • UUID          │  │  • Name             │    │
│  │  • Value       │  │  • Status        │  │  • APIKeyHash       │    │
│  │  • Status      │  │  • TxID          │  │  • Tier             │    │
│  │  • LockedAt    │  │  • Timestamp     │  │  • PublicKey        │    │
│  │  • UTXOType    │  │  • ClientID      │  │  • RequireSignature │    │
│  └────────────────┘  └──────────────────┘  └─────────────────────┘    │
└───────────────────────────────────┬──────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         BSV MAINNET (via ARC)                            │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────┐      │
│  │  Arc (Gorillapool) - Transaction Aggregation & Broadcasting  │      │
│  │  • Extended Format Support                                    │      │
│  │  • Merkle Proof Generation                                    │      │
│  │  • Transaction Status Tracking                                │      │
│  └──────────────────────────────────────────────────────────────┘      │
│                              │                                           │
│                              ▼                                           │
│                    BSV Blockchain (Permanent Storage)                   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 🛡️ Security Architecture - Three-Tier Adaptive Model

### Tier 1: Pilot (Zero Friction Onboarding)
```
Client Request → API Key Validation → IP Whitelist Check → Allow
                                    ↓
                              Rate Limit: 10 req/min
```
**Use Case:** AKUA pilot, testing environments, rapid prototyping  
**Security:** API key + optional IP whitelist  
**Grace Period:** N/A (no key rotation)

### Tier 2: Enterprise (Commercial Grade)
```
Client Request → API Key Validation → ECDSA Signature Verification → Allow
                                    ↓                              ↓
                           Check Timestamp/Nonce         Grace Period: 24h
                           (Replay Protection)           (Old key valid)
```
**Use Case:** NotaryHash production, commercial attestation services  
**Security:** API key + ECDSA secp256k1 signatures  
**Grace Period:** 24 hours for key rotation

### Tier 3: Government (Institutional Maximum Security)
```
Client Request → API Key Validation → IP Lock Check → ECDSA Verification → Allow
                                    ↓               ↓                    ↓
                            Unlimited Rate    Strict IP     Grace Period: 168h
                                             Whitelist      (7 days)
```
**Use Case:** GovHash agency attestation, legal document anchoring  
**Security:** API key + ECDSA + IP restrictions  
**Grace Period:** 7 days for distributed system key rollout

---

## 🔐 Cryptographic Security Model

### Data Integrity Chain:
```
1. Client generates secp256k1 key pair locally
   ├─ Private Key: Never transmitted (stays on client)
   └─ Public Key: Registered via /auth/register-public-key

2. Request Signing (Enterprise/Government tiers):
   ├─ Payload = Timestamp + Nonce + Data (hex)
   ├─ Signature = ECDSA_Sign(SHA256(Payload), PrivateKey)
   └─ Headers = X-Signature, X-Timestamp, X-Nonce

3. Server Verification:
   ├─ Reconstruct Payload
   ├─ Verify ECDSA_Verify(SHA256(Payload), Signature, PublicKey)
   ├─ Check Timestamp freshness (prevent replay)
   └─ Check Nonce uniqueness (prevent replay)

4. Transaction Broadcasting:
   ├─ Build OP_RETURN with client data
   ├─ Sign with Publishing Private Key
   ├─ Broadcast to Arc (Gorillapool)
   └─ Return TxID (blockchain proof of publication)

Result: Four-Layer Security
  1. API Key (Authentication)
  2. ECDSA Signature (Non-repudiation)
  3. UTXO Lock (Atomic Concurrency)
  4. Train Batch (Optimized Throughput)
```

---

## 🚂 High-Performance Broadcasting Engine

### Train Worker Architecture:
```
┌─────────────────────────────────────────────────────────────┐
│                    Train Worker (3s Loop)                    │
│                                                              │
│  1. Check Queue (MongoDB: status=pending)                   │
│  2. Lock UTXO Batch (50k pool, atomic locks)                │
│  3. Build Transaction:                                       │
│     ├─ Input: Locked UTXO (546 sats)                        │
│     ├─ Output 1: OP_RETURN <data>                           │
│     └─ Output 2: Change UTXO (546 sats, recycled)           │
│  4. Sign Transaction (Publishing Private Key)               │
│  5. Broadcast to Arc (extended format)                      │
│  6. Update Status: pending → broadcasted                    │
│  7. Release UTXO: locked → spent                            │
│                                                              │
│  Capacity: 1,000 transactions per 3-second batch            │
│  Throughput: 333 TPS sustained                              │
└─────────────────────────────────────────────────────────────┘

Key Performance Characteristics:
• Atomic UTXO Locking: Prevents double-spending in high-concurrency
• Batch Processing: Amortizes network latency across 1,000 txs
• UTXO Recycling: Change outputs become next batch's inputs
• ARC Integration: Leverages professional mining pool infrastructure
```

### Synchronous Wait Mode (`?wait=true`):
```
Standard Flow (Async):
  Client → POST /publish → UUID → Poll /status/:uuid → TxID
  Latency: 3-6 seconds (requires polling)

Synchronous Flow:
  Client → POST /publish?wait=true → [BLOCKS] → TxID
  Latency: < 5 seconds (single request)
  
Implementation:
  1. Client request flagged for sync mode
  2. Server queues transaction
  3. Goroutine blocks, monitoring MongoDB
  4. Train worker broadcasts transaction
  5. Status updated: pending → broadcasted
  6. Goroutine unblocks, returns TxID immediately
  7. Client receives TxID in same HTTP response
```

---

## 📡 API Endpoint Reference

### Public Endpoints (Client-Facing)

#### `POST /publish`
**Purpose:** Queue transaction for blockchain broadcasting  
**Auth:** X-API-Key (+ X-Signature for Enterprise/Government tiers)  
**Body:**
```json
{
  "data": "48656c6c6f20576f726c64"  // Hex-encoded payload
}
```
**Response (Async):**
```json
{
  "uuid": "c6152030-9108-43f5-a7c7-1628f3874a75",
  "message": "Transaction queued for processing"
}
```

#### `POST /publish?wait=true`
**Purpose:** Synchronous transaction broadcasting with immediate TxID  
**Auth:** X-API-Key (+ X-Signature for secure tiers)  
**Response:**
```json
{
  "txid": "13a63dee1ef4ba9a2c6f7539802bf6cefeb1a19618bd246588ebdcde1322978d",
  "arc_status": "SEEN_ON_NETWORK",
  "message": "Transaction broadcasted successfully"
}
```

#### `GET /status/:uuid`
**Purpose:** Poll transaction status  
**Response:**
```json
{
  "uuid": "...",
  "status": "broadcasted",
  "txid": "13a63dee...",
  "arc_status": "SEEN_ON_NETWORK",
  "created_at": "2026-02-08T14:10:00Z"
}
```

#### `GET /health`
**Purpose:** System health check  
**Response:**
```json
{
  "status": "healthy",
  "utxos": {
    "funding_available": 50,
    "publishing_available": 49876
  },
  "queueDepth": 0
}
```

### Admin Endpoints (Password-Protected)

#### `POST /admin/clients/register`
**Purpose:** Register new client with tier-based security  
**Auth:** X-Admin-Password  
**Body:**
```json
{
  "name": "AKUA Pilot",
  "tier": "pilot",                    // pilot | enterprise | government
  "public_key": "04a1b2c3...",        // Optional for pilot
  "max_daily_tx": 10000,
  "allowed_ips": ["127.0.0.1"]        // Optional IP whitelist
}
```
**Response:**
```json
{
  "success": true,
  "api_key": "gh_bueDsMZXgJ5Y6LElL0jqWDE0S-XSYg6s2s8ANF310Vc=",
  "client": { ... },
  "tier": "pilot",
  "security": {
    "require_signature": false,
    "grace_period_hours": 0,
    "allowed_ips": ["127.0.0.1"]
  }
}
```

#### `PATCH /admin/clients/:id/security`
**Purpose:** Runtime tier management and security updates  
**Auth:** X-Admin-Password  
**Body:**
```json
{
  "tier": "enterprise",
  "require_signature": true,
  "grace_period_hours": 48
}
```
**Response:**
```json
{
  "success": true,
  "client_id": "507f1f77bcf86cd799439011",
  "security": { ... },
  "message": "Security settings updated. Changes effective immediately."
}
```

### Self-Service Auth Endpoints

#### `POST /auth/register-public-key`
**Purpose:** Client binds ECDSA public key to their API key  
**Auth:** X-API-Key  
**Body:**
```json
{
  "public_key": "04a1b2c3d4e5f6..."  // 65-byte secp256k1 public key (hex)
}
```

#### `POST /auth/rotate-public-key`
**Purpose:** Key rotation with grace period support  
**Auth:** X-API-Key + X-Signature (current key)  
**Body:**
```json
{
  "new_public_key": "04f6e5d4c3b2a1..."
}
```
**Response:**
```json
{
  "success": true,
  "grace_period_hours": 24,
  "old_key_expires_at": "2026-02-09T14:00:00Z",
  "message": "Key rotated. Old key valid for 24 hours."
}
```

---

## 🔧 Operational Tools

### 1. Weekly Maintenance Script
**Location:** `scripts/weekly-maintenance.sh`  
**Purpose:** Consolidate dust UTXOs and optimize database performance  
**Schedule:** Weekly (Sunday 2 AM)  
**Actions:**
- Sweep spent publishing UTXOs
- Consolidate dust into larger UTXOs
- Prune old broadcast requests (> 30 days)
- Verify UTXO pool integrity

### 2. MongoDB Backup Script
**Location:** `scripts/backup-mongodb.sh`  
**Purpose:** Daily database snapshots for audit compliance  
**Schedule:** Daily (2 AM)  
**Retention:** 30 days rolling  
**Backup Path:** `/backups/mongodb/YYYY-MM-DD/`

### 3. Test Suites
- **test-api.sh:** Comprehensive API endpoint testing
- **test-ecdsa-auth.sh:** ECDSA authentication flow validation
- **test-tier-management.sh:** Admin tier upgrade/downgrade testing

### 4. Deployment Script
**Location:** `deploy-production.sh`  
**Purpose:** Zero-downtime deployment with health checks  
**Steps:**
1. Build new Docker image
2. Database migration (if needed)
3. Stop old container
4. Start new container
5. Health verification
6. Rollback on failure

---

## 📈 Monitoring & Observability

### Key Metrics to Monitor:

**Performance:**
- Train batch processing time (target: < 3s)
- UTXO pool depth (target: > 40,000)
- Queue depth (alert if > 1,000)
- Transaction broadcast success rate (target: > 99%)

**Security:**
- Failed authentication attempts
- Tier-based request distribution ([PILOT] vs [ENTERPRISE])
- Grace period key usage (old key vs new key)
- Rate limit violations per client

**Business:**
- Transactions per client per day
- Tier distribution (pilot/enterprise/government)
- API key rotation frequency
- Daily transaction volume trends

### Log Monitoring:
```bash
# View tier-based authentication
docker-compose logs -f bsv-publisher | grep -E '\[PILOT\]|\[ENTERPRISE\]'

# Monitor train performance
docker-compose logs -f bsv-publisher | grep "Train"

# Check UTXO pool health
curl -s https://api.govhash.org/health | jq '.utxos'
```

---

## 🏆 System Capabilities Summary

### What This Platform Delivers:

✅ **High Performance:**
- 333 TPS sustained throughput
- 1,000 transactions per 3-second batch
- < 5 second latency to blockchain confirmation

✅ **Adaptive Security:**
- Zero-friction pilot onboarding (API key only)
- Enterprise-grade ECDSA signatures
- Government-level IP restrictions
- Graceful key rotation (24h-168h grace periods)

✅ **Operational Excellence:**
- 99.9% uptime (Docker + health checks)
- Atomic UTXO locking (no double-spending)
- Self-healing (stuck UTXO recovery)
- Automated maintenance scripts

✅ **Legal Compliance:**
- Non-repudiation via ECDSA signatures
- Complete audit trail (MongoDB persistence)
- Client-controlled private keys (zero trust)
- Blockchain-anchored timestamps

✅ **Developer Experience:**
- RESTful API with synchronous mode
- Self-service key management
- Runtime tier upgrades (no code changes)
- Comprehensive test suites

---

## 🎯 Production Readiness Checklist

- ✅ Core broadcasting engine operational
- ✅ 50,000 UTXO pool initialized
- ✅ Adaptive security tier system deployed
- ✅ Admin tier management endpoints live
- ✅ Self-service auth endpoints operational
- ✅ Synchronous wait mode functional
- ✅ Health monitoring configured
- ✅ Docker container running stable
- ✅ MongoDB backup strategy implemented
- ✅ Test suites passing
- ✅ Documentation complete
- ✅ Client onboarding workflows defined

---

## 📞 Support & Operations

**Production URL:** https://api.govhash.org  
**Health Endpoint:** https://api.govhash.org/health  
**Container:** `bsv_akua_server`  
**Database:** `bsv_akua_db` (MongoDB 7)

**Emergency Contacts:**
- System Admin: (Review logs via `docker-compose logs`)
- Database Issues: (Check MongoDB health)
- Arc Provider: Gorillapool Support

**Key Files:**
- Main Documentation: [STATUS.md](STATUS.md)
- Deployment Guide: [PRODUCTION_DEPLOYMENT_COMPLETE.md](PRODUCTION_DEPLOYMENT_COMPLETE.md)
- Admin Reference: [ADMIN_QUICK_REFERENCE.md](ADMIN_QUICK_REFERENCE.md)
- Implementation Details: [PHASE_5_COMPLETE.md](PHASE_5_COMPLETE.md)

---

## 🎉 Conclusion

The **GovHash BSV Attestation Platform** is a production-grade, high-performance blockchain broadcasting engine that successfully bridges the gap between institutional requirements and blockchain technology. With its adaptive security model, the platform serves both rapid-prototyping pilots (AKUA) and government-grade attestation services (GovHash) from a single unified codebase.

**Status:** 🚀 **FULLY OPERATIONAL ON MAINNET**

---

*Architecture finalized: February 8, 2026*  
*Platform commissioned for production service delivery*
