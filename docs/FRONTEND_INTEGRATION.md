# Frontend Integration Guide

**Connecting GovHash.org frontend to the API for real-time stats**

---

## Network Stats Bar Integration

Your GovHash.org landing page can display real-time network statistics by polling the health endpoint.

### Health Endpoint

```bash
GET https://api.govhash.org/health
```

**Response:**
```json
{
  "status": "healthy",
  "queueDepth": 0,
  "utxos": {
    "funding_available": 50,
    "publishing_available": 49899,
    "publishing_spent": 101
  }
}
```

### Frontend Implementation

#### Vanilla JavaScript

```html
<!-- Network Stats Component -->
<div class="network-stats">
  <div class="stat">
    <span class="stat-label">Broadcasting Capacity</span>
    <span class="stat-value" id="utxo-count">Loading...</span>
  </div>
  <div class="stat">
    <span class="stat-label">Queue Depth</span>
    <span class="stat-value" id="queue-depth">0</span>
  </div>
  <div class="stat">
    <span class="stat-label">System Status</span>
    <span class="stat-value status-healthy" id="status">Operational</span>
  </div>
</div>

<script>
async function updateNetworkStats() {
  try {
    const response = await fetch('https://api.govhash.org/health');
    const data = await response.json();
    
    // Update UTXO count
    const utxoCount = data.utxos.publishing_available.toLocaleString();
    document.getElementById('utxo-count').textContent = `${utxoCount} UTXOs`;
    
    // Update queue depth
    document.getElementById('queue-depth').textContent = data.queueDepth;
    
    // Update status
    const statusEl = document.getElementById('status');
    if (data.status === 'healthy') {
      statusEl.textContent = 'Operational';
      statusEl.className = 'stat-value status-healthy';
    } else {
      statusEl.textContent = 'Degraded';
      statusEl.className = 'stat-value status-warning';
    }
    
    // Show capacity warning if below 10k UTXOs
    if (data.utxos.publishing_available < 10000) {
      console.warn('Low UTXO pool capacity');
    }
  } catch (error) {
    console.error('Failed to fetch network stats:', error);
    document.getElementById('status').textContent = 'Offline';
    document.getElementById('status').className = 'stat-value status-error';
  }
}

// Update every 10 seconds
updateNetworkStats();
setInterval(updateNetworkStats, 10000);
</script>

<style>
.network-stats {
  display: flex;
  gap: 2rem;
  padding: 1.5rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 8px;
  color: white;
}

.stat {
  flex: 1;
  text-align: center;
}

.stat-label {
  display: block;
  font-size: 0.875rem;
  opacity: 0.9;
  margin-bottom: 0.5rem;
}

.stat-value {
  display: block;
  font-size: 1.5rem;
  font-weight: bold;
}

.status-healthy {
  color: #10b981;
}

.status-warning {
  color: #f59e0b;
}

.status-error {
  color: #ef4444;
}
</style>
```

#### React Implementation

```jsx
import React, { useState, useEffect } from 'react';

function NetworkStats() {
  const [stats, setStats] = useState({
    utxoCount: 0,
    queueDepth: 0,
    status: 'loading'
  });

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const response = await fetch('https://api.govhash.org/health');
        const data = await response.json();
        
        setStats({
          utxoCount: data.utxos.publishing_available,
          queueDepth: data.queueDepth,
          status: data.status === 'healthy' ? 'operational' : 'degraded'
        });
      } catch (error) {
        setStats(prev => ({ ...prev, status: 'offline' }));
      }
    };

    fetchStats();
    const interval = setInterval(fetchStats, 10000);
    
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="network-stats">
      <div className="stat">
        <span className="stat-label">Broadcasting Capacity</span>
        <span className="stat-value">
          {stats.utxoCount.toLocaleString()} UTXOs
        </span>
      </div>
      <div className="stat">
        <span className="stat-label">Queue Depth</span>
        <span className="stat-value">{stats.queueDepth}</span>
      </div>
      <div className="stat">
        <span className="stat-label">System Status</span>
        <span className={`stat-value status-${stats.status}`}>
          {stats.status.charAt(0).toUpperCase() + stats.status.slice(1)}
        </span>
      </div>
    </div>
  );
}

export default NetworkStats;
```

---

## Verification Portal Integration

### Check Transaction Status

