# GovHash API Reference

Complete API documentation for the GovHash Broadcasting Engine.

**Base URL:** `https://api.govhash.org`  
**Version:** 1.0  
**Production Status:** ✅ Live

---

## Table of Contents

1. [Authentication](#authentication)
2. [Public Endpoints](#public-endpoints)
3. [Admin Endpoints](#admin-endpoints)
4. [Error Codes](#error-codes)
5. [Rate Limiting](#rate-limiting)
6. [Webhooks](#webhooks-future)

---

## Authentication

### API Key Authentication

All requests to `/publish` require an API key in the header:

```http
X-API-Key: gh_your_api_key_here
```

API keys are obtained through client registration (admin-only operation).

### ECDSA Signature Authentication

For enhanced security, requests must include an ECDSA signature:

```http
X-Signature: <hex_signature>
X-Timestamp: <unix_timestamp>
X-Nonce: <random_string>
```

**Signature Generation:**

1. Create message: `<timestamp>:<nonce>:<request_body_json>`
2. Hash with double SHA-256 (Bitcoin standard)
3. Sign with ECDSA using your private key
4. Encode signature as hex

**Example (Node.js):**

```javascript
const crypto = require('crypto');
const { PrivateKey } = require('bsv');

function signRequest(privateKeyWIF, timestamp, nonce, body) {
  const message = `${timestamp}:${nonce}:${JSON.stringify(body)}`;
  const hash = crypto.createHash('sha256')
    .update(crypto.createHash('sha256').update(message).digest())
    .digest();
  
  const privateKey = PrivateKey.fromWIF(privateKeyWIF);
  const signature = privateKey.sign(hash);
  return signature.toString('hex');
}
```

### Admin Authentication

Admin endpoints require the admin password:

```http
X-Admin-Password: your_admin_password
```

---

## Public Endpoints

### 1. Health Check

Get system health and UTXO statistics.

**Endpoint:** `GET /health`

**Authentication:** None required

**Response:**

```json
{
  "status": "healthy",
  "queueDepth": 0,
  "utxos": {
    "publishing_available": 49897,
    "publishing_spent": 2,
    "funding_available": 50
  }
}
```

**Status Codes:**
- `200 OK` - System healthy
- `503 Service Unavailable` - System unhealthy

---

### 2. Publish OP_RETURN

Submit data for blockchain broadcasting.

**Endpoint:** `POST /publish`

**Authentication:** Required (API Key + ECDSA Signature)

**Headers:**

```http
Content-Type: application/json
X-API-Key: gh_your_api_key_here
X-Signature: <hex_signature>
X-Timestamp: <unix_timestamp>
X-Nonce: <random_string>
```

**Request Body:**

```json
{
  "op_return": "Your data here (string or hex)",
  "metadata": {
    "custom": "optional fields"
  }
}
```

**Response (202 Accepted):**

```json
{
  "success": true,
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued",
  "message": "Request queued for broadcasting"
}
```

**Status Codes:**
- `202 Accepted` - Request queued successfully
- `400 Bad Request` - Invalid request body
- `401 Unauthorized` - Invalid API key or signature
- `403 Forbidden` - Client inactive or quota exceeded
- `429 Too Many Requests` - Rate limit exceeded
- `503 Service Unavailable` - No UTXOs available

**Error Response:**

```json
{
  "error": "Daily quota exceeded (10000 tx/day)"
}
```

---

### 3. Check Status

Poll the broadcast status of a transaction.

**Endpoint:** `GET /status/:uuid`

**Authentication:** None required (public lookup)

**Example:** `GET /status/550e8400-e29b-41d4-a716-446655440000`

**Response (Queued):**

```json
{
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued",
  "queued_at": "2026-02-06T18:30:00Z"
}
```

**Response (Broadcasting):**

```json
{
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "status": "broadcasting",
  "queued_at": "2026-02-06T18:30:00Z",
  "locked_at": "2026-02-06T18:30:03Z"
}
```

**Response (Complete):**

```json
{
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "status": "complete",
  "txid": "abc123...",
  "queued_at": "2026-02-06T18:30:00Z",
  "locked_at": "2026-02-06T18:30:03Z",
  "broadcasted_at": "2026-02-06T18:30:04Z"
}
```

**Response (Failed):**

```json
{
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "status": "failed",
  "error": "ARC returned error: insufficient fee",
  "queued_at": "2026-02-06T18:30:00Z",
  "locked_at": "2026-02-06T18:30:03Z"
}
```

**Status Values:**
- `queued` - Waiting for next train batch
- `broadcasting` - Currently being sent to ARC
- `complete` - Successfully broadcasted (txid available)
- `failed` - Broadcasting failed (error provided)

**Status Codes:**
- `200 OK` - Status retrieved
- `404 Not Found` - UUID not found

---

### 4. Detailed Statistics

Get detailed system statistics.

**Endpoint:** `GET /admin/stats`

**Authentication:** None required (public)

**Response:**

```json
{
  "utxos": {
    "publishing_available": 49897,
    "publishing_spent": 2,
    "funding_available": 50
  },
  "queue": {
    "depth": 0,
    "processing": true
  },
  "uptime": "3h15m",
  "version": "1.0.0"
}
```

---

## Admin Endpoints

All admin endpoints require the `X-Admin-Password` header.

### Client Management

#### 1. Register Client

Create a new API client with credentials.

**Endpoint:** `POST /admin/clients/register`

**Headers:**

```http
Content-Type: application/json
X-Admin-Password: your_admin_password
```

**Request Body:**

```json
{
  "name": "Client Name",
  "public_key": "02f3d17ca1ac6dcf42b0297a71abb87f79dfa2c66278caf9103135f7d0a7a9e8b8",
  "site_origin": "https://example.com",
  "max_daily_tx": 10000
}
```

**Fields:**
- `name` (required) - Human-readable client name
- `public_key` (required) - Compressed Bitcoin public key (hex, 66 chars)
- `site_origin` (optional) - Domain for CORS/tracking
- `max_daily_tx` (optional) - Daily transaction quota (default: 1000)

**Response (200 OK):**

```json
{
  "success": true,
  "message": "Client registered successfully. Save the API key - it will only be shown once!",
  "api_key": "gh_5-8MnZghnGpxKNMN1FTZlOvAnElGsa8dbyp0J7aKQI4=",
  "client": {
    "id": "698637fb3d19ed41cd8a2dd3",
    "name": "Client Name",
    "publicKey": "02f3d17ca...",
    "isActive": true,
    "siteOrigin": "https://example.com",
    "maxDailyTx": 10000,
    "txCount": 0,
    "lastResetDate": "2026-02-06",
    "createdAt": "2026-02-06T18:50:35Z",
    "updatedAt": "2026-02-06T18:50:35Z"
  }
}
```

**⚠️ Important:** The `api_key` is only shown once. Store it securely!

**Status Codes:**
- `200 OK` - Client registered
- `400 Bad Request` - Invalid request body
- `401 Unauthorized` - Invalid admin password
- `500 Internal Server Error` - Database error

---

#### 2. List Clients

Get all registered clients.

**Endpoint:** `GET /admin/clients/list`

**Headers:**

```http
X-Admin-Password: your_admin_password
```

**Response (200 OK):**

```json
{
  "success": true,
  "clients": [
    {
      "id": "698637fb3d19ed41cd8a2dd3",
      "name": "GovHash Production Client",
      "publicKey": "02f3d17ca...",
      "isActive": true,
      "siteOrigin": "https://govhash.org",
      "maxDailyTx": 50000,
      "txCount": 127,
      "lastResetDate": "2026-02-06",
      "createdAt": "2026-02-06T18:50:35Z",
      "updatedAt": "2026-02-06T19:15:00Z"
    }
  ]
}
```

---

#### 3. Activate Client

Enable a deactivated client.

**Endpoint:** `POST /admin/clients/:id/activate`

**Headers:**

```http
X-Admin-Password: your_admin_password
```

**Example:** `POST /admin/clients/698637fb3d19ed41cd8a2dd3/activate`

**Response (200 OK):**

```json
{
  "success": true,
  "message": "Client activated"
}
```

**Status Codes:**
- `200 OK` - Client activated
- `400 Bad Request` - Invalid client ID
- `401 Unauthorized` - Invalid admin password
- `500 Internal Server Error` - Database error

---

#### 4. Deactivate Client

Suspend client access (soft delete).

**Endpoint:** `POST /admin/clients/:id/deactivate`

**Headers:**

```http
X-Admin-Password: your_admin_password
```

**Example:** `POST /admin/clients/698637fb3d19ed41cd8a2dd3/deactivate`

**Response (200 OK):**

```json
{
  "success": true,
  "message": "Client deactivated"
}
```

---

### Maintenance

#### 5. Sweep UTXOs

Consolidate spent UTXOs to a destination address.

**Endpoint:** `POST /admin/maintenance/sweep`

**Headers:**

```http
Content-Type: application/json
X-Admin-Password: your_admin_password
```

**Request Body:**

```json
{
  "dest_address": "1YourBitcoinAddress...",
  "max_inputs": 100,
  "utxo_type": "publishing"
}
```

**Fields:**
- `dest_address` (required) - Destination Bitcoin address
- `max_inputs` (optional) - Max UTXOs per transaction (default: 100)
- `utxo_type` (optional) - "publishing" or "funding" (default: "publishing")

**Response (200 OK):**

```json
{
  "success": true,
  "txid": "abc123def456...",
  "amount": 0.00495000,
  "message": "UTXOs consolidated successfully"
}
```

**Status Codes:**
- `200 OK` - Sweep successful
- `400 Bad Request` - Invalid request
- `401 Unauthorized` - Invalid admin password
- `500 Internal Server Error` - Transaction failed

---

#### 6. Consolidate Dust

Combine small UTXOs to reduce database bloat.

**Endpoint:** `POST /admin/maintenance/consolidate-dust`

**Headers:**

```http
Content-Type: application/json
X-Admin-Password: your_admin_password
```

**Request Body:**

```json
{
  "funding_address": "1FundingAddress...",
  "max_inputs": 100
}
```

**Response (200 OK):**

```json
{
  "success": true,
  "txid": "def456abc123...",
  "amount": 0.00000546,
  "message": "Dust UTXOs consolidated successfully"
}
```

---

#### 7. Estimate Sweep Value

Calculate total value of UTXOs available to sweep.

**Endpoint:** `GET /admin/maintenance/estimate-sweep`

**Headers:**

```http
X-Admin-Password: your_admin_password
```

**Query Parameters:**
- `utxo_type` (optional) - "publishing" or "funding" (default: "publishing")
- `max_inputs` (optional) - Max UTXOs to estimate (default: 100)

**Example:** `GET /admin/maintenance/estimate-sweep?utxo_type=publishing&max_inputs=100`

**Response (200 OK):**

```json
{
  "success": true,
  "utxo_type": "publishing",
  "count": 2,
  "total_sats": 99,
  "total_bsv": 0.00000099
}
```

---

### Emergency

#### 8. Stop Train

Emergency kill switch - stops the broadcasting train worker.

**Endpoint:** `POST /admin/emergency/stop-train`

**Headers:**

```http
X-Admin-Password: your_admin_password
```

**Response (200 OK):**

```json
{
  "success": true,
  "message": "Train worker stopped. Restart server to resume."
}
```

**⚠️ Warning:** This stops all broadcasting. Requires server restart to resume.

---

#### 9. Check Train Status

Check if the train worker is running.

**Endpoint:** `GET /admin/emergency/status`

**Headers:**

```http
X-Admin-Password: your_admin_password
```

**Response (200 OK):**

```json
{
  "success": true,
  "running": true
}
```

---

## Error Codes

### HTTP Status Codes

| Code | Meaning | Common Causes |
|------|---------|---------------|
| 200 | OK | Request successful |
| 202 | Accepted | Request queued for processing |
| 400 | Bad Request | Invalid JSON, missing fields |
| 401 | Unauthorized | Invalid API key or signature |
| 403 | Forbidden | Client inactive or quota exceeded |
| 404 | Not Found | UUID or resource not found |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Database or system error |
| 503 | Service Unavailable | No UTXOs or system down |

### Error Response Format

```json
{
  "error": "Detailed error message here"
}
```

### Common Error Messages

**Authentication Errors:**
- `"Missing API key"` - No X-API-Key header
- `"Invalid API key"` - Key not found or incorrect
- `"Client is not active"` - Client has been deactivated
- `"Invalid signature"` - ECDSA verification failed
- `"Signature expired"` - Timestamp too old (>5 min)

**Quota Errors:**
- `"Daily quota exceeded (10000 tx/day)"` - Rate limit hit
- `"No publishing UTXOs available"` - System out of capacity

**Validation Errors:**
- `"Invalid request body"` - Malformed JSON
- `"op_return data is required"` - Missing required field
- `"name and public_key are required"` - Registration missing fields

---

## Rate Limiting

### Per-Client Limits

Each client has a daily transaction quota set during registration.

**Default:** 1,000 transactions/day  
**Maximum:** Configurable per client  
**Reset:** Midnight UTC daily

**Headers:**

```http
X-RateLimit-Limit: 10000
X-RateLimit-Remaining: 9873
X-RateLimit-Reset: 1675728000
```

### System-Wide Limits

- **UTXO Pool:** 50,000 concurrent operations
- **Queue Depth:** Unlimited (queued until UTXOs available)
- **Batch Size:** 1,000 transactions per train batch
- **Batch Interval:** 3 seconds

---

## Best Practices

### 1. Signature Security

- **Never expose private keys** in client-side code
- **Rotate keys** if compromised
- **Use unique nonces** for replay protection
- **Validate timestamps** (max 5 min clock skew)

### 2. Error Handling

```javascript
async function publishWithRetry(data, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await publish(data);
      return response;
    } catch (error) {
      if (error.status === 503) {
        // System at capacity, wait and retry
        await sleep(5000);
        continue;
      }
      if (error.status === 429) {
        // Rate limited, don't retry
        throw error;
      }
      // Other errors, retry
      await sleep(1000 * (i + 1));
    }
  }
  throw new Error('Max retries exceeded');
}
```

### 3. Status Polling

Poll every **3-5 seconds** after submission:

```javascript
async function waitForBroadcast(uuid, timeoutMs = 60000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const status = await checkStatus(uuid);
    if (status.status === 'complete') {
      return status.txid;
    }
    if (status.status === 'failed') {
      throw new Error(status.error);
    }
    await sleep(3000); // Poll every 3 seconds
  }
  throw new Error('Timeout waiting for broadcast');
}
```

### 4. Quota Management

Track your daily usage to avoid hitting limits:

```javascript
// Get current usage from /admin/clients/list
async function checkQuota() {
  const clients = await listClients();
  const myClient = clients.find(c => c.id === MY_CLIENT_ID);
  const remaining = myClient.maxDailyTx - myClient.txCount;
  console.log(`Remaining quota: ${remaining}/${myClient.maxDailyTx}`);
  return remaining;
}
```

---

## Webhooks (Future)

*Coming Soon:* Webhook notifications for transaction status changes.

**Planned Events:**
- `broadcast.queued` - Transaction accepted
- `broadcast.complete` - Transaction confirmed
- `broadcast.failed` - Transaction failed

---

## Support

**Documentation:** https://github.com/codenlighten/bsv-go-publisher  
**Issues:** https://github.com/codenlighten/bsv-go-publisher/issues  
**Email:** support@govhash.org

---

## Changelog

### v1.0 (2026-02-06)
- ✅ Initial production release
- ✅ 4-layer authentication
- ✅ Client management
- ✅ UTXO maintenance tools
- ✅ Emergency controls

---

**Last Updated:** February 6, 2026  
**API Version:** 1.0  
**Status:** Production ✅
