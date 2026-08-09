// k6 load test for the DocuFlow API.
//
//   k6 run -e BASE_URL=http://localhost:8080 -e EMAIL=admin@aeroxe.io \
//     -e PASSWORD=ChangeMe123! scripts/load.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 }, // ramp up
    { duration: '1m', target: 20 },  // hold
    { duration: '30s', target: 0 },  // ramp down
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],         // <1% errors
    http_req_duration: ['p(95)<500'],       // p95 < 500ms
    'http_req_duration{endpoint:list}': ['p(95)<300'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const EMAIL = __ENV.EMAIL || 'admin@aeroxe.io';
const PASSWORD = __ENV.PASSWORD || 'ChangeMe123!';

export function setup() {
  const res = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
    email: EMAIL, password: PASSWORD,
  }), { headers: { 'Content-Type': 'application/json' } });
  check(res, { 'login ok': (r) => r.status === 200 && r.json('code') === 0 });
  return res.json('data.token');
}

export default function (token) {
  const headers = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
  };
  const res = http.post(`${BASE_URL}/api/v1/documents/list`,
    JSON.stringify({ limit: 20 }), { headers, tags: { endpoint: 'list' } });
  check(res, { 'list ok': (r) => r.json('code') === 0 });
  sleep(0.2);
}
