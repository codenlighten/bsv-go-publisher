# GovHash API - Client Integration Guide

**Version:** 1.0  
**Production URL:** `https://api.govhash.org`  
**Support:** support@govhash.org

---

## Overview

GovHash provides high-throughput Bitcoin SV (BSV) OP_RETURN broadcasting with guaranteed delivery. Our infrastructure handles 50,000+ concurrent transactions with sub-5-second latency.

**Key Features:**
- ✅ Enterprise-grade reliability (99.9% uptime SLA)
- ✅ Cryptographic non-repudiation (ECDSA signature verification)
- ✅ Real-time status tracking
- ✅ Tier-based rate limiting (1K-100K daily transactions)
- ✅ Domain isolation for multi-tenant security

---

## Getting Started

### 1. Request API Access

Contact your account manager or email support@govhash.org with:
- Company name
- Technical contact email
- Expected daily transaction volume
- Use case description

You'll receive:
- **API Key** (e.g., `gh_abc123...`) - shown only once, store securely
- **Public Key Registration** - your ECDSA public key registered on our system
- **Daily Transaction Limit** - based on your tier (Pilot/Enterprise/Government)

---

## Authentication

GovHash uses **dual-layer authentication** for maximum security:

### Layer 1: API Key (Required)
Every request must include your API key in the header:
```http
X-API-Key: gh_your_api_key_here
```

### Layer 2: ECDSA Signature (Enterprise/Government Tiers)
Sign your data payload with your private key:
```http
X-Signature: <hex_encoded_der_signature>
X-Timestamp: <unix_timestamp_ms>
X-Nonce: <random_string>
```

**Signature Algorithm:**
1. Convert your hex data to bytes
2. Double SHA-256 hash (Bitcoin standard)
3. Sign the hash with your ECDSA private key
4. Encode signature to DER format, then hex

---

## Core Endpoints

### 1. Publish OP_RETURN Data

**Endpoint:** `POST /publish`

**Request:**
```json
{
  "data": "48656c6c6f20576f726c64"  // hex-encoded payload
}
```

**Headers:**
```http
Content-Type: application/json
X-API-Key: gh_your_api_key
X-Signature: 304502...  // Enterprise+ only
X-Timestamp: 1738880000000
X-Nonce: random123
```

**Response (202 Accepted):**
```json
{
  "success": true,
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Request queued for broadcasting"
}
```

**Rate Limits:**
- **Pilot Tier:** 1,000 tx/day
- **Enterprise Tier:** 10,000 tx/day
- **Government Tier:** 100,000 tx/day

---

### 2. Check Broadcast Status

**Endpoint:** `GET /status/:uuid`

**Response:**
```json
{
  "uuid": "550e8400-...",
  "status": "mined",
  "txid": "abc123...",
  "created_at": "2026-02-08T12:00:00Z",
  "updated_at": "2026-02-08T12:00:03Z"
}
```

**Status Values:**
- `pending` - Queued, waiting for train departure
- `broadcasting` - Submitted to ARC network
- `success` - Accepted by ARC, propagating to miners
- `mined` - Confirmed in blockchain
- `failed` - Broadcast failed (see error field)

---

### 3. Health Check

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "queueDepth": 42,
  "utxos": {
    "funding_available": 50,
    "publishing_available": 48523
  }
}
```

Use this endpoint for monitoring and alerting.

---

## Code Examples

### JavaScript/Node.js (with @bsv/sdk)

```javascript
const bsv = require('@bsv/sdk');
const axios = require('axios');

const API_KEY = 'gh_your_key_here';
const PRIVATE_KEY_WIF = 'L...';  // Your ECDSA private key

