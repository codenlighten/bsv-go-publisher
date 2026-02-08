# GovHash AKUA Integration Guide

Production-ready Node.js scripts for integrating with the GovHash transaction publishing API. These scripts provide everything you need to start broadcasting Bitcoin transactions at scale.

**Last Updated:** 2024  
**API Version:** v1  
**Status:** ✅ Production Ready

---

## 📦 What's Included

### Scripts

1. **basic-publish.js** - Single transaction publishing
   - Simple wrapper around the publish API
   - Converts text to hex automatically
   - Returns TXID immediately with `?wait=true`

2. **batch-publish.js** - Batch transaction processing
   - Load CSV or JSON files with transactions
   - Respects train architecture timing
   - Parallel worker threads with configurable concurrency
   - Generates results CSV with TXID and status

3. **stress-test.js** - Performance testing
   - Configurable concurrency and duration
   - Rate limiting (RPS) support
   - Latency percentiles and throughput metrics
   - HTML report generation option

4. **health-monitor.js** - Production monitoring
   - Continuous health checks
   - Slack webhook alerts
   - Queue depth and latency monitoring
   - Configurable thresholds and intervals

5. **status-tracker.js** - UUID resolution
   - Track pending transactions by UUID
   - Poll until TXID is assigned
   - Batch UUID tracking from file
   - CSV results export

---

## 🚀 Quick Start

### 1. Installation

```bash
# Clone or download the scripts
cd akua-integration-scripts

# Copy environment template
cp .env.example .env

# Edit .env with your API key
nano .env
```

### 2. Set Your API Key

Edit `.env`:
```env
GOVHASH_API_KEY=gh_KqxxVawkirYuNvyzXEELUzUAA3-_20nzRAm-QWF2P-M=
GOVHASH_API_URL=https://api.govhash.org
```

Or set as environment variable:
```bash
export GOVHASH_API_KEY="gh_KqxxVawkirYuNvyzXEELUzUAA3-_20nzRAm-QWF2P-M="
node basic-publish.js "Hello GovHash"
```

### 3. Publish Your First Transaction

```bash
# Simple text message
node basic-publish.js "Hello World"

# Or hex data
node basic-publish.js --data="48656c6c6f"
```

Expected output:
```
✅ Transaction Published Successfully

  TXID:         f9c95e2a6f937e02a90df161269ab4b21ea109ea97888cd2ea6cd44ea25c2990
  UUID:         b38b8785-ec96-4c30-ab68-82e01029b7d7
  ARC Status:   SEEN_ON_NETWORK
  Latency:      2,145ms
  Timestamp:    2024-01-15T10:30:45.123Z
```

---

## 🚂 Understanding Train Broadcasting

The GovHash publishing system uses a **train architecture** for efficient batch broadcasting:

### Train Specifications

- **Capacity:** 1,000 transactions per train
- **Frequency:** Every 3 seconds
- **Throughput:** 333 transactions/second sustained
- **Daily Capacity:** 28.8 million transactions

### Queue Behavior

When you publish a transaction, one of two things happens:

**Case 1: Queue < 1,000 pending**
```
Request: POST /publish?wait=true
Response: {
  "txid": "f9c95e2a6f937e02...",  // Immediate TXID
  "arc_status": "SEEN_ON_NETWORK"
}
Latency: ~2-3 seconds
```

**Case 2: Queue ≥ 1,000 pending**
```
Request: POST /publish
Response: {
  "uuid": "b38b8785-ec96-4c30-ab68-82e01029b7d7",  // UUID instead
  "arc_status": "QUEUED"
}
Latency: Varies (added to queue)
→ Later: Check /status/{uuid} to get TXID
```

### Best Practices

- Use `?wait=true` for real-time applications
- Expect 2-3 second response time (one train cycle)
- Batch requests in groups < 1,000 when possible
- Space large batches 3+ seconds apart for predictable timing

---

## 📋 Usage Examples

