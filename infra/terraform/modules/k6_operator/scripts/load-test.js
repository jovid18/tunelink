import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '1m', target: 20 },  // ramp up to 20 users
    { duration: '2m', target: 20 },  // stay at 20 users
    { duration: '1m', target: 50 },  // ramp up to 50 users
    { duration: '2m', target: 50 },  // stay at 50 users
    { duration: '1m', target: 0 },   // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    http_req_failed: ['rate<0.05'],
  },
};

const BASE_URL = 'http://tunelink-dev-api.tunelink.svc.cluster.local';

export default function () {
  // 1. Health check
  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health ok': (r) => r.status === 200,
  });

  // 2. Create short URL
  const createRes = http.post(
    `${BASE_URL}/api/urls`,
    JSON.stringify({ originalUrl: `https://example.com/test/${__VU}/${__ITER}` }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  const createSuccess = check(createRes, {
    'create status 201': (r) => r.status === 201,
    'create has shortUrl': (r) => r.json('shortUrl') !== undefined,
    'create has fullUrl': (r) => r.json('fullUrl') !== undefined,
  });

  // 3. Redirect test (only if create succeeded)
  if (createSuccess && createRes.status === 201) {
    const shortUrl = createRes.json('shortUrl');
    const redirectRes = http.get(`${BASE_URL}/r/${shortUrl}`, {
      redirects: 0,  // don't follow redirects
    });
    check(redirectRes, {
      'redirect status 302': (r) => r.status === 302,
      'redirect has location header': (r) => r.headers['Location'] !== undefined,
    });
  }

  sleep(1);
}
