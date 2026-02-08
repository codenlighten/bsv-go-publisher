#!/usr/bin/env node

/**
 * GovHash Health Monitor Script
 * 
 * Continuous monitoring of GovHash API health, performance, and queue depth.
 * Useful for production deployments to detect issues early.
 * 
 * Usage:
 *   node health-monitor.js --interval=30 --output=health.log
 *   node health-monitor.js --check-once --quiet
 * 
 * Options:
 *   --interval N      Check every N seconds (default: 60)
 *   --duration M      Monitor for M minutes (default: infinite)
 *   --check-once      Run single check and exit
 *   --output file     Log results to file
 *   --quiet           Minimal output (errors only)
 *   --alert-latency N Alert if latency exceeds N ms (default: 15000)
 *   --alert-queue N   Alert if queue depth exceeds N (default: 500)
 * 
 * Environment:
 *   GOVHASH_API_KEY - Your API key (required)
 *   GOVHASH_API_URL - API endpoint (optional)
 *   SLACK_WEBHOOK    - Slack webhook for alerts (optional)
 * 
 * Exit codes:
 *   0 - Healthy
 *   1 - Unhealthy
 *   2 - Configuration error
 */

const https = require('https');
const http = require('http');
const fs = require('fs');
const { URL } = require('url');
require('dotenv').config();

// Configuration
const API_KEY = process.env.GOVHASH_API_KEY || '';
const API_URL = process.env.GOVHASH_API_URL || 'https://api.govhash.org';
const SLACK_WEBHOOK = process.env.SLACK_WEBHOOK || '';

// Constants
const REQUEST_TIMEOUT = 10000;

class HealthMonitor {
  constructor(options = {}) {
    this.interval = (options.interval || 60) * 1000;
    this.duration = options.duration ? (options.duration * 60 * 1000) : null;
    this.checkOnce = options.checkOnce || false;
    this.outputFile = options.output || null;
    this.quiet = options.quiet || false;
    this.alertLatency = options.alertLatency || 15000;
    this.alertQueue = options.alertQueue || 500;
    
    this.checks = [];
    this.startTime = Date.now();
    this.isHealthy = true;
    this.lastAlertTime = {};
  }