### Single Transaction (Real-Time)

```bash
node basic-publish.js "Your message here"
```

### Batch from CSV File

Create `transactions.csv`:
```csv
id,data,priority
1,48656c6c6f,high
2,776f726c64,normal
3,7465737420646174610a,low
```

Process with:
```bash
node batch-publish.js transactions.csv --workers=10

# Output:
# 📤 Publishing 3 transactions...
# 📊 Progress: 3/3 (100%) | ✅ 3 | ❌ 0
# 
# 📈 Statistics:
#   Total:        3
#   Success:      3 (100%)
#   Duration:     3.5s
#   Throughput:   0.86 tx/s
#   Avg Latency:  2,341ms
# 
# 💾 Results saved to: batch-results.csv
```

### Stress Testing

Test your system's capacity:

```bash
# Light load: 100 requests, 10 concurrent
node stress-test.js --requests=100 --concurrency=10

# Moderate load: 500 requests, 50 concurrent, 2-second limit
node stress-test.js --requests=500 --concurrency=50 --rps=250

# Heavy load with report: Run for 60 seconds at unlimited concurrency
node stress-test.js --duration=60 --analyze

# Output shows:
# - P50, P95, P99 latencies
# - Success rate percentage
# - HTTP status breakdown
# - Throughput (tx/s)
# - Recommendations for optimization
```

### Continuous Health Monitoring

```bash
# Check once
node health-monitor.js --check-once

# Monitor every 30 seconds
node health-monitor.js --interval=30

# Monitor for 24 hours with alerts
node health-monitor.js --duration=1440 --alert-latency=5000 --alert-queue=800

# Log to file and Slack
export SLACK_WEBHOOK="https://hooks.slack.com/services/YOUR/WEBHOOK"
node health-monitor.js --output=health.log --alert-latency=10000
```

### Track Pending Transactions

When queue is full, you get a UUID instead of TXID:

```bash
# Check single UUID status
node status-tracker.js --uuid=b38b8785-ec96-4c30-ab68-82e01029b7d7

# Wait until it has a TXID
node status-tracker.js --uuid=b38b8785-ec96-4c30-ab68-82e01029b7d7 --wait

# Track multiple UUIDs from file
echo "b38b8785-ec96-4c30-ab68-82e01029b7d7" > uuids.txt
echo "a1234567-b8c9-d0e1-f2g3-h4i5j6k7l8m9" >> uuids.txt
node status-tracker.js --input=uuids.txt --wait --output=status.csv
```

---

## 🔌 Integration Patterns

### Pattern 1: Real-Time Publishing (< 50 concurrent)

For applications needing immediate confirmation:

```javascript
const { publishTransaction, textToHex } = require('./basic-publish.js');

async function publishRealTime(message) {
  const hexData = textToHex(message);
  const result = await publishTransaction(hexData);
  
  if (result.success) {
    console.log(`Transaction: ${result.txid}`);
    // Process TXID immediately
  }
}
```

**Characteristics:**
- Latency: 2-3 seconds (typical)
- Success Rate: 100%
- When to use: User-facing transactions, APIs needing instant response
- Max concurrency: 50 (before hitting train limits)

### Pattern 2: Batch Processing (100-1,000 tx/min)

For high-volume batch operations:

```bash
# Create batches that respect 3-second train cycles
node batch-publish.js large-batch.csv --workers=50

# Or programmatically:
# - Split data into 333-transaction chunks
# - Wait 3+ seconds between chunks
# - Parallel processing within chunks
```

**Characteristics:**
- Throughput: 100-1,000 tx/min
- Latency: 2-5 seconds per batch
- Success Rate: 95%+
- When to use: Data import, periodic sync operations

### Pattern 3: High-Throughput Publishing (1,000+ tx/min)

For maximum capacity utilization:

```bash
# Run continuous batches respecting train timing
for i in {1..24}; do
  node batch-publish.js batch-$i.csv --workers=100 &
  sleep 3  # Wait for train cycle
done
wait
```

