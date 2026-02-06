# Example Usage Guide

This document demonstrates practical usage of the BSV AKUA Broadcast Server.

## Initial Setup

```bash
# 1. Copy environment template
cp .env.example .env

# 2. Edit .env and set your values
nano .env

# Required settings:
# - MONGO_PASSWORD (choose a secure password)
# - ARC_TOKEN (from GorillaPool or TAAL)

# 3. Start the server
make run

# OR with development UI:
make run-dev
```

## First Run

When you start the server for the first time, it will auto-generate BSV keypairs:

```
⚠️  WARNING: No FUNDING_PRIVKEY found in environment!
Generated new keypair:
  Address: 1AbCdEfGhIjKlMnOpQrStUvWxYz...
  Private Key (WIF): L1a2b3c4d5e6f7g8h9...

Add this to your .env file:
FUNDING_PRIVKEY=L1a2b3c4d5e6f7g8h9...
```

**Important:** Copy these keys to your `.env` file to persist them across restarts.

## Funding the Server

Send BSV to your **funding address** (printed on startup):

```bash
# Example: Send 0.1 BSV (10,000,000 satoshis)
bsv-cli sendtoaddress 1AbCdEfGhIjKlMnOpQrStUvWxYz... 0.1
```

This will create UTXOs for splitting into 100-sat publishing UTXOs.

## Creating Publishing UTXOs

Once funded, you need to split your funding UTXOs into 50,000 × 100-sat publishing UTXOs.

### Manual Approach (Current)

The splitter code is implemented but requires integration. You can:

1. Call the splitter programmatically
2. Use the (future) admin endpoint
3. Manually create and broadcast the split transactions

### Expected Flow

```bash
# Future admin endpoint:
curl -X POST http://localhost:8080/admin/split \
  -H "Content-Type: application/json" \
  -d '{"count": 50000}'
```

## Publishing OP_RETURN Data

### 1. Simple Text Example

```bash
# Convert "Hello BSV" to hex
echo -n "Hello BSV" | xxd -p
# Output: 48656c6c6f20425356

# Publish it
curl -X POST http://localhost:8080/publish \
  -H "Content-Type: application/json" \
  -d '{"data":"48656c6c6f20425356"}'
```

**Response:**
```json
{
  "uuid": "a1b2c3d4-5e6f-7g8h-9i0j-k1l2m3n4o5p6",
  "message": "Transaction queued for broadcast",
  "queueDepth": 1
}
```

### 2. JSON Data Example

```bash
# Create JSON payload
DATA=$(echo -n '{"type":"tweet","content":"Hello from BSV!"}' | xxd -p | tr -d '\n')

# Publish
curl -X POST http://localhost:8080/publish \
  -H "Content-Type: application/json" \
  -d "{\"data\":\"$DATA\"}"
```

### 3. File Upload Example

```bash
# Upload an image (small files only, OP_RETURN has size limits)
FILE_HEX=$(xxd -p small-image.png | tr -d '\n')

curl -X POST http://localhost:8080/publish \
  -H "Content-Type: application/json" \
  -d "{\"data\":\"$FILE_HEX\"}"
```

### 4. Using Makefile Helper

```bash
# Publish hex data directly
make publish DATA=48656c6c6f20425356
```

## Checking Status

### Check Specific Transaction

```bash
# Get the UUID from publish response
UUID="a1b2c3d4-5e6f-7g8h-9i0j-k1l2m3n4o5p6"

# Check status
curl http://localhost:8080/status/$UUID | jq

# Or use Makefile
make status UUID=$UUID
```

**Response Examples:**

**Pending (waiting for train):**
```json
{
  "uuid": "a1b2c3d4-...",
  "status": "pending",
  "createdAt": "2026-02-06T10:30:00Z",
  "updatedAt": "2026-02-06T10:30:00Z"
}
```

**Success (broadcasted):**
```json
{
  "uuid": "a1b2c3d4-...",
  "status": "success",
  "txid": "abc123def456...",
  "arcStatus": "SEEN_ON_NETWORK",
  "createdAt": "2026-02-06T10:30:00Z",
  "updatedAt": "2026-02-06T10:30:03Z"
}
```

**Mined (confirmed):**
```json
{
  "uuid": "a1b2c3d4-...",
  "status": "mined",
  "txid": "abc123def456...",
  "arcStatus": "MINED",
  "createdAt": "2026-02-06T10:30:00Z",
  "updatedAt": "2026-02-06T10:32:00Z"
}
```

## Monitoring

### Server Health

```bash
curl http://localhost:8080/health | jq

# Or
make health
```

**Response:**
```json
{
  "status": "healthy",
  "queueDepth": 15,
  "utxos": {
    "publishing_available": 48234,
    "publishing_locked": 42,
    "publishing_spent": 1724,
    "funding_available": 3
  }
}
```

### Detailed Statistics

```bash
curl http://localhost:8080/admin/stats | jq

# Or
make stats
```

### View Logs

```bash
# Server logs only
make logs

# All services
make logs-all

# Follow specific events
docker-compose logs -f bsv-publisher | grep "🚂"  # Train departures
docker-compose logs -f bsv-publisher | grep "🧹"  # Janitor cleanups
```

## Batch Publishing Example

```bash
#!/bin/bash
# batch-publish.sh - Publish 100 transactions

for i in {1..100}; do
  DATA=$(echo -n "Message #$i" | xxd -p | tr -d '\n')
  
  RESPONSE=$(curl -s -X POST http://localhost:8080/publish \
    -H "Content-Type: application/json" \
    -d "{\"data\":\"$DATA\"}")
  
  UUID=$(echo $RESPONSE | jq -r '.uuid')
  echo "[$i] UUID: $UUID"
  
  # Small delay to avoid overwhelming queue
  sleep 0.1
done

echo "✓ Submitted 100 transactions"
echo "  They will be batched and broadcast every 3 seconds"
```

