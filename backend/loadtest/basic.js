// k6 load test script for ticketing system
// Run: k6 run --vus 100 --duration 30s loadtest/basic.js

import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  scenarios: {
    queue_rush: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 100 },
        { duration: '30s', target: 1000 },
        { duration: '10s', target: 10000 },
        { duration: '30s', target: 10000 },
        { duration: '10s', target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.05'],
  },
};

export default function () {
  // List events
  const eventsRes = http.get(`${BASE_URL}/api/events`);
  check(eventsRes, { 'events 200': (r) => r.status === 200 });

  // Get event detail
  const eventId = 'e0000000-0000-0000-0000-000000000001';
  const detailRes = http.get(`${BASE_URL}/api/events/${eventId}`);
  check(detailRes, { 'detail 200': (r) => r.status === 200 });

  // Get availability
  const availRes = http.get(`${BASE_URL}/api/events/${eventId}/availability`);
  check(availRes, { 'avail 200': (r) => r.status === 200 });

  sleep(0.5);
}