async function publishData(hexData) {
  // Sign the data
  const privKey = bsv.PrivateKey.fromWif(PRIVATE_KEY_WIF);
  const dataBuffer = Buffer.from(hexData, 'hex');
  
  // Double SHA-256 (Bitcoin standard)
  const hash1 = bsv.Hash.sha256(dataBuffer);
  const hash2 = bsv.Hash.sha256(hash1);
  
  // Sign and encode
  const signature = privKey.sign(hash2);
  const sigHex = signature.toDER().toString('hex');
  
  // Publish
  const response = await axios.post('https://api.govhash.org/publish', 
    { data: hexData },
    {
      headers: {
        'X-API-Key': API_KEY,
        'X-Signature': sigHex,
        'X-Timestamp': Date.now().toString(),
        'X-Nonce': Math.random().toString(36).substring(7)
      }
    }
  );
  
  console.log('UUID:', response.data.uuid);
  return response.data.uuid;
}

// Usage
const hexData = Buffer.from('Hello GovHash').toString('hex');
publishData(hexData).then(uuid => {
  console.log(`Published! Track status at: /status/${uuid}`);
});
```

### Python (with python-bitcoinlib)

```python
import requests
import hashlib
import time
import random
import string
from bitcoinlib.keys import Key

API_KEY = 'gh_your_key_here'
PRIVATE_KEY_WIF = 'L...'

def publish_data(hex_data):
    # Sign the data
    key = Key(PRIVATE_KEY_WIF)
    data_bytes = bytes.fromhex(hex_data)
    
    # Double SHA-256
    hash1 = hashlib.sha256(data_bytes).digest()
    hash2 = hashlib.sha256(hash1).digest()
    
    # Sign
    signature = key.sign(hash2)
    sig_hex = signature.hex()
    
    # Publish
    response = requests.post('https://api.govhash.org/publish',
        json={'data': hex_data},
        headers={
            'X-API-Key': API_KEY,
            'X-Signature': sig_hex,
            'X-Timestamp': str(int(time.time() * 1000)),
            'X-Nonce': ''.join(random.choices(string.ascii_letters, k=10))
        }
    )
    
    return response.json()['uuid']

# Usage
hex_data = "Hello GovHash".encode().hex()
uuid = publish_data(hex_data)
print(f"Published! UUID: {uuid}")
```

### Go

```go
package main

import (
    "bytes"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "math/rand"
    "net/http"
    "time"
)

const (
    APIURL = "https://api.govhash.org"
    APIKey = "gh_your_key_here"
)

