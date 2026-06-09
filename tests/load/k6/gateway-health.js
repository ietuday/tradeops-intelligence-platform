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

export default function () {
  for (const path of ['/health', '/ready']) {
    const res = http.get(`${BASE_URL}${path}`, {
      headers: { 'x-correlation-id': `k6-gateway-${__VU}-${__ITER}` },
    });
    check(res, {
      [`${path} returned 200`]: (r) => r.status === 200,
    });
  }
}
