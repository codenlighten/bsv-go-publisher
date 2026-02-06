# Quick Start Guide

Get your BSV AKUA Broadcast Server running in under 5 minutes.

## Prerequisites

- **Docker** and **Docker Compose** installed
- **BSV ARC API token** (from [GorillaPool](https://www.gorillapool.io/) or [TAAL](https://taal.com/))
- **BSV wallet** with funds for initial setup

## Step-by-Step Setup

### 1. Configure Environment

```bash
cd /home/greg/dev/go-bsv-akua-broadcast

# Copy environment template
cp .env.example .env

# Edit configuration
nano .env
```

**Minimum required changes:**
```bash
MONGO_PASSWORD=your_secure_password_here
ARC_TOKEN=your_arc_api_token_here
```

**Optional configuration:**
```bash
ARC_URL=https://arc.gorillapool.io  # Default is fine
TRAIN_INTERVAL=3s                    # Batch every 3 seconds
TRAIN_MAX_BATCH=1000                 # Up to 1000 tx per batch
```

### 2. Start the Server

```bash
make run

# OR for development with Mongo Express UI:
make run-dev
```

**Expected output:**
```
✓ Services started
  API: http://localhost:8080
  MongoDB: localhost:27017

View logs with: make logs
```

### 3. Check Initial Status

```bash
# View server logs
make logs

# Should show:
# ⚠️  WARNING: No FUNDING_PRIVKEY found in environment!
# Generated new keypair:
#   Address: 1ABC...
#   Private Key (WIF): L1a2b3c...
```

**Important:** Copy the generated private keys and add them to your `.env`:

```bash
# Stop the server
make stop

# Edit .env and add:
nano .env

# Add these lines (use YOUR keys from the logs):
FUNDING_PRIVKEY=L1a2b3c...
PUBLISHING_PRIVKEY=K4d5e6f...

# Restart
make run
```

### 4. Fund the Server

Send BSV to your **funding address** (from step 3):

```bash
# Using BSV CLI:
bsv-cli sendtoaddress 1ABC... 0.1

# Or using any BSV wallet
# Send to: 1ABC...
# Amount: 0.1 BSV (10,000,000 sats)
```

Wait for 1-2 confirmations (~10-20 minutes).

### 5. Verify Setup

```bash
# Check health
make health

# Should show:
# {
#   "status": "healthy",
#   "utxos": {
#     "funding_available": 1  # Your funded UTXO
#   }
# }
```

### 6. Create Publishing UTXOs

**Note:** The splitter is implemented but requires manual integration. For testing, you can:

**Option A - Wait for Production Feature:**
```bash
# Future endpoint (not yet active):
curl -X POST http://localhost:8080/admin/split \
  -H "Content-Type: application/json" \
  -d '{"count": 1000}'
```

**Option B - Manual Test Setup:**

For immediate testing, you can manually create a few 100-sat UTXOs by:
1. Sending exactly 100 sats to your publishing address multiple times
2. Or implementing the splitter broadcast integration (see [STATUS.md](STATUS.md))

### 7. Test Broadcasting

Once you have publishing UTXOs available:

```bash
# Test publish
make publish DATA=48656c6c6f  # "Hello" in hex

# Response:
# {
#   "uuid": "abc123...",
#   "message": "Transaction queued for broadcast",
#   "queueDepth": 1
# }

# Check status (wait 3-5 seconds)
make status UUID=abc123...

# Response:
# {
#   "status": "success",
#   "txid": "def456..."
# }
```

### 8. Run Integration Tests

```bash
./test.sh

# Should show:
# ✓ Health check passed
# ✓ Stats endpoint accessible
# ✓ Publish request accepted
# ✓ Status endpoint working
# ✓ Error handling correct
# === All Tests Passed ===
```

## What's Next?

### Production Checklist

- [ ] **Secure Private Keys**: Move keys to a secrets manager (not `.env`)
- [ ] **Enable TLS**: Use nginx/Caddy reverse proxy with HTTPS
- [ ] **Authentication**: Protect `/admin/*` endpoints
- [ ] **Rate Limiting**: Prevent abuse
- [ ] **Monitoring**: Set up alerts for low UTXOs
- [ ] **Backup**: Backup `.env` and MongoDB data

### Development Workflow

```bash
# View logs
make logs

# Check UTXO stats
make stats

# Test publish
make publish DATA=$(echo -n "Your message" | xxd -p)

# Monitor train activity
make logs | grep "🚂"

# Check for recovered UTXOs
make logs | grep "🧹"
```

### Common Commands

```bash
make help              # Show all commands
make run               # Start services
make stop              # Stop services
make restart           # Restart server
make logs              # View server logs
make health            # Check health
make stats             # View UTXO statistics
make clean             # Remove all data (WARNING!)
```

## Architecture Overview

```
User Request
     ↓
  POST /publish (create raw tx, lock UTXO)
     ↓
  UUID Response (202 Accepted)
     ↓
  Queue (channel buffer)
     ↓
  🚂 Train (every 3s or 1000 tx)
     ↓
  ARC Batch Broadcast
     ↓
  Update Status in DB
     ↓
  GET /status/:uuid (check result)
```

## Key Features

- **Atomic UTXO Locking**: MongoDB FindAndModify ensures no double-spending
- **Train Batching**: Collects up to 1000 tx, broadcasts every 3 seconds
- **Graceful Shutdown**: Finishes current batch before stopping
- **Auto-Recovery**: Janitor unlocks stuck UTXOs every 10 minutes
- **Thread-Safe**: Supports massive concurrency with 50k UTXOs

## Troubleshooting

### "No publishing UTXOs available"

**Cause:** You haven't created publishing UTXOs yet  
**Solution:** Fund the server and run the splitter (see step 6)

### "ARC health check failed"

**Cause:** Invalid `ARC_TOKEN` or network issue  
**Solution:** Verify token in `.env`, check ARC service status

### "Database connection failed"

**Cause:** MongoDB not running or wrong password  
**Solution:** 
```bash
make stop
make run
make logs-all  # Check MongoDB logs
```

### Keys Keep Regenerating

**Cause:** Keys not persisted to `.env`  
**Solution:** Copy private keys from first startup to `.env` (see step 3)

## Monitoring Production

```bash
# Health check (returns 200 if healthy)
curl -f http://localhost:8080/health || echo "Server down!"

# Check UTXO count
AVAILABLE=$(curl -s http://localhost:8080/admin/stats | jq '.utxos.publishing_available')
if [ "$AVAILABLE" -lt 5000 ]; then
  echo "Warning: Low on UTXOs ($AVAILABLE remaining)"
fi

# Check queue depth
QUEUE=$(curl -s http://localhost:8080/health | jq '.queueDepth')
if [ "$QUEUE" -gt 5000 ]; then
  echo "Warning: Queue backlog ($QUEUE pending)"
fi
```

## Getting Help

- **Logs:** `make logs` - Check for errors
- **Health:** `make health` - Verify server status
- **Stats:** `make stats` - Check UTXO counts
- **Documentation:** See [README.md](README.md) and [EXAMPLES.md](EXAMPLES.md)

## Next Steps

1. **Read [EXAMPLES.md](EXAMPLES.md)** - Detailed usage examples
2. **Read [README.md](README.md)** - Full documentation
3. **Check [STATUS.md](STATUS.md)** - Implementation status and roadmap
4. **Implement Splitter** - See production integration notes

---

**Ready to broadcast to BSV at scale!** 🚀
