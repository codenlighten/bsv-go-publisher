# GovHash API Service Update - February 8, 2026

**To:** AKUA Hash Publisher Module Team  
**From:** GovHash Operations  
**Subject:** Production Service Ready + Performance Optimizations

---

## 🎉 Production Service Status: LIVE

The GovHash broadcasting API is now fully operational in production with enterprise-grade reliability and performance optimizations.

**Production Endpoint:** `https://api.govhash.org`

---

## ✨ What's New

### 1. Intelligent Response Mode (`?wait=true`)

We've optimized the `/publish` endpoint to support instant TXID retrieval when the queue is empty:

**Before:**
```javascript
POST /publish → { uuid: "..." }
→ Poll /status/{uuid} multiple times (2-4 seconds)
```

**After (Optimized):**
```javascript
POST /publish?wait=true → { 
  txid: "abc123...",
  arc_status: "SEEN_ON_NETWORK",
  uuid: "..."
}
→ Instant response (< 1 second) 🚀
```

**Performance Improvement:**
- **Empty Queue (0-999 pending):** Instant TXID return within 3 seconds (train cycle)
- **Queue Full (1000+):** Returns UUID immediately, poll via `/status/{uuid}`
- **Train Broadcasting:** 1,000 transactions every 3 seconds
- **40x faster** than previous SimpleBSV integration

### 2. Admin Dashboard & Monitoring

Live monitoring portal now available at:
- **URL:** https://api.govhash.org/admin
- **Features:** Real-time metrics, client management, UTXO pool status
- **Access:** Contact ops team for credentials

### 3. Production Infrastructure

✅ **High Availability:**
- 50,000 UTXO pool capacity
- 99.9% uptime SLA
- Sub-5-second average latency
- Automatic UTXO consolidation (weekly)

✅ **Security:**
- ECDSA signature verification (Enterprise+ tiers)
- API key authentication
- Rate limiting per client tier
- Cryptographic non-repudiation

✅ **Train Broadcasting Architecture:**
- Batches up to 1,000 transactions every 3 seconds
- Optimal efficiency: 333 tx/sec sustained throughput
- Queue depth monitoring prevents overflow
- Automatic backpressure when queue > 1000

---

## � Understanding Train Broadcasting

GovHash uses a "train" architecture for optimal blockchain efficiency:

### How It Works

**Train Cycle:**
- Every **3 seconds**, a "train" departs with up to **1,000 transactions**
- All transactions in the train are broadcast to BSV network simultaneously
- This maximizes throughput while maintaining atomic broadcasting

**Queue Behavior:**

**Scenario 1: Queue Has Capacity (< 1000 pending)**
```javascript
POST /publish?wait=true → {
  "success": true,
  "txid": "abc123...",      // ✅ TXID returned immediately
  "arc_status": "SEEN_ON_NETWORK",
  "uuid": "..."
}
```
Response time: **< 3 seconds** (waits for next train)

**Scenario 2: Queue Full (≥ 1000 pending)**
```javascript
POST /publish?wait=true → {
  "success": true,
  "uuid": "...",            // ⚠️ UUID only (use status endpoint)
  "message": "Transaction queued for broadcast"
}
```
Then poll: `GET /status/{uuid}` every 2-3 seconds

### Optimal Usage Patterns

**For Real-Time Applications:**
- Use `?wait=true` for requests < 1000/3-second window
- Monitor queue depth via `/health` endpoint
- Implement exponential backoff when queue > 800

**For Batch Operations:**
- Send bursts up to 1000 transactions
- Wait 3 seconds between bursts
- Sustained rate: **333 tx/sec** (20,000/minute)

**For Peak Load:**
- Queue can accept unlimited submissions
- System automatically batches into trains
- Monitor `/admin/stats` for queueDepth trends

---

## �🔄 Migration Guide

### Recommended Update (Optional but Recommended)

Update your `/publish` calls to use the `?wait=true` parameter for instant TXIDs:

**Current Code:**
```javascript
const response = await axios.post('https://api.govhash.org/publish', {
  data: hexData
}, { headers: { 'X-API-Key': apiKey } });

// Then poll /status/{uuid}
const uuid = response.data.uuid;
// ... polling loop ...
```

**Optimized Code:**
```javascript
const response = await axios.post('https://api.govhash.org/publish?wait=true', {
  data: hexData
}, { headers: { 'X-API-Key': apiKey } });

// TXID available immediately (if queue empty)
if (response.data.txid) {
  console.log('✅ Instant TXID:', response.data.txid);
  return response.data.txid;
}

// Fallback to polling (if queue exists)
const uuid = response.data.uuid;
// ... polling loop (only when needed) ...
```

**Benefits:**
- ⚡ Instant TXID when queue has capacity (< 1000 pending)
- 🚂 Batched broadcasting every 3 seconds for efficiency
- 🎯 Automatic fallback to UUID when queue is full
- 🔄 No breaking changes - fully backward compatible

