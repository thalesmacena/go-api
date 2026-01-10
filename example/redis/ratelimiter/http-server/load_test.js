import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const acceptedRate = new Rate('accepted_requests');
const rejectedRate = new Rate('rejected_requests');
const responseTime = new Trend('response_time');

// Global test start time (initialized on first access)
let testStartTime = Date.now();

// Test configuration
// 10 users running for 3 minutes
// Total requests per user: 500 (100 + 150 + 250)
export const options = {
  vus: 10,        // 10 virtual users from the start (no warm-up)
  duration: '3m', // Run for 3 minutes
  thresholds: {
    http_req_duration: ['p(95)<500'],
    accepted_requests: ['rate>0'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// User IDs for the 10 users
const USER_IDS = [
  'user_001',
  'user_002',
  'user_003',
  'user_004',
  'user_005',
  'user_006',
  'user_007',
  'user_008',
  'user_009',
  'user_010',
];

// Counters per user for accepted/rejected (total)
const userAcceptedCounters = {};
const userRejectedCounters = {};
const userTotalCounters = {};

// Counters per user per phase (minute)
const phase1AcceptedCounters = {}; // First minute (0-60s)
const phase1RejectedCounters = {};
const phase2AcceptedCounters = {}; // Second minute (60-120s)
const phase2RejectedCounters = {};
const phase3AcceptedCounters = {}; // Third minute (120-180s)
const phase3RejectedCounters = {};

USER_IDS.forEach(userId => {
  userAcceptedCounters[userId] = new Counter(`accepted_${userId}`);
  userRejectedCounters[userId] = new Counter(`rejected_${userId}`);
  userTotalCounters[userId] = new Counter(`total_${userId}`);
  
  // Phase counters
  phase1AcceptedCounters[userId] = new Counter(`phase1_accepted_${userId}`);
  phase1RejectedCounters[userId] = new Counter(`phase1_rejected_${userId}`);
  phase2AcceptedCounters[userId] = new Counter(`phase2_accepted_${userId}`);
  phase2RejectedCounters[userId] = new Counter(`phase2_rejected_${userId}`);
  phase3AcceptedCounters[userId] = new Counter(`phase3_accepted_${userId}`);
  phase3RejectedCounters[userId] = new Counter(`phase3_rejected_${userId}`);
});

export default function () {
  // Get user ID based on VU (Virtual User) number
  // Each VU corresponds to one user
  const userId = USER_IDS[(__VU - 1) % USER_IDS.length];
  
  // Calculate elapsed time since test start (in seconds)
  const elapsedSeconds = (Date.now() - testStartTime) / 1000;
  
  // Determine which phase (minute) we're in based on elapsed time
  // Phase 1: 0-60 seconds (100 requests)
  // Phase 2: 60-120 seconds (150 requests)
  // Phase 3: 120-180 seconds (250 requests)
  let phase = 0;
  let sleepTime = 0.6;
  
  if (elapsedSeconds < 60) {
    // First minute: 100 requests in 60 seconds = ~0.6s per request
    phase = 1;
    sleepTime = 0.6;
  } else if (elapsedSeconds < 120) {
    // Second minute: 150 requests in 60 seconds = ~0.4s per request
    phase = 2;
    sleepTime = 0.4;
  } else if (elapsedSeconds < 180) {
    // Third minute: 250 requests in 60 seconds = ~0.24s per request
    phase = 3;
    sleepTime = 0.24;
  } else {
    // Test finished
    return;
  }
  
  const url = `${BASE_URL}/api/rate-limit/${userId}`;
  const startTime = Date.now();
  
  const response = http.get(url, {
    tags: { userId: userId },
  });
  
  const responseTimeMs = Date.now() - startTime;
  responseTime.add(responseTimeMs);
  
  // Always count as total request
  userTotalCounters[userId].add(1);
  
  // Check if request failed (connection error, etc.)
  if (response.status === 0 || !response.body) {
    // Connection failed - don't count as accepted/rejected
    sleep(sleepTime);
    return;
  }
  
  // Parse response
  let responseBody;
  try {
    responseBody = JSON.parse(response.body);
  } catch (e) {
    // Failed to parse - don't count as accepted/rejected
    sleep(sleepTime);
    return;
  }
  
  // Check if request was accepted or rejected
  const isAccepted = response.status === 200 && responseBody && responseBody.status === 'accepted';
  const isRejected = response.status === 429 || (responseBody && responseBody.status === 'rejected');
  
  if (isAccepted) {
    acceptedRate.add(1);
    userAcceptedCounters[userId].add(1);
    
    // Track by phase
    if (phase === 1) {
      phase1AcceptedCounters[userId].add(1);
    } else if (phase === 2) {
      phase2AcceptedCounters[userId].add(1);
    } else if (phase === 3) {
      phase3AcceptedCounters[userId].add(1);
    }
  } else if (isRejected) {
    rejectedRate.add(1);
    userRejectedCounters[userId].add(1);
    
    // Track by phase
    if (phase === 1) {
      phase1RejectedCounters[userId].add(1);
    } else if (phase === 2) {
      phase2RejectedCounters[userId].add(1);
    } else if (phase === 3) {
      phase3RejectedCounters[userId].add(1);
    }
  }
  
  // Verify response structure
  check(response, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    'response has userId': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.userId === userId;
      } catch {
        return false;
      }
    },
    'response has status field': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status === 'accepted' || body.status === 'rejected';
      } catch {
        return false;
      }
    },
  });
  
  sleep(sleepTime);
}