**Characteristics:**
- Throughput: 1,000-20,000 tx/min
- Some transactions get UUID instead of TXID
- Use status-tracker to resolve UUIDs
- When to use: System migration, bulk data processing

---

## 💡 Best Practices

### DO

- ✅ Use `?wait=true` for interactive users
- ✅ Batch requests when possible (respects train timing)
- ✅ Implement connection pooling in production
- ✅ Monitor queue depth and latency metrics
- ✅ Space large batches 3+ seconds apart
- ✅ Handle both TXID and UUID responses
- ✅ Implement retry logic for transient failures
- ✅ Use HTTPS only (never HTTP)
- ✅ Store API key in environment variables
- ✅ Set reasonable timeouts (30s default)
- ✅ Log all transaction IDs for audit trail
- ✅ Test with small batches first before scaling

### DON'T

- ❌ Hardcode API keys in source code
- ❌ Use HTTP instead of HTTPS
- ❌ Ignore UUID responses (they will convert to TXID)
- ❌ Exceed 1,000 concurrent requests
- ❌ Retry immediately on failure (exponential backoff)
- ❌ Send more than 1MB payload per transaction
- ❌ Ignore monitoring alerts
- ❌ Run without error handling
- ❌ Assume instant response (expect 2-3 seconds min)
- ❌ Forget to handle queue overflow gracefully
- ❌ Store transactions in memory (use database)
- ❌ Skip testing on staging before production

---

## 📊 Performance Expectations

### Light Load (< 50 concurrent)

```
Success Rate: 99%+
Avg Latency: 2,400ms
P95 Latency: 3,200ms
P99 Latency: 3,800ms
Throughput: 40 tx/s
```

Use for: User-facing applications, APIs, interactive tools

### Medium Load (50-200 concurrent)

```
Success Rate: 95%+
Avg Latency: 3,100ms
P95 Latency: 4,500ms
P99 Latency: 5,200ms
Throughput: 150 tx/s
```

Use for: Batch operations, data import, periodic tasks

### Heavy Load (200+ concurrent)

```
Success Rate: 90%+ (some UUIDs)
Avg Latency: 4,200ms
P95 Latency: 6,800ms
P99 Latency: 8,100ms
Throughput: 280 tx/s
```

Use for: System migration, full capacity testing

**Note:** Train architecture ensures graceful degradation. System never crashes, but returns UUIDs instead of TXIDs at capacity.

---

## 🔍 Monitoring & Debugging

### Check System Health

```bash
# One-time health check
node health-monitor.js --check-once --quiet

# Continuous monitoring with metrics
node health-monitor.js --interval=30 --output=health.log

# Expected output:
# ✅ Latency: 2,445ms | Queue: 0 | UTXOs: 49640
```

### Interpret Metrics

| Metric | Good | Warning | Critical |
|--------|------|---------|----------|
| Latency | < 3s | 3-10s | > 10s |
| Queue Depth | < 100 | 100-500 | > 500 |
| UTXO Pool | > 1000 | 100-1000 | < 100 |
| Success Rate | > 98% | 90-98% | < 90% |

### Common Issues

**Issue: HTTP 000 errors**
- Cause: Client-side timeout or connection limit
- Solution: Reduce concurrency, increase timeouts, use proper HTTP client
- Try: `--concurrency=10` (reduced from 50)

**Issue: Queue grows large**
- Cause: Publishing faster than train capacity (1,000/3s)
- Solution: Reduce RPS, batch requests further apart
- Try: Add 3+ second delays between batch submissions

**Issue: High latency (> 10 seconds)**
- Cause: Queue is large, waiting multiple train cycles
- Solution: Reduce concurrent requests, space batches apart
- Try: Monitor queue with `health-monitor.js --interval=5`