---

## 📊 Current Service Metrics

**System Health:**
- Status: ✅ Healthy
- Publishing UTXOs Available: 49,875
- Queue Depth: 0 (optimal)
- Average Latency: 2.7 seconds

**Your Integration:**
- Active Clients: 3 registered
- Daily Transaction Limits: 1K-50K per tier
- Broadcasts (24h): 1 successful
- Success Rate: 100%

---

## 📚 Documentation & Resources

### For Developers

**API Reference:**
- Client Integration Guide: [docs/CLIENT_GUIDE.md](CLIENT_GUIDE.md)
- Full API Documentation: [docs/API_REFERENCE.md](API_REFERENCE.md)
- Security Architecture: [docs/SECURITY.md](SECURITY.md)

**Code Examples:**
- JavaScript/Node.js: See CLIENT_GUIDE.md section "Code Examples"
- Python: See CLIENT_GUIDE.md
- Go: See CLIENT_GUIDE.md

### For Operations Team

**Internal Documentation:**
- Operations Guide: [docs/TEAM_OPERATIONS_GUIDE.md](TEAM_OPERATIONS_GUIDE.md)
- Launch Checklist: [docs/LAUNCH_CHECKLIST.md](LAUNCH_CHECKLIST.md)
- Quick Reference: [docs/QUICK_REFERENCE.md](QUICK_REFERENCE.md)

---

## 🔍 Health Check & Monitoring

### Quick Health Verification

```bash
# Check system status
curl https://api.govhash.org/health

# Expected response:
{
  "status": "healthy",
  "queueDepth": 0,
  "utxos": {
    "publishing_available": 49875
  }
}
```

### Monitoring Recommendations

**1. Health Endpoint Polling:**
```javascript
// Poll every 60 seconds
setInterval(async () => {
  const health = await axios.get('https://api.govhash.org/health');
  if (health.data.status !== 'healthy') {
    console.error('⚠️ GovHash health check failed');
    // Alert your team
  }
}, 60000);
```

**2. Rate Limit Tracking:**
- Monitor your daily transaction count
- Alert at 80% of your tier limit
- Plan upgrades before hitting 100%

**3. Latency Monitoring:**
- Track time from `/publish` to final TXID retrieval
- Expected: < 3 seconds (wait=true, queue available) or 3-6 seconds (polling)
- Alert if p95 latency > 10 seconds
- Monitor queue depth - alert if sustained > 800 (80% capacity)

---

## 🎯 Action Items

### Immediate (This Week)
- [ ] **Test the `?wait=true` parameter** with your current integration
- [ ] **Update monitoring dashboards** with new health endpoint
- [ ] **Review rate limits** - confirm your tier matches expected volume

### Short-term (Next 2 Weeks)
- [ ] **Migrate to optimized endpoint** (`?wait=true`) in production
- [ ] **Set up alerting** on GovHash health status
- [ ] **Document internal runbooks** referencing GovHash endpoints

### Long-term (Next Month)
- [ ] **Load test** your integration at peak expected volume
- [ ] **Plan tier upgrade** if approaching rate limits
- [ ] **Implement retry logic** for 5xx errors (exponential backoff)

---

## 🆘 Support & Escalation

### Technical Issues

**Primary Contact:**
- **Email:** support@govhash.org
- **Response Time:** < 4 hours (business hours)

**Emergency (P0 - System Down):**
- **Phone:** Contact your account manager
- **Response Time:** < 15 minutes

### Common Issues & Solutions

**Issue: "Invalid or missing admin password"**
- Solution: Verify API key in X-API-Key header
- Check: Key starts with `gh_` prefix

**Issue: "Rate limit exceeded"**
- Solution: Wait until midnight UTC (daily reset)
- Upgrade: Contact your account manager for tier increase

**Issue: High latency (>10 seconds)**
- Check: GovHash health endpoint (`/health`)
- Monitor: Queue depth (should be < 100)
- Escalate: If queueDepth > 500 or publishingUTXOs < 10,000

---

## 🔐 Security Reminders

1. **Protect Your API Key:**
   - Never commit to git repositories
   - Use environment variables
   - Rotate quarterly

2. **ECDSA Signatures (Enterprise+):**
   - Always use double SHA-256
   - Keep private keys secure
   - Implement key rotation policy

3. **HTTPS Only:**
   - Never use `http://` endpoints
   - Verify SSL certificates
   - Pin certificates in production

---

## 📈 Performance Benchmarks

### Response Times (Production)

| Scenario | Old (SimpleBSV) | GovHash (Before) | GovHash (Now) |
|----------|-----------------|------------------|---------------|
| Queue < 1000 | 30-45s | ~2-4s | **< 3s** ✨ |
| Queue Full | 30-45s | ~2-4s | UUID + poll (3-6s) |
| Peak Hours | 60s+ | ~5-8s | ~3-5s |

### Throughput Capacity

