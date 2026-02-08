# GovHash API Stress Testing Suite

Comprehensive stress tests for the GovHash BSV broadcasting API, testing concurrency, authentication methods, and performance characteristics.

## 📋 Test Scripts

### 1. `stress-test-basic.sh`
Tests basic API key authentication with configurable concurrency.

**Features:**
- Concurrent request testing (default: 50 concurrent, 200 total)
- API key authentication
- Configurable `?wait=true` parameter
- Automatic client registration
- Detailed latency metrics (avg, p50, p95, p99)
- CSV results export

**Usage:**
```bash
# Default test (50 concurrent, 200 total requests)
ADMIN_PASSWORD='your_password' ./stress-test-basic.sh

# Custom configuration
ADMIN_PASSWORD='your_password' \
  CONCURRENT_REQUESTS=100 \
  TOTAL_REQUESTS=500 \
  USE_WAIT_PARAM=true \
  ./stress-test-basic.sh
```

### 2. `stress-test-ecdsa.sh`
Tests ECDSA signature authentication with secp256k1.

**Features:**
- ECDSA secp256k1 key generation
- Signature-based authentication
- Timestamp and nonce validation
- Replay attack protection testing
- Concurrent signed requests (default: 25 concurrent, 100 total)
- Security overhead measurement

**Usage:**
```bash
# Default ECDSA test
ADMIN_PASSWORD='your_password' ./stress-test-ecdsa.sh

# High concurrency ECDSA test
ADMIN_PASSWORD='your_password' \
  CONCURRENT_REQUESTS=50 \
  TOTAL_REQUESTS=200 \
  ./stress-test-ecdsa.sh
```

### 3. `stress-test-comparison.sh`
Comprehensive suite comparing all authentication methods and parameters.

**Features:**
- Runs 4 test scenarios:
  1. Basic auth with `wait=true`
  2. Basic auth with `wait=false`
  3. ECDSA auth with `wait=true`
  4. ECDSA auth with `wait=false`
- Side-by-side performance comparison
- HTML report generation
- Recommendations based on results

**Usage:**
```bash
ADMIN_PASSWORD='your_password' ./stress-test-comparison.sh
```

## 📊 Metrics Collected

### Performance Metrics
- **Throughput:** Transactions per second
- **Latency Distribution:**
  - Average (mean)
  - Minimum / Maximum
  - Median (p50)
  - 95th percentile (p95)
  - 99th percentile (p99)
- **Success Rates:**
  - Instant success (TXID returned immediately)
  - Queued (UUID returned, async processing)
  - Failed requests

### System Metrics
- Queue depth (before/after)
- UTXO consumption
- Response time breakdown
- HTTP status code distribution

## 🎯 Expected Results

### Basic Authentication (wait=true)
```
Total Requests:       200
Instant Success:      180-200 (90-100%)
Queued (Async):       0-20 (0-10%)
Failed:               <5 (<2.5%)

Performance:
- Throughput:         40-60 tx/sec
- Average Latency:    800-1500 ms
- P95 Latency:        2000-3000 ms
- P99 Latency:        3000-5000 ms
```

### ECDSA Authentication (wait=true)
```
Total Requests:       100
Instant Success:      90-100 (90-100%)
Queued (Async):       0-10 (0-10%)
Failed:               <3 (<3%)

Performance:
- Throughput:         20-35 tx/sec
- Average Latency:    1000-1800 ms (includes signature generation)
- P95 Latency:        2500-4000 ms
- Signature Overhead: 10-20 ms per request
```

## 🚀 Performance Optimization Tips

### When Queue is Empty (Most Common)
- Use `?wait=true` for instant TXID retrieval
- Expected response time: < 1 second
- No polling required

### When Queue Exists
- System automatically falls back to async mode
- Polling interval: 2 seconds
- Maximum polling: 10 attempts (20 seconds)

### Concurrency Tuning
```bash
# Low concurrency (safer, slower)
CONCURRENT_REQUESTS=10 TOTAL_REQUESTS=100

# Medium concurrency (balanced)
CONCURRENT_REQUESTS=50 TOTAL_REQUESTS=200

# High concurrency (stress test)
CONCURRENT_REQUESTS=100 TOTAL_REQUESTS=500
```

## 📁 Results Structure

Each test creates a timestamped directory in `/tmp/`:

```
/tmp/govhash_stress_1738963200/
├── results.csv          # Raw request data
├── summary.txt          # Human-readable summary
├── api_key.txt          # Generated API key
├── ecdsa_private.pem    # ECDSA private key (if applicable)
├── ecdsa_public.pem     # ECDSA public key (if applicable)
└── public_key.txt       # Public key hex (if applicable)
```

### CSV Format
```csv
request_id,http_code,duration_ms,curl_time,txid,uuid,arc_status
1,201,850,0.852,abc123...,uuid-123...,SEEN_ON_NETWORK
2,202,920,0.925,,uuid-456...,
```