**Issue: Latency spikes**
- Cause: Transient network issues or train backlog
- Solution: Implement exponential backoff retry logic
- Expected: Occasional spikes are normal in high-load scenarios

---

## 📈 Advanced Configuration

### Connection Pooling (Production)

For high-throughput applications, consider using a proper HTTP client:

```javascript
const https = require('https');

const agent = new https.Agent({
  keepAlive: true,
  keepAliveMsecs: 30000,
  maxSockets: 50,
  maxFreeSockets: 10,
});

// Use agent in requests
const options = {
  // ... other options
  agent: agent,
};
```

### Rate Limiting

Distribute requests evenly to avoid queue buildup:

```bash
# Limit to 300 RPS (below 333 train capacity)
node stress-test.js --duration=120 --rps=300

# Or programmatically with delays between batches
for batch in batches; do
  node batch-publish.js $batch &
  sleep 3  # Between-batch delay
done
```

### Exponential Backoff Retry

```javascript
async function publishWithRetry(data, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const result = await publishTransaction(data);
      if (result.success) return result;
    } catch (error) {
      if (i < maxRetries - 1) {
        const delay = Math.pow(2, i) * 1000; // 1s, 2s, 4s
        await new Promise(r => setTimeout(r, delay));
      }
    }
  }
  throw new Error('Max retries exceeded');
}
```

### CSV Output Analysis

After batch processing, analyze results:

```bash
# Count successes
grep "SUCCESS" batch-results.csv | wc -l

# Extract TXIDs only
grep "SUCCESS" batch-results.csv | cut -d',' -f3

# Find failures
grep "FAILED" batch-results.csv

# Calculate average latency
awk -F',' 'NR>1 {sum+=$5; count++} END {print "Avg:", sum/count "ms"}' batch-results.csv
```

### Slack Alerts

Enable Slack notifications for critical issues:

```bash
# Get webhook from: https://api.slack.com/messaging/webhooks
export SLACK_WEBHOOK="https://hooks.slack.com/services/YOUR/WEBHOOK/ID"

# Run with alerts
node health-monitor.js \
  --interval=60 \
  --alert-latency=5000 \
  --alert-queue=800
```

---

## 🛠️ Troubleshooting Guide

### Issue: "GOVHASH_API_KEY not set"

**Solution:**
```bash
# Option 1: Create .env file
echo 'GOVHASH_API_KEY=gh_...' > .env

# Option 2: Export variable
export GOVHASH_API_KEY="gh_..."

# Option 3: Command line
GOVHASH_API_KEY=gh_... node basic-publish.js "message"
```

### Issue: "Network error: ENOTFOUND"

**Solution:**
- Check API URL is correct: `https://api.govhash.org`
- Verify internet connectivity
- Check firewall/proxy settings
- Try: `curl https://api.govhash.org/health`

### Issue: "Invalid JSON response"

**Solution:**
- May indicate API error or maintenance
- Check GovHash API status page
- Try: `curl -H "X-API-Key: $GOVHASH_API_KEY" https://api.govhash.org/health`

### Issue: "Request timeout (30s)"

**Solution:**
- Server may be overloaded
- Reduce concurrency: `--concurrency=5`
- Increase timeout in code: `REQUEST_TIMEOUT = 60000`
- Check queue depth: `node health-monitor.js --check-once`

### Issue: Most responses are UUIDs instead of TXIDs

**Solution:**
- Queue is full (> 1,000 pending)
- Use status-tracker to resolve: `node status-tracker.js --input=uuids.txt --wait`
- Reduce publishing rate
- Space batches 3+ seconds apart

### Issue: Success rate < 95%

**Solutions to try in order:**
1. Reduce concurrency by 50%
2. Increase timeout to 60 seconds
3. Check queue depth with health monitor
4. Reduce payload size
5. Contact GovHash support if persistent

---

## 📚 CSV Output Formats

### batch-results.csv