```javascript
async function checkTransactionStatus(uuid) {
  const response = await fetch(`https://api.govhash.org/status/${uuid}`);
  const data = await response.json();
  
  return {
    status: data.status,        // "pending", "success", "failed"
    txid: data.txid,           // BSV transaction ID
    timestamp: data.created_at,
    blockchainUrl: data.txid 
      ? `https://whatsonchain.com/tx/${data.txid}` 
      : null
  };
}

// Example usage in verification form
document.getElementById('verify-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const uuid = document.getElementById('uuid-input').value;
  
  const result = await checkTransactionStatus(uuid);
  
  if (result.status === 'success') {
    document.getElementById('result').innerHTML = `
      <div class="success">
        ✓ Document Verified on Blockchain
        <a href="${result.blockchainUrl}" target="_blank">
          View Transaction: ${result.txid.slice(0, 8)}...
        </a>
      </div>
    `;
  } else if (result.status === 'pending') {
    document.getElementById('result').innerHTML = `
      <div class="pending">
        ⏳ Transaction Pending (typically 3-5 seconds)
      </div>
    `;
    // Poll again in 2 seconds
    setTimeout(() => checkTransactionStatus(uuid), 2000);
  } else {
    document.getElementById('result').innerHTML = `
      <div class="error">✗ Transaction Failed</div>
    `;
  }
});
```

---

## Admin Dashboard Integration

For internal admin panel (requires authentication):

### System Health Dashboard

```javascript
async function loadAdminDashboard() {
  const adminPassword = localStorage.getItem('admin_password');
  
  // Get system stats
  const healthResponse = await fetch('https://api.govhash.org/health');
  const health = await healthResponse.json();
  
  // Get client list
  const clientsResponse = await fetch('https://api.govhash.org/admin/clients/list', {
    headers: { 'X-Admin-Password': adminPassword }
  });
  const clients = await clientsResponse.json();
  
  // Get train status
  const trainResponse = await fetch('https://api.govhash.org/admin/emergency/status', {
    headers: { 'X-Admin-Password': adminPassword }
  });
  const train = await trainResponse.json();
  
  return {
    utxos: health.utxos,
    queueDepth: health.queueDepth,
    clients: clients.clients,
    trainRunning: train.running
  };
}
```

### Client Usage Table

```html
<table id="clients-table">
  <thead>
    <tr>
      <th>Client Name</th>
      <th>Daily Usage</th>
      <th>Status</th>
      <th>Actions</th>
    </tr>
  </thead>
  <tbody>
    <!-- Populated by JavaScript -->
  </tbody>
</table>

<script>
async function loadClients() {
  const response = await fetch('https://api.govhash.org/admin/clients/list', {
    headers: { 'X-Admin-Password': adminPassword }
  });
  const data = await response.json();
  
  const tbody = document.querySelector('#clients-table tbody');
  tbody.innerHTML = data.clients.map(client => `
    <tr>
      <td>${client.name}</td>
      <td>${client.tx_count} / ${client.max_daily_tx}</td>
      <td>
        <span class="badge ${client.is_active ? 'badge-success' : 'badge-danger'}">
          ${client.is_active ? 'Active' : 'Inactive'}
        </span>
      </td>
      <td>
        <button onclick="toggleClient('${client.id}', ${!client.is_active})">
          ${client.is_active ? 'Deactivate' : 'Activate'}
        </button>
      </td>
    </tr>
  `).join('');
}

async function toggleClient(clientId, activate) {
  const action = activate ? 'activate' : 'deactivate';
  await fetch(`https://api.govhash.org/admin/clients/${clientId}/${action}`, {
    method: 'POST',
    headers: { 'X-Admin-Password': adminPassword }
  });
  loadClients(); // Refresh table
}
</script>
```

---

## CORS Configuration

If your frontend is on a different domain, ensure CORS is enabled in your server:

### Add to main.go (if needed)

```go
import "github.com/gofiber/fiber/v2/middleware/cors"

func main() {
    app := fiber.New()
    
    // Enable CORS for your frontend domains
    app.Use(cors.New(cors.Config{
        AllowOrigins: "https://govhash.org,https://notaryhash.com",
        AllowHeaders: "Origin, Content-Type, Accept, X-API-Key, X-Signature",
        AllowMethods: "GET,POST",
    }))
    
    // ... rest of setup
}
```

---

## Progressive Web App (PWA) Support

For government auditors who need offline access to verification portal:

### service-worker.js

```javascript
const CACHE_NAME = 'govhash-v1';
const urlsToCache = [
  '/',
  '/index.html',
  '/verify.html',
  '/styles.css',
  '/app.js'
];

