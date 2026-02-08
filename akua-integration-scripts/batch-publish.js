#!/usr/bin/env node

/**
 * GovHash Batch Publish Script
 * 
 * Production-ready script for publishing multiple transactions while
 * respecting the train batching architecture (1,000 tx per 3-second cycle).
 * 
 * Implements intelligent queuing:
 * - Groups transactions into batches
 * - Respects 3-second train cycle timing
 * - Handles queue overflow gracefully
 * - Tracks all TXIDs and UUIDs
 * 
 * Usage:
 *   node batch-publish.js file.csv --workers=10
 *   node batch-publish.js --input=data.json --batch-size=100
 * 
 * CSV Format:
 *   id,data,priority
 *   1,48656c6c6f,high
 *   2,476f7648617368,normal
 * 
 * Environment:
 *   GOVHASH_API_KEY - Your API key (required)
 *   GOVHASH_API_URL - API endpoint (optional)
 * 
 * Output:
 *   results.csv - Transaction results with TXID/UUID/status
 */

const https = require('https');
const fs = require('fs');
const path = require('path');
const { URL } = require('url');
const readline = require('readline');
require('dotenv').config();

// Configuration
const API_KEY = process.env.GOVHASH_API_KEY || '';
const API_URL = process.env.GOVHASH_API_URL || 'https://api.govhash.org';

// Constants
const TRAIN_CYCLE_MS = 3000; // 3-second train cycle
const MAX_BATCH_SIZE = 1000; // Max per train
const REQUEST_TIMEOUT = 30000;
const DEFAULT_WORKERS = 5;
const DEFAULT_BATCH_SIZE = 50;

class BatchPublisher {
  constructor(options = {}) {
    this.workers = options.workers || DEFAULT_WORKERS;
    this.batchSize = options.batchSize || DEFAULT_BATCH_SIZE;
    this.apiKey = API_KEY;
    this.apiUrl = API_URL;
    this.queue = [];
    this.results = [];
    this.activeRequests = 0;
    this.totalSent = 0;
    this.stats = {
      success: 0,
      failed: 0,
      queued: 0,
      startTime: null,
      endTime: null,
    };
  }