```csv
id,status,txid,uuid,arc_status,latency_ms,error,timestamp
1,SUCCESS,f9c95e2a6f937e02...,b38b8785-ec96-4c30...,SEEN_ON_NETWORK,2145,,2024-01-15T10:30:45.123Z
2,SUCCESS,27b2405cabb424be...,a1234567-b8c9-d0e1...,SEEN_ON_NETWORK,2089,,2024-01-15T10:30:47.210Z
3,FAILED,,,UNKNOWN,30000,HTTP 500,2024-01-15T10:30:50.300Z
```

### stress-test-results.csv

```csv
sent,responseTime,statusCode,txid,uuid,error,success
1705316445123,2145,201,f9c95e2a6f937e02...,,yes
1705316447210,2089,201,27b2405cabb424be...,,yes
1705316450300,30000,0,,timeout error,no
```

### status-tracker Results

```csv
uuid,txid,status,checkTime,confirmed
b38b8785-ec96-4c30-ab68-82e01029b7d7,f9c95e2a6f937e02...,CONFIRMED,2024-01-15T10:35:20.123Z,yes
a1234567-b8c9-d0e1-f2g3-h4i5j6k7l8m9,27b2405cabb424be...,CONFIRMED,2024-01-15T10:35:25.456Z,yes
```

---

## 🔐 Security Guidelines

### API Key Protection

- **Never commit to git:** Use `.env` or environment variables only
- **Never log:** Don't print keys to console or files
- **Never share:** Only give to trusted AKUA team members
- **Rotate regularly:** Request new key quarterly
- **Scope narrowly:** Use read-only keys where possible

### Network Security

- **Always HTTPS:** Never use HTTP for production
- **Verify certificates:** Don't disable SSL verification
- **Use firewalls:** Restrict to known IPs when possible
- **Monitor access:** Check health monitor for suspicious patterns

### Audit Trail

```bash
# Log all transactions to file
echo "$(date): TXID=$txid" >> transaction.log

# Monitor access patterns
grep "ERROR" health.log | tail -20

# Review CSV export regularly
cat batch-results.csv | grep "FAILED"
```

### Key Rotation Example

```bash
# 1. Get new key from GovHash support
# 2. Update .env with new key
sed -i 's/GOVHASH_API_KEY=.*/GOVHASH_API_KEY=gh_new_key/' .env

# 3. Test with health check
node health-monitor.js --check-once

# 4. Archive old logs with old key
tar czf logs-old-key-$(date +%s).tar.gz *.log

# 5. Confirm old key no longer works, then ask to disable
```

---

## 📞 Support & Resources

### Getting Help

- **Documentation:** This file and inline code comments
- **Logs:** Check .env output and CSV files
- **Health Monitor:** `node health-monitor.js --check-once` for quick diagnostics
- **GovHash Support:** Contact your AKUA liaison

### Related Documentation

- AKUA_PUBLISHER_UPDATE.md - API overview and parameters
- STRESS_TEST_RESULTS.md - Performance analysis and benchmarks
- tests/README.md - Advanced stress testing guide

### API Reference

**Base URL:** `https://api.govhash.org`

**Endpoints Used:**

- `POST /publish?wait=true` - Publish with immediate response
- `GET /status/{uuid}` - Check UUID status
- `GET /admin/api/stats` - System statistics (health monitor)

**Headers:**
```
X-API-Key: {your-api-key}
Content-Type: application/json
User-Agent: GovHash-AKUA-Client/1.0
```

### Version Information

- **Scripts Version:** 1.0
- **API Version:** v1
- **Last Updated:** January 2024
- **Tested With:** Node.js 14+

---

## 📝 License

These scripts are provided by GovHash for AKUA team use. Proprietary and confidential.

---

## 💬 Feedback

Have suggestions for improvements? Found an issue? Contact your GovHash liaison with:

1. Script name and version
2. Command that failed (without API key)
3. Output or error message
4. Expected vs actual behavior

Thank you for using GovHash!
