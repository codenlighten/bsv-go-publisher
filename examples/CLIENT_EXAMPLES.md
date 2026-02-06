# Client Examples

Examples of how to authenticate and send data to the GovHash API.

## JavaScript (Node.js)

```javascript
const axios = require('axios');
const bsv = require('@bsv/sdk');

// Your API credentials (from registration)
const API_KEY = 'gh_your_api_key_here';
const PRIVATE_KEY = 'your_private_key_wif_here';

async function publishData(dataHex) {
    // Create signature
    const privKey = bsv.PrivateKey.fromWif(PRIVATE_KEY);
    const dataBuffer = Buffer.from(dataHex, 'hex');
    
    // Double SHA-256 hash (Bitcoin standard)
    const hash1 = bsv.Hash.sha256(dataBuffer);
    const hash2 = bsv.Hash.sha256(hash1);
    
    const signature = privKey.sign(hash2);
    const sigHex = signature.toDER().toString('hex');

    // Send request
    try {
        const response = await axios.post('https://api.govhash.org/publish', 
            { data: dataHex },
            {
                headers: {
                    'Content-Type': 'application/json',
                    'X-API-Key': API_KEY,
                    'X-Signature': sigHex
                }
            }
        );

        console.log('Success:', response.data);
        return response.data.uuid;
    } catch (error) {
        console.error('Error:', error.response?.data || error.message);
        throw error;
    }
}

// Example usage
const message = "Hello, GovHash!";
const dataHex = Buffer.from(message).toString('hex');

publishData(dataHex)
    .then(uuid => {
        console.log('Published with UUID:', uuid);
        
        // Check status after a moment
        setTimeout(() => checkStatus(uuid), 5000);
    })
    .catch(err => console.error('Failed:', err));

async function checkStatus(uuid) {
    const response = await axios.get(`https://api.govhash.org/status/${uuid}`);
    console.log('Status:', response.data);
}
```

## JavaScript (Browser)

```html
<!DOCTYPE html>
<html>
<head>
    <title>GovHash Client</title>
    <script src="https://unpkg.com/@bsv/sdk"></script>
</head>
<body>
    <h1>GovHash Publisher</h1>
    <textarea id="message" placeholder="Enter your message"></textarea>
    <button onclick="publish()">Publish</button>
    <div id="result"></div>

    <script>
        const API_KEY = 'gh_your_api_key_here';
        const PRIVATE_KEY = 'your_private_key_wif_here';

        async function publish() {
            const message = document.getElementById('message').value;
            const dataHex = stringToHex(message);
            
            // Create signature
            const privKey = bsv.PrivateKey.fromWif(PRIVATE_KEY);
            const dataBuffer = hexToBuffer(dataHex);
            
            const hash1 = bsv.Hash.sha256(dataBuffer);
            const hash2 = bsv.Hash.sha256(hash1);
            
            const signature = privKey.sign(hash2);
            const sigHex = signature.toDER().toString('hex');

            // Send request
            try {
                const response = await fetch('https://api.govhash.org/publish', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-API-Key': API_KEY,
                        'X-Signature': sigHex
                    },
                    body: JSON.stringify({ data: dataHex })
                });

                const result = await response.json();
                document.getElementById('result').innerHTML = 
                    `<p>Published! UUID: ${result.uuid}</p>`;
            } catch (error) {
                document.getElementById('result').innerHTML = 
                    `<p style="color: red;">Error: ${error.message}</p>`;
            }
        }

        function stringToHex(str) {
            return Array.from(str).map(c => 
                c.charCodeAt(0).toString(16).padStart(2, '0')
            ).join('');
        }

        function hexToBuffer(hex) {
            const bytes = new Uint8Array(hex.length / 2);
            for (let i = 0; i < hex.length; i += 2) {
                bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
            }
            return bytes;
        }
    </script>
</body>
</html>
```

## Python

```python
import requests
import hashlib
from bsv import PrivateKey

# Your API credentials
API_KEY = 'gh_your_api_key_here'
PRIVATE_KEY_WIF = 'your_private_key_wif_here'