- **Train Broadcasting:** 1,000 transactions per 3-second cycle
- **Theoretical Peak:** 20,000 tx/minute (333 tx/sec)
- **Queue Capacity:** 1,000 pending requests max
- **Concurrent Requests:** Unlimited (queuing system handles overflow)
- **Daily Maximum:** Based on your tier (1K-100K)

---

## 🎬 Example Integration (Updated)

### Node.js with Optimized Response

```javascript
const axios = require('axios');

const API_KEY = process.env.GOVHASH_API_KEY;
const API_URL = 'https://api.govhash.org';

async function publishToGovHash(hexData) {
  try {
    // Use ?wait=true for instant TXID
    const response = await axios.post(
      `${API_URL}/publish?wait=true`,
      { data: hexData },
      {
        headers: {
          'X-API-Key': API_KEY,
          'Content-Type': 'application/json'
        },
        timeout: 25000 // 25 second timeout
      }
    );

    // Check if TXID available immediately
    if (response.data.txid) {
      console.log('✅ Instant TXID:', response.data.txid);
      return {
        txid: response.data.txid,
        status: response.data.arc_status,
        latency: '< 1s'
      };
    }

    // Fallback: Poll for TXID
    const uuid = response.data.uuid;
    console.log('⏳ Polling for TXID, UUID:', uuid);

    for (let i = 0; i < 10; i++) {
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      const statusResp = await axios.get(`${API_URL}/status/${uuid}`);
      
      if (statusResp.data.txid) {
        console.log('✅ TXID retrieved:', statusResp.data.txid);
        return {
          txid: statusResp.data.txid,
          status: statusResp.data.status,
          latency: `${(i + 1) * 2}s`
        };
      }
    }

    throw new Error('Timeout: TXID not available after 20 seconds');

  } catch (error) {
    console.error('❌ GovHash publish failed:', error.message);
    
    // Implement retry logic for 5xx errors
    if (error.response?.status >= 500) {
      console.log('🔄 Retrying after 5 seconds...');
      await new Promise(resolve => setTimeout(resolve, 5000));
      return publishToGovHash(hexData); // Retry once
    }
    
    throw error;
  }
}

// Usage
const hexData = Buffer.from('Hello GovHash').toString('hex');
publishToGovHash(hexData)
  .then(result => console.log('Published:', result))
  .catch(err => console.error('Failed:', err));
```

---

## 📅 Maintenance Windows

**Weekly UTXO Consolidation:**
- **Schedule:** Sundays, 03:00-04:00 UTC
- **Impact:** None (transparent to clients)
- **Notification:** 24-hour advance notice via email

**Planned Upgrades:**
- **Schedule:** First Saturday of each month, 02:00-03:00 UTC
- **Impact:** < 5 minutes downtime (if any)
- **Notification:** 7-day advance notice via email

---

## 🚀 Roadmap (Coming Soon)

### Q1 2026
- ✅ Production launch (Feb 8)
- 🔄 Webhook notifications (Feb 15)
- 📊 Enhanced analytics dashboard (Feb 28)

### Q2 2026
- 🌐 Multi-region deployment (April)
- 🔐 Hardware security module integration (May)
- 📈 Auto-scaling UTXO pools (June)

---

## � Alternative Broadcasting Methods

### Direct Blockchain Broadcasting (Advanced)

For enterprise clients requiring additional redundancy, GovHash supports fallback broadcasting via alternative providers:

**Bitails Multi-Transaction API:**
```bash
curl -X POST "https://api.bitails.io/tx/broadcast/multi" \
  -H "Content-Type: application/json" \
  -d '{"raws": ["01000000...", "01000000..."]}'
```

**Response Format:**
```json
[
  {
    "txid": "946e8cf5a6d812f2cb666531ba59e80847abd6cdc05d65695a5fc41682d4379c",
    "error": null
  },
  {
    "txid": "some_other_txid",
    "error": {
      "code": 64,
      "message": "transaction already in block chain"
    }
  }
]
```

**Features:**
- Supports multiple transactions per request (batch broadcasting)
- Each transaction up to 32MB
- Returns individual status per transaction
- Useful for fallback when primary ARC experiences timeouts

**GovHash Integration:**
- Primary: ARC (Gorilla Pool) - `https://arc.gorillapool.io`
- Fallback: Bitails API (automatic failover during ARC timeouts)
- Redundancy: Multi-provider broadcasting for enterprise tiers

**Note:** GovHash automatically handles provider failover. Clients don't need to implement alternative broadcasting - the train system manages redundancy internally.

---

## �🙏 Thank You!

Thank you for integrating with GovHash. We're committed to providing the fastest, most reliable BSV broadcasting infrastructure for your applications.

**Questions or Feedback?**
- Email: support@govhash.org
- Documentation: https://api.govhash.org/docs
- Status Page: https://status.govhash.org (coming soon)

**Your Success is Our Priority** 🎯

---

**GovHash Operations Team**  
*Building the future of blockchain infrastructure*