  /**
   * Make HTTPS request with timeout
   */
  makeRequest(method, path, body = null) {
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
          'User-Agent': 'GovHash-AKUA-Monitor/1.0',
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

      if (body) {
        req.write(JSON.stringify(body));
      }

      req.end();
    });
  }

  /**
   * Check API health
   */
  async checkHealth() {
    const check = {
      timestamp: new Date().toISOString(),
      healthy: true,
      issues: [],
      metrics: {},
    };

    // Test 1: Basic connectivity
    const connTest = await this.makeRequest('POST', '/publish?wait=true', {
      data: '7465737420646174610a', // "test data\n" in hex
    });

    if (connTest.status !== 201) {
      check.healthy = false;
      check.issues.push(`API returned status ${connTest.status}`);
    }

    if (connTest.error) {
      check.healthy = false;
      check.issues.push(`Connection error: ${connTest.error}`);
    }

    if (connTest.body?.success) {
      check.metrics.txid = connTest.body.txid || 'pending';
      check.metrics.arcStatus = connTest.body.arc_status || 'unknown';
      check.metrics.latency = connTest.latency || 0;

      // Check latency threshold
      if (check.metrics.latency > this.alertLatency) {
        check.issues.push(`Latency ${check.metrics.latency}ms exceeds threshold ${this.alertLatency}ms`);
      }
    }

    // Test 2: Get admin stats
    const statsReq = await this.makeRequest('GET', '/admin/api/stats');
    if (statsReq.status === 200 && statsReq.body) {
      check.metrics.utxos = statsReq.body.utxos || 0;
      check.metrics.queueDepth = statsReq.body.queue_depth || 0;
      check.metrics.broadcasts24h = statsReq.body.broadcasts_24h || 0;

      // Check queue depth threshold
      if (check.metrics.queueDepth > this.alertQueue) {
        check.issues.push(`Queue depth ${check.metrics.queueDepth} exceeds threshold ${this.alertQueue}`);
      }

      // Check UTXO availability
      if (check.metrics.utxos < 100) {
        check.issues.push(`Low UTXO pool: ${check.metrics.utxos}`);
      }
    } else {
      check.issues.push('Could not retrieve admin stats');
    }

    check.healthy = check.issues.length === 0;
    return check;
  }

  /**
   * Send Slack alert
   */
  async sendSlackAlert(check) {
    if (!SLACK_WEBHOOK) return;

    const hourAgo = Date.now() - 3600000;
    const lastAlert = this.lastAlertTime['slack'] || 0;

    // Rate limit alerts to 1 per hour
    if (lastAlert > hourAgo) return;

    const issueText = check.issues.map(i => `• ${i}`).join('\n');
    const payload = {
      text: '⚠️ GovHash API Health Alert',
      attachments: [{
        color: check.healthy ? 'good' : 'danger',
        fields: [
          {
            title: 'Status',
            value: check.healthy ? '✅ Healthy' : '❌ Unhealthy',
            short: true,
          },
          {
            title: 'Timestamp',
            value: check.timestamp,
            short: true,
          },
          {
            title: 'Issues',
            value: issueText || 'None',
          },
          {
            title: 'Queue Depth',
            value: check.metrics.queueDepth || 'N/A',
            short: true,
          },
          {
            title: 'Latency',
            value: `${check.metrics.latency || 0}ms`,
            short: true,
          },
        ],
      }],
    };

    try {
      await new Promise((resolve, reject) => {
        const url = new URL(SLACK_WEBHOOK);
        const options = {
          hostname: url.hostname,
          path: url.pathname + url.search,
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          timeout: 5000,
        };

        const req = https.request(options, (res) => {
          resolve();
        });

        req.on('error', reject);
        req.write(JSON.stringify(payload));
        req.end();
      });

      this.lastAlertTime['slack'] = Date.now();
    } catch (err) {
      // Silently ignore Slack errors
    }
  }

  /**
   * Log output
   */
  log(message) {
    const timestamp = new Date().toISOString();
    const logLine = `[${timestamp}] ${message}`;

    if (!this.quiet) {
      console.log(logLine);
    }

    if (this.outputFile) {
      fs.appendFileSync(this.outputFile, logLine + '\n');
    }
  }

  /**
   * Format check for display
   */
  formatCheck(check) {
    const status = check.healthy ? '✅' : '❌';
    const metrics = [
      `Latency: ${check.metrics.latency || 'N/A'}ms`,
      `Queue: ${check.metrics.queueDepth || 'N/A'}`,
      `UTXOs: ${check.metrics.utxos || 'N/A'}`,
    ].join(' | ');

    let output = `${status} ${metrics}`;

    if (check.issues.length > 0) {
      output += ` | Issues: ${check.issues.join('; ')}`;
    }

    return output;
  }

  /**
   * Run monitoring loop
   */
  async run() {
    if (!this.quiet) {
      console.log('\n🏥 GovHash Health Monitor Started');
      console.log(`   Interval: ${this.interval / 1000}s`);
      console.log(`   Latency Alert: ${this.alertLatency}ms`);
      console.log(`   Queue Alert: ${this.alertQueue}\n`);
    }

    while (true) {
      const check = await this.checkHealth();
      this.checks.push(check);
      this.isHealthy = check.healthy;

      this.log(this.formatCheck(check));

      if (!check.healthy) {
        await this.sendSlackAlert(check);
      }

      // Check if we should stop
      if (this.checkOnce) {
        break;
      }

      if (this.duration && Date.now() - this.startTime > this.duration) {
        break;
      }

      // Wait for next interval
      await new Promise(resolve => setTimeout(resolve, this.interval));
    }

    this.printSummary();
  }

  /**
   * Print summary statistics
   */
  printSummary() {
    if (this.checks.length === 0) return;

    const healthy = this.checks.filter(c => c.healthy).length;
    const uptime = (healthy / this.checks.length * 100).toFixed(1);

    const latencies = this.checks
      .filter(c => c.metrics.latency)
      .map(c => c.metrics.latency)
      .sort((a, b) => a - b);

    if (!this.quiet && this.checks.length > 1) {
      console.log('\n📊 Summary:');
      console.log(`  Checks: ${this.checks.length}`);
      console.log(`  Uptime: ${uptime}%`);

      if (latencies.length > 0) {
        const avg = (latencies.reduce((a, b) => a + b, 0) / latencies.length).toFixed(0);
        const p95 = latencies[Math.floor(latencies.length * 0.95)];
        console.log(`  Avg Latency: ${avg}ms (P95: ${p95}ms)`);
      }
    }

    this.log(`Monitor stopped. Uptime: ${uptime}%`);
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
    process.exit(2);
  }

  const argv = parseArgs();
  const monitor = new HealthMonitor({
    interval: argv.interval,
    duration: argv.duration,
    checkOnce: argv['check-once'],
    output: argv.output,
    quiet: argv.quiet,
    alertLatency: argv['alert-latency'],
    alertQueue: argv['alert-queue'],
  });

  try {
    await monitor.run();
    process.exit(monitor.isHealthy ? 0 : 1);
  } catch (error) {
    console.error('Fatal error:', error.message);
    process.exit(2);
  }
}

if (require.main === module) {
  main();
}

module.exports = { HealthMonitor };
