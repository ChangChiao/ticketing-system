// k6 load test script for ticketing system
// Run: k6 run --vus 100 --duration 30s loadtest/basic.js
// Full test: k6 run loadtest/basic.js

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import crypto from 'k6/crypto';
import { Rate, Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const EVENT_ID = __ENV.EVENT_ID || 'e0000000-0000-0000-0000-000000000001';
const REQUEST_SIGN_SECRET = __ENV.REQUEST_SIGN_SECRET || '';
const CAPTCHA_TOKEN = __ENV.CAPTCHA_TOKEN || '';

// Custom metrics
const queueJoinDuration = new Trend('queue_join_duration');
const seatAllocDuration = new Trend('seat_allocation_duration');
const queueJoinErrors = new Rate('queue_join_errors');
const allocErrors = new Rate('seat_alloc_errors');

export const options = {
  scenarios: {
    // Scenario 1: Browsing load (unauthenticated)
    browsing: {
      executor: 'constant-vus',
      vus: 50,
      duration: '60s',
      exec: 'browsing',
      tags: { scenario: 'browsing' },
    },
    // Scenario 2: Queue rush (10,000 users joining queue simultaneously)
    queue_rush: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 100 },
        { duration: '20s', target: 1000 },
        { duration: '10s', target: 5000 },
        { duration: '30s', target: 10000 },
        { duration: '10s', target: 0 },
      ],
      exec: 'queueRush',
      tags: { scenario: 'queue_rush' },
    },
    // Scenario 3: Seat allocation under load
    seat_allocation: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 50 },
        { duration: '30s', target: 500 },
        { duration: '20s', target: 500 },
        { duration: '10s', target: 0 },
      ],
      exec: 'seatAllocation',
      startTime: '30s', // Start after queue rush ramps up
      tags: { scenario: 'seat_allocation' },
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000', 'p(99)<5000'],
    http_req_failed: ['rate<0.05'],
    queue_join_duration: ['p(95)<3000'],
    seat_allocation_duration: ['p(95)<5000'],
    queue_join_errors: ['rate<0.10'],
    seat_alloc_errors: ['rate<0.20'], // Higher tolerance due to contention
  },
};

// Helper: register a unique user and return auth token
function registerUser() {
  const uniqueId = `${__VU}_${Date.now()}_${Math.random().toString(36).slice(2)}`;
  const payload = JSON.stringify({
    email: `loadtest_${uniqueId}@test.com`,
    password: 'loadtest123456',
    name: `Load Test User ${uniqueId}`,
  });

  const res = http.post(`${BASE_URL}/api/auth/register`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (res.status === 201) {
    return JSON.parse(res.body).token;
  }
  return null;
}

function protectedHeaders(token, method, path) {
  const timestamp = Date.now().toString();
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
    'X-Device-Fingerprint': `k6-${__VU}`,
  };
  if (CAPTCHA_TOKEN) {
    headers['X-Captcha-Token'] = CAPTCHA_TOKEN;
  }
  if (REQUEST_SIGN_SECRET) {
    headers['X-Request-Timestamp'] = timestamp;
    headers['X-Request-Signature'] = crypto.hmac(
      'sha256',
      REQUEST_SIGN_SECRET,
      method + path + timestamp,
      'hex'
    );
  }
  return {
    headers,
  };
}

function protectedGet(token, path) {
  return http.get(`${BASE_URL}${path}`, protectedHeaders(token, 'GET', path));
}

function protectedPost(token, path, body = null) {
  return http.post(`${BASE_URL}${path}`, body, protectedHeaders(token, 'POST', path));
}

