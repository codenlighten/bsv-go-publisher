#!/usr/bin/env node

/**
 * GovHash Stress Test Script
 * 
 * Production-grade stress testing tool for validating GovHash API
 * capacity, latency, and reliability under controlled load.
 * 
 * Usage:
 *   node stress-test.js --requests=100 --concurrency=10
 *   node stress-test.js --duration=60 --rps=50 --analyze
 * 
 * Options:
 *   --requests N      Total requests to send (default: 100)
 *   --concurrency N   Parallel requests (default: 10)
 *   --duration S      Run for S seconds instead of fixed count
 *   --rps N           Target requests per second (default: unlimited)
 *   --size N          Payload size in bytes (default: 256)
 *   --analyze         Generate detailed analysis report
 *   --output file     Save results to CSV file
 * 
 * Environment:
 *   GOVHASH_API_KEY - Your API key (required)
 *   GOVHASH_API_URL - API endpoint (optional)
 * 
 * Output:
 *   stress-test-results.csv - Raw results
 *   stress-test-report.txt - Analysis and statistics
 */

const https = require('https');
const fs = require('fs');
const { URL } = require('url');
require('dotenv').config();

// Configuration
const API_KEY = process.env.GOVHASH_API_KEY || '';
const API_URL = process.env.GOVHASH_API_URL || 'https://api.govhash.org';

// Constants
const REQUEST_TIMEOUT = 30000;

class StressTest {
  constructor(options = {}) {
    this.requests = options.requests || 100;
    this.concurrency = options.concurrency || 10;
    this.duration = options.duration || null;
    this.rps = options.rps || null;
    this.payloadSize = options.size || 256;
    this.analyze = options.analyze || false;
    this.outputFile = options.output || 'stress-test-results.csv';
    
    this.results = [];
    this.sent = 0;
    this.completed = 0;
    this.startTime = null;
    this.endTime = null;
    this.lastPrintTime = 0;
  }

  /**
   * Make HTTPS request
   */
  makeRequest(data) {
    return new Promise((resolve) => {
      const startTime = Date.now();
      const url = new URL('/publish?wait=true', API_URL);

      const options = {
        hostname: url.hostname,
        port: url.port || 443,
        path: url.pathname + url.search,
        method: 'POST',
        headers: {
          'X-API-Key': API_KEY,
          'Content-Type': 'application/json',
          'User-Agent': 'GovHash-AKUA-StressTest/1.0',
        },
        timeout: REQUEST_TIMEOUT,
      };

      const bodyStr = JSON.stringify({ data });
      options.headers['Content-Length'] = Buffer.byteLength(bodyStr);

      let statusCode = 0;
      let responseTime = 0;
      let error = null;
      let txid = null;
      let uuid = null;

      const req = https.request(options, (res) => {
        statusCode = res.statusCode;
        let body = '';

        res.on('data', (chunk) => {
          body += chunk;
        });

        res.on('end', () => {
          responseTime = Date.now() - startTime;
          
          try {
            const parsed = JSON.parse(body);
            if (parsed.success) {
              txid = parsed.txid || null;
              uuid = parsed.uuid || null;
            } else {
              error = parsed.error || 'Request failed';
            }
          } catch (err) {
            error = `Invalid response: ${err.message}`;
          }

          resolve({
            sent: startTime,
            responseTime,
            statusCode,
            txid,
            uuid,
            error,
            success: statusCode === 201 && !error,
          });
        });
      });

      req.on('timeout', () => {
        req.destroy();
        responseTime = Date.now() - startTime;
        resolve({
          sent: startTime,
          responseTime,
          statusCode: 0,
          error: 'Timeout (30s)',
          success: false,
        });
      });

      req.on('error', (err) => {
        responseTime = Date.now() - startTime;
        resolve({
          sent: startTime,
          responseTime,
          statusCode: 0,
          error: `Error: ${err.message}`,
          success: false,
        });
      });

      req.write(bodyStr);
      req.end();
    });
  }

