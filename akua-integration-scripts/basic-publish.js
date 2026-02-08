#!/usr/bin/env node

/**
 * GovHash Basic Publish Script
 * 
 * Simple, production-ready script to publish a single transaction
 * and retrieve the TXID using the wait=true parameter for instant response.
 * 
 * Usage:
 *   node basic-publish.js "Your message here"
 *   node basic-publish.js --data="48656c6c6f"  # hex format
 * 
 * Environment:
 *   GOVHASH_API_KEY - Your API key (required)
 *   GOVHASH_API_URL - API endpoint (optional, defaults to https://api.govhash.org)
 * 
 * Exit codes:
 *   0 - Success
 *   1 - Error (missing key, network error, etc)
 */

const https = require('https');
const { URL } = require('url');
require('dotenv').config();

// Configuration
const API_KEY = process.env.GOVHASH_API_KEY || '';
const API_URL = process.env.GOVHASH_API_URL || 'https://api.govhash.org';

// Constants
const REQUEST_TIMEOUT = 30000; // 30 seconds
const WAIT_PARAM = true; // Use instant TXID mode

/**
 * Convert text to hex format
 */
function textToHex(text) {
  return Buffer.from(text).toString('hex');
}

/**
 * Check if string is valid hex
 */
function isValidHex(str) {
  return /^[0-9a-fA-F]*$/.test(str) && str.length % 2 === 0;
}

/**
 * Make HTTPS request to GovHash API
 */
function makeRequest(method, path, body = null) {
  return new Promise((resolve, reject) => {
    const url = new URL(path, API_URL);
    
    const options = {
      hostname: url.hostname,
      port: url.port || 443,
      path: url.pathname + url.search,
      method: method,
      headers: {
        'X-API-Key': API_KEY,
        'Content-Type': 'application/json',
        'User-Agent': 'GovHash-AKUA-Client/1.0',
      },
      timeout: REQUEST_TIMEOUT,
    };

    if (body) {
      const bodyStr = JSON.stringify(body);
      options.headers['Content-Length'] = Buffer.byteLength(bodyStr);
    }

    const req = https.request(options, (res) => {
      let data = '';

      res.on('data', (chunk) => {
        data += chunk;
      });

      res.on('end', () => {
        try {
          const parsed = JSON.parse(data);
          resolve({
            status: res.statusCode,
            headers: res.headers,
            body: parsed,
          });
        } catch (err) {
          reject(new Error(`Invalid JSON response: ${data}`));
        }
      });
    });

    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Request timeout (30s)'));
    });

    req.on('error', (error) => {
      reject(new Error(`Network error: ${error.message}`));
    });

    if (body) {
      req.write(JSON.stringify(body));
    }

    req.end();
  });
}

/**
 * Publish a transaction
 */
async function publishTransaction(dataHex) {
  if (!dataHex) {
    throw new Error('Data is required');
  }

  if (!isValidHex(dataHex)) {
    throw new Error('Data must be valid hexadecimal');
  }

  if (dataHex.length > 1000000) {
    throw new Error('Data too large (max 500KB)');
  }

  const startTime = Date.now();

  try {
    const path = WAIT_PARAM ? '/publish?wait=true' : '/publish';
    
    const response = await makeRequest('POST', path, {
      data: dataHex,
    });

    const latency = Date.now() - startTime;

    if (response.status !== 201) {
      throw new Error(
        `HTTP ${response.status}: ${response.body?.error || 'Unknown error'}`
      );
    }

    if (!response.body?.success) {
      throw new Error('Request not marked as successful');
    }

    return {
      success: true,
      txid: response.body.txid || null,
      uuid: response.body.uuid || null,
      arcStatus: response.body.arc_status || null,
      latency: latency,
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    return {
      success: false,
      error: error.message,
      latency: Date.now() - startTime,
      timestamp: new Date().toISOString(),
    };
  }
}

/**
 * Format output for console
 */
function formatOutput(result) {
  if (result.success) {
    console.log('\n✅ Transaction Published Successfully\n');
    console.log(`  TXID:         ${result.txid || 'pending'}`);
    console.log(`  UUID:         ${result.uuid || 'N/A'}`);
    console.log(`  ARC Status:   ${result.arcStatus || 'N/A'}`);
    console.log(`  Latency:      ${result.latency}ms`);
    console.log(`  Timestamp:    ${result.timestamp}`);
    console.log('');
    return 0;
  } else {
    console.error('\n❌ Transaction Failed\n');
    console.error(`  Error:        ${result.error}`);
    console.error(`  Latency:      ${result.latency}ms`);
    console.error(`  Timestamp:    ${result.timestamp}`);
    console.error('');
    return 1;
  }
}

/**
 * Main execution
 */
async function main() {
  // Validate API key
  if (!API_KEY) {
    console.error('❌ Error: GOVHASH_API_KEY not set');
    console.error('   Set via: export GOVHASH_API_KEY="your_key"');
    console.error('   Or in .env file');
    process.exit(1);
  }

  // Parse arguments
  let data = null;
  
  if (process.argv.length < 3) {
    console.error('❌ Error: No data provided');
    console.error('\nUsage:');
    console.error('  node basic-publish.js "Your message"');
    console.error('  node basic-publish.js --data="48656c6c6f"');
    process.exit(1);
  }

  const arg = process.argv[2];
  
  if (arg.startsWith('--data=')) {
    data = arg.substring(7);
  } else {
    data = textToHex(arg);
  }

  // Publish
  console.log('\n📡 Publishing transaction to GovHash...\n');
  const result = await publishTransaction(data);
  
  // Output and exit
  const exitCode = formatOutput(result);
  process.exit(exitCode);
}

// Run
if (require.main === module) {
  main().catch((error) => {
    console.error('Fatal error:', error.message);
    process.exit(1);
  });
}

module.exports = { publishTransaction, textToHex };