// Scenario 1: Browsing (unauthenticated)
export function browsing() {
  group('Event Browsing', () => {
    // List events
    const eventsRes = http.get(`${BASE_URL}/api/events`);
    check(eventsRes, { 'events 200': (r) => r.status === 200 });

    // Event detail
    const detailRes = http.get(`${BASE_URL}/api/events/${EVENT_ID}`);
    check(detailRes, { 'detail 200': (r) => r.status === 200 });

    // Availability
    const availRes = http.get(`${BASE_URL}/api/events/${EVENT_ID}/availability`);
    check(availRes, { 'availability 200': (r) => r.status === 200 });
  });

  sleep(Math.random() * 2 + 0.5);
}

// Scenario 2: Queue rush (concurrent queue joins)
export function queueRush() {
  const token = registerUser();
  if (!token) {
    queueJoinErrors.add(1);
    return;
  }

  group('Queue Join', () => {
    const start = Date.now();
    const path = `/api/events/${EVENT_ID}/queue/join`;
    const res = protectedPost(token, path);
    queueJoinDuration.add(Date.now() - start);

    const success = check(res, {
      'queue join 200': (r) => r.status === 200,
      'has position': (r) => {
        try {
          return JSON.parse(r.body).position !== undefined;
        } catch {
          return false;
        }
      },
    });
    queueJoinErrors.add(success ? 0 : 1);

    // Check position
    if (res.status === 200) {
      const posRes = protectedGet(token, `/api/events/${EVENT_ID}/queue/position`);
      check(posRes, { 'position 200': (r) => r.status === 200 });
    }
  });

  sleep(Math.random() * 3 + 1);
}

// Scenario 3: Seat allocation under contention
export function seatAllocation() {
  const token = registerUser();
  if (!token) {
    allocErrors.add(1);
    return;
  }

  group('Seat Allocation', () => {
    const joinPath = `/api/events/${EVENT_ID}/queue/join`;
    const joinRes = protectedPost(token, joinPath);
    if (joinRes.status !== 200 && joinRes.status !== 409) {
      allocErrors.add(1);
      return;
    }

    let admitted = false;
    for (let i = 0; i < 36; i++) {
      const enterRes = protectedPost(token, `/api/events/${EVENT_ID}/queue/enter`, JSON.stringify({}));
      if (enterRes.status === 200) {
        admitted = true;
        break;
      }
      sleep(5);
    }
    if (!admitted) {
      allocErrors.add(1);
      return;
    }

    // Get available sections
    const availRes = http.get(`${BASE_URL}/api/events/${EVENT_ID}/availability`);
    if (availRes.status !== 200) {
      allocErrors.add(1);
      return;
    }

    let sections;
    try {
      sections = JSON.parse(availRes.body).sections;
    } catch {
      allocErrors.add(1);
      return;
    }

    // Pick a random section with remaining seats
    const available = sections.filter((s) => s.remaining > 0);
    if (available.length === 0) {
      return; // No seats left, expected under load
    }
    const section = available[Math.floor(Math.random() * available.length)];
    const quantity = Math.min(Math.floor(Math.random() * 4) + 1, section.remaining);

    // Allocate seats
    const start = Date.now();
    const allocPath = `/api/events/${EVENT_ID}/allocate`;
    const allocRes = protectedPost(
      token,
      allocPath,
      JSON.stringify({ section_id: section.section_id, quantity })
    );
    seatAllocDuration.add(Date.now() - start);

    const success = check(allocRes, {
      'allocate success or expected conflict': (r) =>
        r.status === 200 || r.status === 409,
    });
    allocErrors.add(success ? 0 : 1);

    // If allocation succeeded, try to create order
    if (allocRes.status === 200) {
      let allocData;
      try {
        allocData = JSON.parse(allocRes.body);
      } catch {
        return;
      }

      const orderRes = protectedPost(
        token,
        '/api/orders',
        JSON.stringify({
          event_id: EVENT_ID,
          seats: allocData.seats,
          price_per_seat: 2800,
        })
      );
      check(orderRes, {
        'order created or payment service down': (r) =>
          r.status === 201 || r.status === 500,
      });
    }
  });

  sleep(Math.random() * 2 + 1);
}