func publishData(hexData string) (string, error) {
    // Sign the data (using your ECDSA library)
    // ... signature generation code ...
    
    payload := map[string]string{"data": hexData}
    body, _ := json.Marshal(payload)
    
    req, _ := http.NewRequest("POST", APIURL+"/publish", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-API-Key", APIKey)
    req.Header.Set("X-Signature", signatureHex)
    req.Header.Set("X-Timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))
    req.Header.Set("X-Nonce", randomString(10))
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result["uuid"].(string), nil
}
```

---

## Error Handling

### HTTP Status Codes

| Code | Meaning | Action |
|------|---------|--------|
| 202 | Accepted | Data queued successfully |
| 400 | Bad Request | Check your data format (must be hex) |
| 401 | Unauthorized | Invalid API key or signature |
| 403 | Forbidden | Account disabled or signature mismatch |
| 429 | Rate Limited | Daily transaction limit exceeded |
| 500 | Server Error | Contact support with request UUID |

### Error Response Format

```json
{
  "error": "Invalid cryptographic signature",
  "code": "SIGNATURE_INVALID"
}
```

**Common Error Codes:**
- `MISSING_API_KEY` - X-API-Key header not provided
- `INVALID_API_KEY` - API key not found or inactive
- `SIGNATURE_INVALID` - ECDSA signature verification failed
- `RATE_LIMIT_EXCEEDED` - Daily quota exhausted (resets midnight UTC)
- `INVALID_DATA_FORMAT` - Data must be hex-encoded

---

## Best Practices

### 1. Signature Verification
- Always use double SHA-256 (Bitcoin standard)
- Sign the raw hex data, not the JSON payload
- Include timestamp and nonce to prevent replay attacks
- Keep private keys secure (never expose in client-side code)

### 2. Error Handling
- Implement exponential backoff for 500-level errors
- Cache API keys securely (environment variables, not code)
- Log failed requests with UUID for support debugging
- Monitor your daily transaction count to avoid 429 errors

### 3. Performance
- Batch multiple operations in parallel (GovHash handles 50K concurrent)
- Poll `/status/:uuid` every 3-5 seconds (not faster)
- Use webhooks (coming soon) instead of polling for high-volume use cases
- Cache successful TXIDs to avoid duplicate publishing

### 4. Security
- Rotate API keys quarterly
- Use separate keys per environment (dev/staging/prod)
- Whitelist your server IPs (contact support)
- Never log signatures or private keys

---

## Rate Limits & Quotas

### Daily Transaction Limits

| Tier | Daily Limit | Cost per 1K TX |
|------|-------------|----------------|
| Pilot | 1,000 | Included |
| Enterprise | 10,000 | $50/month |
| Government | 100,000 | Custom pricing |

**Limit Resets:** Midnight UTC  
**Overage:** Contact your account manager for temporary increases

### Concurrent Requests
- **Max concurrent:** Unlimited (system handles 50K+)
- **Rate limiting:** None for authenticated requests
- **Burst protection:** Automatic train batching every 3 seconds

---

## Monitoring & Alerts

### Recommended Monitoring

1. **Health Check Polling**
   ```bash
   curl https://api.govhash.org/health
   ```
   - Alert if `status != "healthy"`
   - Alert if `publishing_available < 1000`

2. **Daily Quota Tracking**
   - Track your transaction count internally
   - Alert at 80% of daily limit
   - Plan for limit increases before hitting 100%

3. **Latency Monitoring**
   - Measure time from `/publish` to `status=mined`
   - Expected: 3-8 seconds average
   - Alert if p95 latency > 15 seconds

---

## Support & Resources

### Technical Support
- **Email:** support@govhash.org
- **Response Time:** < 4 hours (business hours)
- **Emergency:** +1 (555) 123-4567 (Enterprise+ tiers)

### Documentation
- **API Reference:** https://api.govhash.org/docs
- **Status Page:** https://status.govhash.org
- **Changelog:** https://api.govhash.org/changelog

### Community
- **GitHub:** https://github.com/govhash/examples
- **Discord:** https://discord.gg/govhash (coming soon)

---

## Troubleshooting

### "Invalid cryptographic signature"
- Verify you're using double SHA-256
- Check signature format (DER-encoded hex)
- Ensure timestamp is within 5 minutes of server time
- Confirm public key registered matches your private key

### "Daily transaction limit exceeded"
- Check `/admin/clients/list` for your current count
- Limit resets at midnight UTC
- Contact support for temporary increase
- Consider upgrading your tier

### "Request not found" on /status/:uuid
- UUID may be expired (older than 7 days)
- Check UUID format (36-character lowercase with dashes)
- Verify you're using the UUID returned from `/publish`

### High Latency (>10 seconds)
- Normal during peak hours (3-8 seconds typical)
- Check `/health` for UTXO availability
- Monitor ARC network status at https://arc.gorillapool.io

---

## SLA & Uptime

**Service Level Agreement:**
- 99.9% uptime guarantee (Enterprise+ tiers)
- < 5 second average latency (publish to mined)
- 100% data integrity (cryptographic proof on-chain)

**Maintenance Windows:**
- Sundays 03:00-04:00 UTC (weekly UTXO consolidation)
- Advance notice via email for planned downtime

**Incident Response:**
- P0 (system down): < 15 minutes
- P1 (degraded): < 1 hour
- P2 (performance): < 4 hours

---

## Changelog

### Version 1.0 (February 8, 2026)
- ✅ Production launch with adaptive tier security
- ✅ ECDSA signature verification
- ✅ Real-time admin dashboard
- ✅ 50,000 UTXO pool capacity

---

**Questions?** Contact your account manager or support@govhash.org