def publish_data(data_hex: str) -> str:
    """Publish data to GovHash with authentication"""
    
    # Load private key
    priv_key = PrivateKey.from_wif(PRIVATE_KEY_WIF)
    
    # Create signature (double SHA-256)
    data_bytes = bytes.fromhex(data_hex)
    hash1 = hashlib.sha256(data_bytes).digest()
    hash2 = hashlib.sha256(hash1).digest()
    
    signature = priv_key.sign(hash2)
    sig_hex = signature.hex()
    
    # Send request
    response = requests.post(
        'https://api.govhash.org/publish',
        json={'data': data_hex},
        headers={
            'Content-Type': 'application/json',
            'X-API-Key': API_KEY,
            'X-Signature': sig_hex
        }
    )
    
    if response.status_code == 200:
        result = response.json()
        print(f"Success! UUID: {result['uuid']}")
        return result['uuid']
    else:
        print(f"Error: {response.json()}")
        raise Exception(response.json().get('error', 'Unknown error'))

def check_status(uuid: str):
    """Check the status of a published transaction"""
    response = requests.get(f'https://api.govhash.org/status/{uuid}')
    return response.json()

# Example usage
if __name__ == '__main__':
    message = "Hello, GovHash!"
    data_hex = message.encode().hex()
    
    uuid = publish_data(data_hex)
    
    # Wait a moment and check status
    import time
    time.sleep(5)
    
    status = check_status(uuid)
    print(f"Status: {status}")
```

## Go

```go
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bsv-blockchain/go-sdk/ec"
)

const (
	APIKey     = "gh_your_api_key_here"
	PrivateKey = "your_private_key_wif_here"
	BaseURL    = "https://api.govhash.org"
)

type PublishRequest struct {
	Data string `json:"data"`
}

type PublishResponse struct {
	Success bool   `json:"success"`
	UUID    string `json:"uuid"`
}

func publishData(dataHex string) (string, error) {
	// Parse private key
	privKey, err := ec.PrivateKeyFromWif(PrivateKey)
	if err != nil {
		return "", err
	}

	// Double SHA-256 hash
	dataBytes, _ := hex.DecodeString(dataHex)
	hash1 := sha256.Sum256(dataBytes)
	hash2 := sha256.Sum256(hash1[:])

	// Sign
	sig, err := privKey.Sign(hash2[:])
	if err != nil {
		return "", err
	}
	sigHex := hex.EncodeToString(sig.Serialise())

	// Create request
	payload := PublishRequest{Data: dataHex}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", BaseURL+"/publish", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", APIKey)
	req.Header.Set("X-Signature", sigHex)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result PublishResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success {
		return "", fmt.Errorf("publish failed")
	}

	return result.UUID, nil
}

func main() {
	message := "Hello, GovHash!"
	dataHex := hex.EncodeToString([]byte(message))

	uuid, err := publishData(dataHex)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Published! UUID: %s\n", uuid)
}
```

## Authentication Flow

1. **Get Credentials**: Contact admin to register and receive:
   - API Key (e.g., `gh_abc123...`)
   - Your public key is registered on the server

2. **Sign Each Request**:
   - Convert your data to hex
   - Hash the data with double SHA-256 (Bitcoin standard)
   - Sign the hash with your private key (ECDSA)
   - Send data + signature in headers

3. **Rate Limits**:
   - Default: 1000 transactions per day
   - Counter resets at midnight UTC
   - Contact admin to increase limits

4. **Error Codes**:
   - `401`: Invalid API key or signature
   - `403`: Account disabled
   - `429`: Daily limit exceeded
   - `200`: Success

## Testing with curl

```bash
# This won't work directly because signature generation requires cryptography
# Use one of the client libraries above

curl -X POST https://api.govhash.org/publish \
  -H "Content-Type: application/json" \
  -H "X-API-Key: gh_your_key_here" \
  -H "X-Signature: your_signature_here" \
  -d '{"data": "48656c6c6f"}'
```

## Support

For API access or questions:
- Website: https://govhash.org
- Email: support@govhash.org