self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => cache.addAll(urlsToCache))
  );
});

self.addEventListener('fetch', event => {
  // Always fetch API requests from network
  if (event.request.url.includes('/api/')) {
    return fetch(event.request);
  }
  
  // Serve cached assets
  event.respondWith(
    caches.match(event.request)
      .then(response => response || fetch(event.request))
  );
});
```

---

## Real-Time Updates with WebSocket (Optional)

For advanced admin dashboard with live updates:

### Server-Side (future enhancement)

```go
// Add WebSocket endpoint for real-time stats
app.Get("/ws/admin", websocket.New(func(c *websocket.Conn) {
    // Verify admin authentication
    password := c.Query("password")
    if password != os.Getenv("ADMIN_PASSWORD") {
        c.Close()
        return
    }
    
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            stats := getSystemStats()
            c.WriteJSON(stats)
        }
    }
}))
```

### Client-Side

```javascript
const ws = new WebSocket('wss://api.govhash.org/ws/admin?password=xxx');

ws.onmessage = (event) => {
  const stats = JSON.parse(event.data);
  updateDashboard(stats);
};

ws.onerror = () => {
  console.error('WebSocket connection failed');
  // Fallback to polling
  setInterval(updateStats, 5000);
};
```

---

## Performance Considerations

### Caching Strategy

```javascript
// Cache health endpoint for 10 seconds to reduce load
let cachedHealth = null;
let cacheTime = 0;

async function getHealth() {
  const now = Date.now();
  if (cachedHealth && (now - cacheTime) < 10000) {
    return cachedHealth;
  }
  
  const response = await fetch('https://api.govhash.org/health');
  cachedHealth = await response.json();
  cacheTime = now;
  
  return cachedHealth;
}
```

### Rate Limiting on Frontend

```javascript
// Prevent users from spamming the verify button
let lastVerifyTime = 0;

function verifyDocument() {
  const now = Date.now();
  if (now - lastVerifyTime < 1000) {
    alert('Please wait a moment before verifying again');
    return;
  }
  lastVerifyTime = now;
  
  // Proceed with verification
}
```

---

## Example Full Page

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>GovHash - Government-Grade Blockchain Attestation</title>
  <style>
    /* Your existing styles */
  </style>
</head>
<body>
  <header>
    <h1>GovHash</h1>
    <p>Cryptographically Verified Document Attestation</p>
  </header>
  
  <!-- Live Network Stats -->
  <div class="network-stats" id="network-stats">
    <!-- Populated by JavaScript -->
  </div>
  
  <main>
    <section class="hero">
      <h2>Immutable. Verifiable. Permanent.</h2>
      <p>
        Every document hash is cryptographically signed and broadcast to the 
        Bitcoin SV blockchain, providing legally admissible proof of existence.
      </p>
      <button onclick="location.href='https://docs.govhash.org'">
        Get Started
      </button>
    </section>
    
    <section class="verify">
      <h3>Verify a Document</h3>
      <form id="verify-form">
        <input 
          type="text" 
          id="uuid-input" 
          placeholder="Enter transaction UUID"
          required
        />
        <button type="submit">Verify</button>
      </form>
      <div id="verify-result"></div>
    </section>
  </main>
  
  <script src="/js/govhash.js"></script>
</body>
</html>
```

---

## Testing Checklist

- [ ] Health endpoint loads without CORS errors
- [ ] Stats update every 10 seconds
- [ ] Verification form works with valid UUID
- [ ] Verification form shows appropriate error for invalid UUID
- [ ] Admin dashboard requires password
- [ ] Admin dashboard shows client list
- [ ] Mobile responsive design works
- [ ] Offline mode (PWA) caches static assets
- [ ] Page load time < 2 seconds
- [ ] Lighthouse score > 90

---

## Support

For frontend integration questions:
- **API Documentation:** [docs/SECURITY.md](SECURITY.md)
- **Quick Reference:** [docs/QUICK_REFERENCE.md](QUICK_REFERENCE.md)
- **Email:** support@govhash.org
