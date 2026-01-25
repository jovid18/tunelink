import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '2m', target: 100 },  // ramp up to 100 users
    { duration: '5m', target: 100 },  // stay at 100 users
    { duration: '2m', target: 200 },  // ramp up to 200 users
    { duration: '5m', target: 200 },  // stay at 200 users
    { duration: '2m', target: 0 },    // ramp down
  ],
};

const BASE_URL = 'http://tunelink-dev-api.tunelink.svc.cluster.local';

export default function () {
  // 1. Health check (lightweight)
  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health ok': (r) => r.status === 200,
  });

  // 2. Create short URL
  const createRes = http.post(
    `${BASE_URL}/api/urls`,
    JSON.stringify({ originalUrl: `https://example.com/stress/${__VU}/${__ITER}` }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  const createSuccess = check(createRes, {
    'create ok': (r) => r.status === 201,
  });

  // 3. Redirect test
  if (createSuccess && createRes.status === 201) {
    const shortUrl = createRes.json('shortUrl');
    const redirectRes = http.get(`${BASE_URL}/r/${shortUrl}`, {
      redirects: 0,
    });
    check(redirectRes, {
      'redirect ok': (r) => r.status === 302,
    });
  }

  sleep(0.5);
}
