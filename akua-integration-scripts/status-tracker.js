#!/usr/bin/env node

/**
 * GovHash Status Tracker Script
 * 
 * Track transaction status using UUID when queue is full (>1000 pending).
 * Automatically polls API until transaction gets confirmed TXID.
 * 
 * Usage:
 *   node status-tracker.js --uuid=<uuid> --wait
 *   node status-tracker.js --input=uuids.txt --output=status.csv
 * 
 * Options:
 *   --uuid U          Track single UUID
 *   --input file      Track UUIDs from file (one per line)
 *   --output file     Save results to CSV
 *   --wait            Wait until all have TXIDs (vs one-time check)
 *   --interval N      Poll interval in seconds (default: 5)
 *   --timeout M       Give up after M seconds (default: 300)
 * 
 * Environment:
 *   GOVHASH_API_KEY - Your API key (required)
 *   GOVHASH_API_URL - API endpoint (optional)
 * 
 * Output:
 *   CSV with columns: uuid,txid,status,checkTime,confirmed
 */

const https = require('https');
const http = require('http');
const fs = require('fs');
const readline = require('readline');
const { URL } = require('url');
require('dotenv').config();

// Configuration
const API_KEY = process.env.GOVHASH_API_KEY || '';
const API_URL = process.env.GOVHASH_API_URL || 'https://api.govhash.org';

// Constants
const REQUEST_TIMEOUT = 10000;

class StatusTracker {
  constructor(options = {}) {
    this.interval = (options.interval || 5) * 1000;
    this.timeout = (options.timeout || 300) * 1000;
    this.wait = options.wait || false;
    this.outputFile = options.output || null;
    this.uuids = options.uuids || [];
    
    this.results = {};
    this.startTime = Date.now();
  }

  /**
   * Make HTTPS request
   */
  makeRequest(method, path) {
    return new Promise((resolve) => {
      const url = new URL(path, API_URL);
      const isHttps = url.protocol === 'https:';
      const client = isHttps ? https : http;

      const options = {
        hostname: url.hostname,
        port: url.port,
        path: url.pathname + url.search,
        method: method,
        headers: {
          'X-API-Key': API_KEY,
          'Content-Type': 'application/json',
          'User-Agent': 'GovHash-AKUA-Tracker/1.0',
        },
        timeout: REQUEST_TIMEOUT,
      };

      const req = client.request(options, (res) => {
        let data = '';

        res.on('data', (chunk) => {
          data += chunk;
        });

        res.on('end', () => {
          try {
            resolve({
              status: res.statusCode,
              body: JSON.parse(data),
            });
          } catch (err) {
            resolve({
              status: res.statusCode,
              body: null,
              error: 'Invalid JSON',
            });
          }
        });
      });

      req.on('timeout', () => {
        req.destroy();
        resolve({
          status: 0,
          error: 'Request timeout',
        });
      });

      req.on('error', (error) => {
        resolve({
          status: 0,
          error: `Network error: ${error.message}`,
        });
      });

      req.end();
    });
  }

  /**
   * Check single UUID status
   */
  async checkStatus(uuid) {
    const response = await this.makeRequest('GET', `/status/${uuid}`);

    if (response.status === 200 && response.body) {
      return {
        uuid,
        txid: response.body.txid || null,
        status: response.body.status || 'unknown',
        confirmed: !!response.body.txid,
        checkTime: new Date().toISOString(),
        response: response.body,
      };
    }

    return {
      uuid,
      txid: null,
      status: response.error || `HTTP ${response.status}`,
      confirmed: false,
      checkTime: new Date().toISOString(),
      error: response.error,
    };
  }

  /**
   * Load UUIDs from file
   */
  async loadUuidsFromFile(filePath) {
    return new Promise((resolve, reject) => {
      const uuids = [];
      const rl = readline.createInterface({
        input: fs.createReadStream(filePath),
        crlfDelay: Infinity,
      });

      rl.on('line', (line) => {
        const uuid = line.trim();
        if (uuid && uuid.match(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i)) {
          uuids.push(uuid);
        }
      });

      rl.on('close', () => {
        resolve(uuids);
      });

      rl.on('error', reject);
    });
  }