  /**
   * Make HTTPS request
   */
  makeRequest(method, path, body = null) {
    return new Promise((resolve, reject) => {
      const url = new URL(path, this.apiUrl);
      
      const options = {
        hostname: url.hostname,
        port: url.port || 443,
        path: url.pathname + url.search,
        method: method,
        headers: {
          'X-API-Key': this.apiKey,
          'Content-Type': 'application/json',
          'User-Agent': 'GovHash-AKUA-Batch/1.0',
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
              body: parsed,
            });
          } catch (err) {
            reject(new Error(`Invalid JSON: ${data.substring(0, 100)}`));
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
   * Publish a single transaction
   */
  async publishOne(item) {
    const startTime = Date.now();
    
    try {
      const response = await this.makeRequest('POST', '/publish?wait=true', {
        data: item.data,
      });

      const latency = Date.now() - startTime;

      if (response.status === 201 && response.body?.success) {
        this.stats.success++;
        return {
          ...item,
          txid: response.body.txid || null,
          uuid: response.body.uuid || null,
          arcStatus: response.body.arc_status || 'UNKNOWN',
          status: 'SUCCESS',
          latency: latency,
          error: null,
          timestamp: new Date().toISOString(),
        };
      } else {
        this.stats.failed++;
        return {
          ...item,
          status: 'FAILED',
          latency: latency,
          error: response.body?.error || `HTTP ${response.status}`,
          timestamp: new Date().toISOString(),
        };
      }
    } catch (error) {
      this.stats.failed++;
      return {
        ...item,
        status: 'ERROR',
        latency: Date.now() - startTime,
        error: error.message,
        timestamp: new Date().toISOString(),
      };
    }
  }

  /**
   * Worker process - consume from queue
   */
  async worker() {
    while (this.queue.length > 0) {
      const batch = this.queue.splice(0, Math.min(this.workers, this.queue.length));
      
      if (batch.length === 0) break;

      this.activeRequests += batch.length;

      // Process batch in parallel
      const promises = batch.map(item => this.publishOne(item));
      const batchResults = await Promise.all(promises);
      
      this.results.push(...batchResults);
      this.activeRequests -= batch.length;

      // Print progress
      this.printProgress();

      // Wait for next train cycle if queue still has items
      if (this.queue.length > 0) {
        await this.sleep(TRAIN_CYCLE_MS);
      }
    }
  }

  /**
   * Sleep utility
   */
  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * Print progress to console
   */
  printProgress() {
    const processed = this.results.length;
    const total = this.stats.success + this.stats.failed + this.queue.length;
    const percent = Math.round((processed / total) * 100);
    
    process.stdout.write(
      `\r📊 Progress: ${processed}/${total} (${percent}%) | ✅ ${this.stats.success} | ❌ ${this.stats.failed}`
    );
  }

  /**
   * Load CSV file
   */
  async loadCsv(filePath) {
    return new Promise((resolve, reject) => {
      const items = [];
      const rl = readline.createInterface({
        input: fs.createReadStream(filePath),
        crlfDelay: Infinity,
      });

      let isHeader = true;

      rl.on('line', (line) => {
        if (isHeader) {
          isHeader = false;
          return;
        }

        const parts = line.split(',');
        if (parts.length >= 2) {
          items.push({
            id: parts[0]?.trim(),
            data: parts[1]?.trim(),
            priority: parts[2]?.trim() || 'normal',
          });
        }
      });

      rl.on('close', () => {
        resolve(items);
      });

      rl.on('error', reject);
    });
  }

  /**
   * Load JSON file
   */
  loadJson(filePath) {
    const data = fs.readFileSync(filePath, 'utf8');
    const parsed = JSON.parse(data);
    return Array.isArray(parsed) ? parsed : [parsed];
  }

  /**
   * Save results to CSV
   */
  saveResults(outputPath) {
    const csv = [
      'id,status,txid,uuid,arc_status,latency_ms,error,timestamp',
      ...this.results.map(r => [
        r.id,
        r.status,
        r.txid || '',
        r.uuid || '',
        r.arcStatus || '',
        r.latency,
        (r.error || '').replace(/,/g, ';'),
        r.timestamp,
      ].join(',')),
    ].join('\n');

    fs.writeFileSync(outputPath, csv);
    console.log(`\n\n💾 Results saved to: ${outputPath}`);
  }

  /**
   * Publish batch
   */
  async publish(items) {
    console.log(`\n📤 Publishing ${items.length} transactions...\n`);
    
    this.stats.startTime = Date.now();
    this.queue = [...items];
    this.stats.queued = items.length;

    // Start workers
    const workerPromises = Array(this.workers)
      .fill(null)
      .map(() => this.worker());

    await Promise.all(workerPromises);

    this.stats.endTime = Date.now();
    this.printStats();

    return this.results;
  }

  /**
   * Print statistics
   */
  printStats() {
    const duration = this.stats.endTime - this.stats.startTime;
    const throughput = (this.results.length / duration * 1000).toFixed(2);
    const avgLatency = (this.results.reduce((sum, r) => sum + r.latency, 0) / this.results.length).toFixed(0);

    console.log('\n\n📈 Statistics:');
    console.log(`  Total:        ${this.results.length}`);
    console.log(`  Success:      ${this.stats.success} (${(this.stats.success / this.results.length * 100).toFixed(1)}%)`);
    console.log(`  Failed:       ${this.stats.failed} (${(this.stats.failed / this.results.length * 100).toFixed(1)}%)`);
    console.log(`  Duration:     ${(duration / 1000).toFixed(1)}s`);
    console.log(`  Throughput:   ${throughput} tx/s`);
    console.log(`  Avg Latency:  ${avgLatency}ms`);
  }
}

/**
 * Main execution
 */
async function main() {
  // Validate API key
  if (!API_KEY) {
    console.error('❌ Error: GOVHASH_API_KEY not set');
    process.exit(1);
  }

  // Parse arguments
  const args = process.argv.slice(2);
  if (args.length === 0) {
    console.error('Usage: node batch-publish.js <file.csv> [--workers=5]');
    process.exit(1);
  }

  const inputFile = args[0];
  const workersArg = args.find(a => a.startsWith('--workers='));
  const workers = workersArg ? parseInt(workersArg.split('=')[1]) : DEFAULT_WORKERS;

  if (!fs.existsSync(inputFile)) {
    console.error(`❌ File not found: ${inputFile}`);
    process.exit(1);
  }

  // Load data
  let items = [];
  const ext = path.extname(inputFile).toLowerCase();

  try {
    if (ext === '.csv') {
      items = await new BatchPublisher().loadCsv(inputFile);
    } else if (ext === '.json') {
      items = new BatchPublisher().loadJson(inputFile);
    } else {
      throw new Error('Unsupported file format (use .csv or .json)');
    }
  } catch (error) {
    console.error(`❌ Failed to load file: ${error.message}`);
    process.exit(1);
  }

  if (items.length === 0) {
    console.error('❌ No items to publish');
    process.exit(1);
  }

  // Publish
  const publisher = new BatchPublisher({ workers });
  const results = await publisher.publish(items);

  // Save results
  const outputPath = 'batch-results.csv';
  publisher.saveResults(outputPath);

  // Exit with status
  const exitCode = publisher.stats.failed > 0 ? 1 : 0;
  process.exit(exitCode);
}

if (require.main === module) {
  main().catch((error) => {
    console.error('Fatal error:', error.message);
    process.exit(1);
  });
}

module.exports = { BatchPublisher };
