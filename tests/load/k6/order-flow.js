import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: Number(__ENV.VUS || 5),
  duration: __ENV.DURATION || '30s',
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<1000'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TOKEN = __ENV.TOKEN || '';

export default function () {
  const health = http.get(`${BASE_URL}/api/orders/health`);
  check(health, { 'orders health returned 200': (r) => r.status === 200 });

  if (!TOKEN) {
    if (__ITER === 0) {
      console.error('TOKEN is required for POST /api/orders; skipping protected order creation.');
    }
    return;
  }

  const body = JSON.stringify({
    symbol: 'AAPL',
    side: 'BUY',
    orderType: 'MARKET',
    quantity: 1,
    limitPrice: null,
    stopPrice: null,
  });

  const res = http.post(`${BASE_URL}/api/orders`, body, {
    headers: {
      Authorization: `Bearer ${TOKEN}`,
      'Content-Type': 'application/json',
      'Idempotency-Key': `k6-order-${__VU}-${__ITER}`,
      'x-correlation-id': `k6-order-${__VU}-${__ITER}`,
    },
  });

  check(res, {
    'order create returned non-5xx': (r) => r.status < 500,
  });
}