Run it:
```bash
chmod +x batch-publish.sh
./batch-publish.sh
```

## Testing Train Behavior

### Watch the Train in Action

```bash
# Terminal 1: Watch logs
make logs | grep "🚂"

# Terminal 2: Send 10 transactions
for i in {1..10}; do
  make publish DATA=$(echo -n "Test $i" | xxd -p | tr -d '\n')
done

# Observe:
# - If < 1000 tx: Train departs after 3 seconds
# - If = 1000 tx: Train departs immediately
# - Status updates: pending → processing → success
```

### Test Graceful Shutdown

```bash
# Terminal 1: Watch logs
make logs

# Terminal 2: Send transactions then stop
for i in {1..50}; do
  make publish DATA=48656c6c6f
done

docker-compose stop bsv-publisher

# Observe:
# - Server finishes current batch before stopping
# - "Final departure: broadcasting X pending tx"
# - All transactions complete
```

## Production Usage

### Environment Variables

```bash
# .env for production
MONGO_PASSWORD=strong_random_password_here
ARC_TOKEN=your_production_arc_token
ARC_URL=https://arc.gorillapool.io
BSV_NETWORK=mainnet
TRAIN_INTERVAL=3s
TRAIN_MAX_BATCH=1000
TARGET_PUBLISHING_UTXOS=50000

# CRITICAL: Set these after first run
FUNDING_PRIVKEY=L...
PUBLISHING_PRIVKEY=K...
```

### Backup Keys

```bash
# Backup your .env (contains private keys!)
make backup-env

# Store backup securely (encrypted USB, password manager, etc.)
```

### Monitoring Alerts

Set up alerts for:

```bash
# Low UTXO count
if [[ $(curl -s http://localhost:8080/admin/stats | jq '.utxos.publishing_available') -lt 5000 ]]; then
  echo "⚠️  WARNING: Low on publishing UTXOs!"
fi

# High queue depth (backlog)
if [[ $(curl -s http://localhost:8080/health | jq '.queueDepth') -gt 5000 ]]; then
  echo "⚠️  WARNING: Queue backlog detected!"
fi
```

## Troubleshooting

### No Publishing UTXOs Available

```bash
# Check stats
make stats

# If publishing_available = 0:
# 1. Ensure you have funding UTXOs
# 2. Run the splitter (future admin endpoint)
# 3. Wait for split transactions to confirm
```

### ARC Connection Issues

```bash
# Check ARC health
curl http://localhost:8080/health

# Check logs for ARC errors
make logs | grep "ARC"

# Verify ARC_URL and ARC_TOKEN in .env
```

### Database Connection Failed

```bash
# Check MongoDB is running
docker ps | grep mongo

# Check connection string
docker-compose logs mongodb

# Reset database (WARNING: deletes all data)
make clean
make run
```

### Stuck Transactions

```bash
# Janitor runs every 10 minutes automatically
# Force immediate cleanup by restarting:
make restart

# Check logs for recovery
make logs | grep "🧹"
```

## Integration Examples

### Python Client

```python
import requests
import json
from binascii import hexlify

class BSVPublisher:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
    
    def publish(self, data: bytes) -> dict:
        hex_data = hexlify(data).decode()
        response = requests.post(
            f"{self.base_url}/publish",
            json={"data": hex_data}
        )
        return response.json()
    
    def check_status(self, uuid: str) -> dict:
        response = requests.get(f"{self.base_url}/status/{uuid}")
        return response.json()

# Usage
publisher = BSVPublisher()

# Publish
result = publisher.publish(b"Hello from Python!")
print(f"UUID: {result['uuid']}")

# Check status
import time
time.sleep(5)
status = publisher.check_status(result['uuid'])
print(f"Status: {status['status']}")
print(f"TxID: {status.get('txid', 'pending')}")
```

### JavaScript/Node.js Client

```javascript
const axios = require('axios');

class BSVPublisher {
  constructor(baseURL = 'http://localhost:8080') {
    this.client = axios.create({ baseURL });
  }

  async publish(data) {
    const hexData = Buffer.from(data).toString('hex');
    const response = await this.client.post('/publish', { data: hexData });
    return response.data;
  }

  async checkStatus(uuid) {
    const response = await this.client.get(`/status/${uuid}`);
    return response.data;
  }
}

// Usage
(async () => {
  const publisher = new BSVPublisher();
  
  // Publish
  const result = await publisher.publish('Hello from Node.js!');
  console.log('UUID:', result.uuid);
  
  // Wait and check
  await new Promise(resolve => setTimeout(resolve, 5000));
  const status = await publisher.checkStatus(result.uuid);
  console.log('Status:', status.status);
  console.log('TxID:', status.txid || 'pending');
})();
```

## Advanced Topics

### Custom Train Configuration

```bash
# Faster batching (1 second intervals)
TRAIN_INTERVAL=1s make run

# Larger batches (2000 transactions)
TRAIN_MAX_BATCH=2000 make run

# Adjust based on your throughput needs
```

### Multiple Publishers

You can run multiple instances with shared MongoDB for horizontal scaling:

```yaml
# docker-compose.scale.yml
services:
  bsv-publisher-1:
    # ... same config
  bsv-publisher-2:
    # ... same config, different port
  
  # Shared MongoDB
  mongodb:
    # ... same config
```

### Webhook Notifications (Future)

ARC supports webhooks for transaction status updates. Future versions will include:

```bash
# Set webhook URL in .env
WEBHOOK_URL=https://your-app.com/webhooks/bsv
```

---

**Need Help?** Check logs with `make logs` or open an issue on GitHub.