  /**
   * Print progress
   */
  printProgress() {
    const confirmed = Object.values(this.results).filter(r => r.confirmed).length;
    const total = Object.keys(this.results).length;
    const percent = Math.round((confirmed / total) * 100);

    process.stdout.write(`\r📊 Status: ${confirmed}/${total} confirmed (${percent}%)`);
  }

  /**
   * Track UUIDs
   */
  async track() {
    console.log(`\n📍 Tracking ${this.uuids.length} UUID(s)...\n`);

    // Initial checks
    for (const uuid of this.uuids) {
      const status = await this.checkStatus(uuid);
      this.results[uuid] = status;
    }

    this.printProgress();

    // Wait for confirmations if requested
    if (this.wait) {
      const allConfirmed = () => Object.values(this.results).every(r => r.confirmed);
      const timedOut = () => Date.now() - this.startTime > this.timeout;

      while (!allConfirmed() && !timedOut()) {
        await new Promise(resolve => setTimeout(resolve, this.interval));

        for (const uuid of this.uuids) {
          if (!this.results[uuid].confirmed) {
            const status = await this.checkStatus(uuid);
            this.results[uuid] = status;
          }
        }

        this.printProgress();
      }

      console.log('\n');

      if (timedOut()) {
        console.log('⏱️ Timeout reached');
      } else {
        console.log('✅ All confirmed');
      }
    }

    this.printResults();
  }

  /**
   * Print results table
   */
  printResults() {
    console.log('\n📋 Results:\n');
    console.log('UUID                                 | TXID                                  | Status');
    console.log('-'.repeat(100));

    for (const [uuid, result] of Object.entries(this.results)) {
      const txid = result.txid ? result.txid.substring(0, 36) : 'PENDING';
      const status = result.confirmed ? '✅' : '⏳';
      console.log(`${uuid} | ${txid} | ${status}`);
    }

    console.log('');
  }

  /**
   * Save results to CSV
   */
  saveResults() {
    if (!this.outputFile) return;

    const csv = [
      'uuid,txid,status,checkTime,confirmed',
      ...Object.values(this.results).map(r => [
        r.uuid,
        r.txid || '',
        r.status || '',
        r.checkTime,
        r.confirmed ? 'yes' : 'no',
      ].join(',')),
    ].join('\n');

    fs.writeFileSync(this.outputFile, csv);
    console.log(`💾 Results saved to: ${this.outputFile}`);
  }
}

/**
 * Validate UUID format
 */
function isValidUuid(uuid) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(uuid);
}

/**
 * Parse command line arguments
 */
function parseArgs() {
  const args = {};
  process.argv.slice(2).forEach(arg => {
    if (arg.startsWith('--')) {
      const [key, value] = arg.substring(2).split('=');
      args[key] = value === undefined ? true : (isNaN(value) ? value : parseInt(value));
    }
  });
  return args;
}

/**
 * Main execution
 */
async function main() {
  if (!API_KEY) {
    console.error('❌ Error: GOVHASH_API_KEY not set');
    process.exit(1);
  }

  const argv = parseArgs();
  let uuids = [];

  // Get UUIDs from arguments or file
  if (argv.uuid) {
    if (!isValidUuid(argv.uuid)) {
      console.error('❌ Invalid UUID format');
      process.exit(1);
    }
    uuids = [argv.uuid];
  } else if (argv.input) {
    try {
      uuids = await new StatusTracker().loadUuidsFromFile(argv.input);
    } catch (error) {
      console.error(`❌ Failed to load file: ${error.message}`);
      process.exit(1);
    }
  } else {
    console.error('Usage: node status-tracker.js --uuid=<uuid> [--wait]');
    console.error('   or: node status-tracker.js --input=file.txt [--output=results.csv]');
    process.exit(1);
  }

  if (uuids.length === 0) {
    console.error('❌ No valid UUIDs provided');
    process.exit(1);
  }

  const tracker = new StatusTracker({
    interval: argv.interval,
    timeout: argv.timeout,
    wait: argv.wait,
    output: argv.output,
    uuids,
  });

  try {
    await tracker.track();
    tracker.saveResults();

    const allConfirmed = Object.values(tracker.results).every(r => r.confirmed);
    process.exit(allConfirmed ? 0 : 1);
  } catch (error) {
    console.error('Fatal error:', error.message);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = { StatusTracker };
