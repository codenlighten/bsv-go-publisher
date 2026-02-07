# Synchronous Wait Mode (`?wait=true`)

## Overview

The GovHash API now supports **synchronous responses** for the `/publish` endpoint, eliminating the need for polling in most cases. By adding `?wait=true` to your request, the API will wait up to 5 seconds for the transaction to broadcast and return the `txid` immediately.

## Why This Matters

**Before (Async only):**
```
POST /publish → 202 Accepted + UUID
↓
Poll GET /status/:uuid every 3s
↓ (3-6 seconds later)
Get txid
```

**After (with `?wait=true`):**
```
POST /publish?wait=true
↓ (waits 0-5 seconds)
201 Created + txid immediately
```

## How It Works

### Architecture

1. **Client sends** `POST /publish?wait=true`
2. **Server creates** a Go channel attached to the request
3. **Train worker** processes the batch (within 3s)
4. **Worker notifies** the waiting client via the channel
5. **API returns** `201 Created` with txid

### Intelligent Fallback

If the queue is already full (>1000 transactions) or the broadcast takes longer than the timeout, the system automatically falls back to async mode (202 + UUID).

## Configuration

### Environment Variable

```bash
# .env
SYNC_WAIT_TIMEOUT=5s  # Default: 5 seconds
```

You can adjust this based on your network conditions. Recommended range: 3-10 seconds.

## Usage Examples

### JavaScript (Synchronous Upload)

```javascript
async function publishDocument(hexData) {
  const response = await fetch('https://api.govhash.org/publish?wait=true', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': 'gh_your_api_key',
      'X-Signature': generateSignature(...),
      'X-Timestamp': Date.now().toString(),
      'X-Nonce': crypto.randomUUID(),
    },
    body: JSON.stringify({ op_return: hexData }),
  });

  const result = await response.json();

  if (response.status === 201) {
    // Success! Got txid immediately
    console.log('Transaction ID:', result.txid);
    console.log('ARC Status:', result.arc_status);
    return result.txid;
  } else if (response.status === 202) {
    // Queue was busy, fell back to async
    console.log('Polling for result...');
    return await pollForResult(result.uuid);
  } else {
    // Error occurred
    throw new Error(result.error);
  }
}
```

### Python (Government Document Attestation)

```python
import requests
import hashlib
import time

def attest_document(document_hash):
    response = requests.post(
        'https://api.govhash.org/publish',
        params={'wait': 'true'},  # Enable sync mode
        headers={
            'Content-Type': 'application/json',
            'X-API-Key': API_KEY,
            'X-Signature': generate_signature(...),
            'X-Timestamp': str(int(time.time())),
            'X-Nonce': str(uuid.uuid4()),
        },
        json={'op_return': document_hash}
    )

    if response.status_code == 201:
        # Got txid immediately!
        data = response.json()
        print(f"✓ Document attested: {data['txid']}")
        return data['txid']
    elif response.status_code == 202:
        # Fell back to async
        uuid = response.json()['uuid']
        return poll_status(uuid)
    else:
        raise Exception(f"Attestation failed: {response.json()['error']}")
```

### Go (Batch Processing with Fallback)

```go
func publishWithSyncWait(client *http.Client, data string) (string, error) {
    req, _ := http.NewRequest("POST", 
        "https://api.govhash.org/publish?wait=true", 
        bytes.NewBuffer([]byte(`{"op_return":"`+data+`"}`)))
    
    req.Header.Set("X-API-Key", apiKey)
    req.Header.Set("X-Signature", signature)
    req.Header.Set("X-Timestamp", timestamp)
    req.Header.Set("X-Nonce", nonce)

    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)

    switch resp.StatusCode {
    case 201:
        // Got txid immediately
        return result["txid"].(string), nil
    case 202:
        // Fell back to async
        uuid := result["uuid"].(string)
        return pollStatus(uuid)
    default:
        return "", fmt.Errorf("error: %s", result["error"])
    }
}
```

## Response Codes

| Code | Meaning | When It Happens |
|------|---------|-----------------|
| `201 Created` | Success! Transaction broadcasted | Train completed within timeout |
| `202 Accepted` | Queued (async fallback) | Queue >1000 or timeout exceeded |
| `400 Bad Request` | Client error | Malformed data, invalid signature |
| `502 Bad Gateway` | ARC error | ARC service unavailable |
| `500 Internal Server Error` | System error | Database or internal failure |

## When to Use Sync vs Async

### ✅ Use Synchronous Mode (`?wait=true`)

- **Web Forms:** User uploads a document and expects immediate confirmation
- **Interactive Apps:** Real-time feedback required
- **Manual Operations:** Admin panel uploads
- **Low-Volume:** <100 requests/minute
- **User-Facing:** When showing a loading spinner to users