export function handleSummary(data) {
  let summary = '\n=== LOAD TEST SUMMARY ===\n\n';
  
  // Get total requests
  const totalRequests = data.metrics.http_reqs?.values?.count || 0;
  
  // Get accepted/rejected counts from metrics
  const acceptedCount = data.metrics.accepted_requests?.values?.count || 0;
  const rejectedCount = data.metrics.rejected_requests?.values?.count || 0;
  const acceptedRateValue = data.metrics.accepted_requests?.values?.rate || 0;
  const rejectedRateValue = data.metrics.rejected_requests?.values?.rate || 0;
  
  // Get latency metrics
  const httpReqDuration = data.metrics.http_req_duration?.values || {};
  const avgLatency = httpReqDuration.avg || 0;
  const minLatency = httpReqDuration.min || 0;
  const maxLatency = httpReqDuration.max || 0;
  const p50 = httpReqDuration.med || 0;
  const p90 = httpReqDuration['p(90)'] || 0;
  const p95 = httpReqDuration['p(95)'] || 0;
  const p99 = httpReqDuration['p(99)'] || 0;
  
  summary += `Total requests: ${totalRequests}\n`;
  summary += `Accepted requests: ${acceptedCount}\n`;
  summary += `Rejected requests: ${rejectedCount}\n`;
  summary += `Acceptance rate: ${(acceptedRateValue * 100).toFixed(2)}%\n`;
  summary += `Rejection rate: ${(rejectedRateValue * 100).toFixed(2)}%\n\n`;
  
  summary += '=== LATENCY METRICS ===\n';
  summary += `Average: ${formatDuration(avgLatency)}\n`;
  summary += `Min: ${formatDuration(minLatency)}\n`;
  summary += `Max: ${formatDuration(maxLatency)}\n`;
  summary += `p50 (median): ${formatDuration(p50)}\n`;
  summary += `p90: ${formatDuration(p90)}\n`;
  summary += `p95: ${formatDuration(p95)}\n`;
  summary += `p99: ${formatDuration(p99)}\n\n`;
  
  summary += '=== PER-USER STATISTICS (TOTAL) ===\n';
  summary += 'User ID        | Accepted | Rejected | Total    | Acceptance Rate\n';
  summary += '---------------|----------|----------|----------|----------------\n';
  
  // Get per-user statistics from metrics
  USER_IDS.forEach(userId => {
    const acceptedKey = `accepted_${userId}`;
    const rejectedKey = `rejected_${userId}`;
    const totalKey = `total_${userId}`;
    
    const accepted = data.metrics[acceptedKey]?.values?.count || 0;
    const rejected = data.metrics[rejectedKey]?.values?.count || 0;
    const total = data.metrics[totalKey]?.values?.count || 0;
    const acceptanceRate = total > 0 ? ((accepted / total) * 100).toFixed(2) : '0.00';
    
    summary += `${userId.padEnd(14)} | ${String(accepted).padStart(8)} | ${String(rejected).padStart(8)} | ${String(total).padStart(8)} | ${acceptanceRate.padStart(14)}%\n`;
  });
  
  summary += '\n=== PER-USER STATISTICS BY PHASE ===\n';
  summary += 'Phase 1 (0-60s): 100 requests expected (all accepted)\n';
  summary += 'Phase 2 (60-120s): 150 requests expected (100 accepted, 50 rejected)\n';
  summary += 'Phase 3 (120-180s): 250 requests expected (100 accepted, 150 rejected)\n\n';
  
  USER_IDS.forEach(userId => {
    // Phase 1 stats
    const phase1Accepted = data.metrics[`phase1_accepted_${userId}`]?.values?.count || 0;
    const phase1Rejected = data.metrics[`phase1_rejected_${userId}`]?.values?.count || 0;
    const phase1Total = phase1Accepted + phase1Rejected;
    
    // Phase 2 stats
    const phase2Accepted = data.metrics[`phase2_accepted_${userId}`]?.values?.count || 0;
    const phase2Rejected = data.metrics[`phase2_rejected_${userId}`]?.values?.count || 0;
    const phase2Total = phase2Accepted + phase2Rejected;
    
    // Phase 3 stats
    const phase3Accepted = data.metrics[`phase3_accepted_${userId}`]?.values?.count || 0;
    const phase3Rejected = data.metrics[`phase3_rejected_${userId}`]?.values?.count || 0;
    const phase3Total = phase3Accepted + phase3Rejected;
    
    summary += `${userId}:\n`;
    summary += `  Phase 1: ${phase1Accepted} accepted, ${phase1Rejected} rejected, ${phase1Total} total\n`;
    summary += `  Phase 2: ${phase2Accepted} accepted, ${phase2Rejected} rejected, ${phase2Total} total\n`;
    summary += `  Phase 3: ${phase3Accepted} accepted, ${phase3Rejected} rejected, ${phase3Total} total\n`;
  });
  
  summary += '\n=== EXPECTED BEHAVIOR ===\n';
  summary += 'Each user should have a rate limit of 100 requests per minute.\n';
  summary += 'Since each user sends 500 requests over 3 minutes:\n';
  summary += '  - First minute: 100 requests (all should be accepted)\n';
  summary += '  - Second minute: 150 requests (100 accepted, 50 rejected)\n';
  summary += '  - Third minute: 250 requests (100 accepted, 150 rejected)\n';
  summary += 'Expected total per user: 300 accepted, 200 rejected\n\n';
  
  return {
    stdout: summary,
  };
}

// Helper function to format duration in milliseconds
function formatDuration(ms) {
  if (ms === 0 || !ms) return '0ms';
  if (ms < 1) return `${(ms * 1000).toFixed(2)}µs`;
  if (ms < 1000) return `${ms.toFixed(2)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}
