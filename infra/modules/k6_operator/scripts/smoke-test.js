import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    smoke: {
      executor: 'per-vu-iterations',
      vus: 5,
      iterations: 1,  // 각 VU가 1개 URL 생성
      maxDuration: '2m',
    },
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

const BASE_URL = 'https://hearttune.link';
const CLICKS_PER_URL = 10;  // Smoke test는 10번만

export default function () {
  // 1. Health check
  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health ok': (r) => r.status === 200,
  });

  // 2. URL 생성
  const createRes = http.post(
    `${BASE_URL}/api/urls`,
    JSON.stringify({ originalUrl: `https://example.com/smoke/${__VU}/${Date.now()}` }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  const createSuccess = check(createRes, {
    'create ok': (r) => r.status === 201,
  });

  if (!createSuccess) {
    return;
  }

  const shortUrl = createRes.json('shortUrl');

  // 3. 10번 redirect
  for (let i = 0; i < CLICKS_PER_URL; i++) {
    const redirectRes = http.get(`${BASE_URL}/r/${shortUrl}`, { redirects: 0 });
    check(redirectRes, {
      'redirect ok': (r) => r.status === 302,
    });
    sleep(0.1);
  }
}