### ✅ Use Async Mode (Default)

- **High-Volume:** Batch processing 1000+ documents
- **Background Jobs:** Scheduled tasks, cron jobs
- **Automated Systems:** No user waiting for response
- **Reliability Critical:** Don't want to risk HTTP timeouts

## Performance Considerations

### Latency

- **Best case:** 0-3 seconds (next train cycle)
- **Average case:** 3-5 seconds
- **Worst case:** Falls back to async (>5s)

### Throughput

Synchronous mode does **not** reduce throughput. The train worker still processes 1000 transactions every 3 seconds. The only difference is that clients wait for the result instead of polling.

**Capacity:** Still 300-500 tx/sec

### Connection Limits

Each synchronous request holds an HTTP connection open for up to 5 seconds. Most servers can handle thousands of concurrent connections, but if you're sending 10,000+ simultaneous requests, consider using async mode to free up connections faster.

## Technical Implementation

### Go Channels

The implementation uses Go's native channels for thread-safe communication:

```go
// In API handler
responseChan := make(chan BroadcastResult, 1)
request.ResponseChan = responseChan

select {
case result := <-responseChan:
    // Got result, return to client
case <-time.After(5 * time.Second):
    // Timeout, return 202
}
```

```go
// In Train worker
if request.ResponseChan != nil {
    select {
    case request.ResponseChan <- result:
        // Client notified
    default:
        // Client already timed out, no-op
    }
}
```

### Non-Blocking Send

The train worker uses a **non-blocking send** to prevent deadlocks if the client disconnects early:

```go
select {
case request.ResponseChan <- result:
default:
    // Channel closed or client gone
}
```

## Monitoring

### Logs

The server logs sync mode activity:

```
⚠️  Queue is full (1234), falling back to async mode
⚠️  Sync wait timeout for abc-123-def, falling back to async
✓ Batch complete: 847 success, 0 failed
```

### Metrics to Watch

- **Timeout rate:** If >10% of sync requests timeout, consider increasing `SYNC_WAIT_TIMEOUT`
- **Queue depth:** If frequently >1000, you may need more UTXOs or faster train intervals
- **Average response time:** Should be 3-5s in sync mode

## Troubleshooting

### Problem: All sync requests returning 202

**Cause:** Queue depth consistently >1000

**Solution:**
1. Check UTXO pool: `curl https://api.govhash.org/health`
2. Increase publishing UTXOs if needed
3. Consider reducing `TRAIN_INTERVAL` (e.g., 2s instead of 3s)

### Problem: Frequent timeouts (5s+)

**Cause:** ARC is slow or network latency high

**Solution:**
1. Increase `SYNC_WAIT_TIMEOUT` to 7-10s
2. Check ARC health: `curl https://arc.gorillapool.io/health`
3. Consider using a closer ARC node

### Problem: HTTP 502 errors

**Cause:** ARC service unavailable

**Solution:**
1. Check ARC status
2. Verify `ARC_TOKEN` is valid
3. Switch to backup ARC endpoint if available

## Migration Guide

### Existing Code (Async)

```javascript
// Old code - still works!
const response = await fetch('/publish', {...});
const { uuid } = await response.json();
const txid = await pollForResult(uuid);
```

### New Code (Synchronous)

```javascript
// New code - instant txid
const response = await fetch('/publish?wait=true', {...});
const result = await response.json();

if (response.status === 201) {
  const txid = result.txid; // Got it immediately!
} else {
  const txid = await pollForResult(result.uuid); // Fallback
}
```

**No breaking changes!** All existing code continues to work.

## Security

Synchronous mode does **not** bypass any security checks:

- ✅ API key validation
- ✅ ECDSA signature verification
- ✅ Rate limiting
- ✅ Client activation status
- ✅ Daily quota enforcement

The only difference is **when** you receive the response.

## Future Enhancements

Planned improvements for sync mode:

1. **WebSocket Support:** Push notifications instead of long-polling
2. **Server-Sent Events (SSE):** Stream updates for multiple transactions
3. **Adaptive Timeout:** Automatically adjust based on current queue depth
4. **Priority Queue:** Premium clients get faster processing

## Conclusion

The synchronous wait mode (`?wait=true`) transforms the GovHash API from a "queue-and-poll" system to an **instant feedback** system for most use cases. It's perfect for interactive applications while maintaining the high-throughput batching architecture needed for enterprise scale.

**Use it when you want instant results. Fall back to async when you need maximum throughput.**

---

**Version:** 1.0  
**Added:** February 7, 2026  
**Status:** Production Ready ✅