## 🔧 Requirements

### System Requirements
- **OS:** Linux (Ubuntu/Debian recommended)
- **Tools:**
  - `curl` - HTTP client
  - `openssl` - ECDSA key generation
  - `bc` - Calculations
  - `jq` or `python3` - JSON parsing
  - `xxd` - Hex encoding
  - `uuidgen` - UUID generation

### Optional (Recommended)
- **GNU parallel** - For better concurrency performance
  ```bash
  # Ubuntu/Debian
  sudo apt-get install parallel
  
  # macOS
  brew install parallel
  ```

### Install Dependencies
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y curl openssl bc python3 uuid-runtime

# macOS
brew install coreutils
```

## 🔐 Security Notes

### API Keys
- Generated API keys are saved to `api_key.txt`
- Keys are valid for the lifetime of the client
- Rotate keys quarterly for security

### ECDSA Keys
- Private keys saved to `ecdsa_private.pem` - **KEEP SECURE!**
- Public keys registered with API server
- Use separate keys for production vs testing

### Admin Password
- **Never** commit admin password to git
- Use environment variables: `export ADMIN_PASSWORD='...'`
- Rotate admin password regularly

## 📈 Interpreting Results

### Success Rate > 95%
✅ **Excellent** - System performing optimally

### Success Rate 90-95%
⚠️ **Good** - May indicate high load or UTXO pressure

### Success Rate < 90%
❌ **Poor** - Investigate system health, UTXO pool, or network issues

### Latency Benchmarks

| Metric | Excellent | Good | Needs Investigation |
|--------|-----------|------|---------------------|
| P50 Latency | < 1000ms | 1000-2000ms | > 2000ms |
| P95 Latency | < 2500ms | 2500-5000ms | > 5000ms |
| P99 Latency | < 5000ms | 5000-10000ms | > 10000ms |

## 🐛 Troubleshooting

### "Command not found: parallel"
Install GNU parallel or tests will automatically fall back to `xargs`:
```bash
sudo apt-get install parallel
```

### "Registration failed with HTTP 401"
Check admin password:
```bash
echo $ADMIN_PASSWORD  # Should not be empty
```

### "Rate limit exceeded"
Client hit daily transaction limit:
```bash
# Check current usage
curl -s https://api.govhash.org/admin/clients/list \
  -H "X-Admin-Password: $ADMIN_PASSWORD" | jq
```

### High failure rate (>5%)
Check system health:
```bash
curl -s https://api.govhash.org/health | jq
# Look for:
# - queueDepth > 500 (backlog)
# - publishing_available < 10000 (low UTXOs)
```

## 📞 Support

For issues or questions:
- **Email:** support@govhash.org
- **Documentation:** [docs/CLIENT_GUIDE.md](../docs/CLIENT_GUIDE.md)
- **Operations Guide:** [docs/TEAM_OPERATIONS_GUIDE.md](../docs/TEAM_OPERATIONS_GUIDE.md)

## 📝 Example Output

```
╔═════════════════════════════════════════════════════════╗
║              STRESS TEST RESULTS                        ║
╚═════════════════════════════════════════════════════════╝

Request Summary:
  Total Sent:           200
  Instant Success:      195 (97.5%)
  Queued (Async):       3 (1.5%)
  Failed:               2 (1.0%)

Performance Metrics:
  Total Duration:       4s
  Throughput:           50.00 tx/sec
  Concurrency Level:    50

Latency Distribution (ms):
  Average (Mean):       850 ms
  Minimum:              245 ms
  Maximum:              3420 ms
  Median (p50):         780 ms
  95th Percentile:      2100 ms
  99th Percentile:      3200 ms

System State:
  Initial Queue:        0
  Final Queue:          3 (Δ +3)
  Initial UTXOs:        49875
  Final UTXOs:          49675 (Used: 200)

╔═════════════════════════════════════════════════════════╗
║              STRESS TEST COMPLETE ✓                     ║
╚═════════════════════════════════════════════════════════╝
```

## 🎓 Learning Resources

- **Basic Authentication:** See [docs/CLIENT_GUIDE.md](../docs/CLIENT_GUIDE.md#authentication)
- **ECDSA Signatures:** See [docs/CLIENT_GUIDE.md](../docs/CLIENT_GUIDE.md#ecdsa-authentication)
- **Rate Limits:** See [docs/CLIENT_GUIDE.md](../docs/CLIENT_GUIDE.md#rate-limits)
- **Error Handling:** See [docs/CLIENT_GUIDE.md](../docs/CLIENT_GUIDE.md#error-handling)

---

**Last Updated:** February 8, 2026  
**Version:** 1.0.0  
**Maintainer:** GovHash Operations Team