  /**
   * Generate random hex data
   */
  generatePayload(sizeBytes) {
    const hexSize = sizeBytes * 2; // 2 hex chars per byte
    let result = '';
    for (let i = 0; i < hexSize; i++) {
      result += Math.floor(Math.random() * 16).toString(16);
    }
    return result;
  }

  /**
   * Throttle RPS
   */
  async throttle(requestNumber) {
    if (!this.rps) return; // No throttling

    const elapsed = Date.now() - this.startTime;
    const expectedTime = (requestNumber / this.rps) * 1000;
    const sleepTime = Math.max(0, expectedTime - elapsed);

    if (sleepTime > 0) {
      await new Promise(resolve => setTimeout(resolve, sleepTime));
    }
  }

  /**
   * Print progress
   */
  printProgress() {
    const now = Date.now();
    if (now - this.lastPrintTime < 1000) return; // Every 1s

    this.lastPrintTime = now;
    const elapsed = (now - this.startTime) / 1000;
    const completed = this.results.length;
    const rps = (completed / elapsed).toFixed(1);

    let status = `\r📊 Stress Test: ${completed}/${this.sent} | ${rps} tx/s`;
    if (this.duration) {
      const remaining = Math.max(0, this.duration - elapsed);
      status += ` | ${remaining.toFixed(1)}s remaining`;
    } else {
      const remaining = this.requests - this.sent;
      status += ` | ${remaining} remaining`;
    }

    process.stdout.write(status);
  }

  /**
   * Run stress test
   */
  async run() {
    console.log('\n🚀 Starting stress test...\n');
    console.log(`  Concurrency: ${this.concurrency}`);
    console.log(`  Payload:     ${this.payloadSize} bytes`);
    if (this.rps) console.log(`  Target RPS:  ${this.rps}`);
    console.log('');

    this.startTime = Date.now();
    const activeRequests = new Set();
    let requestCount = 0;

    // Limit generator based on duration or request count
    const shouldContinue = () => {
      if (this.duration) {
        return (Date.now() - this.startTime) / 1000 < this.duration;
      } else {
        return requestCount < this.requests;
      }
    };

    while (activeRequests.size > 0 || shouldContinue()) {
      // Fill concurrency slots
      while (activeRequests.size < this.concurrency && shouldContinue()) {
        await this.throttle(requestCount);

        const payload = this.generatePayload(this.payloadSize);
        const promise = this.makeRequest(payload)
          .then(result => {
            this.results.push(result);
            activeRequests.delete(promise);
            this.printProgress();
          });

        activeRequests.add(promise);
        this.sent++;
        requestCount++;
      }

      // Wait for at least one to complete
      if (activeRequests.size > 0) {
        await Promise.race(activeRequests);
      }
    }

    // Wait for remaining
    await Promise.all(activeRequests);
    this.endTime = Date.now();

    this.printProgress();
    console.log('\n');
  }

  /**
   * Calculate statistics
   */
  calculateStats() {
    const successful = this.results.filter(r => r.success);
    const failed = this.results.filter(r => !r.success);
    const latencies = successful.map(r => r.responseTime).sort((a, b) => a - b);

    if (latencies.length === 0) {
      return null;
    }

    const sum = latencies.reduce((a, b) => a + b, 0);
    const avg = sum / latencies.length;
    const min = latencies[0];
    const max = latencies[latencies.length - 1];
    const p50 = latencies[Math.floor(latencies.length * 0.5)];
    const p95 = latencies[Math.floor(latencies.length * 0.95)];
    const p99 = latencies[Math.floor(latencies.length * 0.99)];

    const duration = (this.endTime - this.startTime) / 1000;
    const throughput = this.results.length / duration;

    return {
      total: this.results.length,
      successful: successful.length,
      failed: failed.length,
      successRate: (successful.length / this.results.length * 100).toFixed(1),
      duration,
      throughput: throughput.toFixed(2),
      latency: {
        min,
        max,
        avg: avg.toFixed(0),
        p50,
        p95,
        p99,
      },
    };
  }

  /**
   * Generate report
   */
  generateReport() {
    const stats = this.calculateStats();

    if (!stats) {
      return 'No successful requests to analyze';
    }

    let report = '\n📋 Stress Test Report\n';
    report += '='.repeat(50) + '\n\n';

    report += `Results:\n`;
    report += `  Total:        ${stats.total}\n`;
    report += `  Successful:   ${stats.successful} (${stats.successRate}%)\n`;
    report += `  Failed:       ${stats.failed}\n`;
    report += `  Duration:     ${stats.duration.toFixed(1)}s\n`;
    report += `  Throughput:   ${stats.throughput} tx/s\n\n`;

    report += `Latency (ms):\n`;
    report += `  Min:          ${stats.latency.min}ms\n`;
    report += `  Max:          ${stats.latency.max}ms\n`;
    report += `  Avg:          ${stats.latency.avg}ms\n`;
    report += `  P50:          ${stats.latency.p50}ms\n`;
    report += `  P95:          ${stats.latency.p95}ms\n`;
    report += `  P99:          ${stats.latency.p99}ms\n\n`;

    // HTTP status breakdown
    const statusCounts = {};
    this.results.forEach(r => {
      const code = r.statusCode || 'ERR';
      statusCounts[code] = (statusCounts[code] || 0) + 1;
    });

    report += `Status Codes:\n`;
    Object.entries(statusCounts).forEach(([code, count]) => {
      const pct = ((count / this.results.length) * 100).toFixed(1);
      report += `  ${code}: ${count} (${pct}%)\n`;
    });

    // Recommendations
    report += '\n💡 Recommendations:\n';
    if (stats.successRate < 95) {
      report += `  • Success rate ${stats.successRate}% is below 95% target\n`;
      report += `  • Consider reducing concurrency or payload size\n`;
    }
    if (stats.latency.p95 > 10000) {
      report += `  • P95 latency ${stats.latency.p95}ms is high\n`;
      report += `  • Server may be experiencing load - consider reducing RPS\n`;
    }
    if (stats.throughput > 300) {
      report += `  • Throughput ${stats.throughput} tx/s exceeds train capacity\n`;
      report += `  • System may be queuing excess transactions\n`;
    }

    report += '\n' + '='.repeat(50) + '\n';

    return report;
  }

  /**
   * Save results to CSV
   */
  saveResults() {
    const csv = [
      'sent,responseTime,statusCode,txid,uuid,error,success',
      ...this.results.map(r => [
        r.sent,
        r.responseTime,
        r.statusCode,
        r.txid || '',
        r.uuid || '',
        (r.error || '').replace(/"/g, '""'),
        r.success ? 'yes' : 'no',
      ].join(',')),
    ].join('\n');

    fs.writeFileSync(this.outputFile, csv);
    console.log(`📁 Results saved to: ${this.outputFile}`);
  }

  /**
   * Save analysis report
   */
  saveReport() {
    const report = this.generateReport();
    const reportFile = this.outputFile.replace('.csv', '-report.txt');
    fs.writeFileSync(reportFile, report);
    console.log(`📁 Report saved to: ${reportFile}`);
  }
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
  const test = new StressTest({
    requests: argv.requests,
    concurrency: argv.concurrency,
    duration: argv.duration,
    rps: argv.rps,
    size: argv.size,
    analyze: argv.analyze,
    output: argv.output,
  });

  try {
    await test.run();
    test.saveResults();

    const report = test.generateReport();
    console.log(report);

    if (test.analyze) {
      test.saveReport();
    }

    const stats = test.calculateStats();
    const exitCode = stats && stats.successRate >= 95 ? 0 : 1;
    process.exit(exitCode);
  } catch (error) {
    console.error('Fatal error:', error.message);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = { StressTest };
