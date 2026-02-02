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

const BASE_URL = 'https://hearttune.link';
const CLICKS_PER_URL = 20; // URL 1개당 20번 조회 후 새 URL 생성

// VU별 현재 URL과 조회 카운트
let currentUrl = null;
let redirectCount = 0;

export default function () {
  // 1. Health check
  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health ok': (r) => r.status === 200,
  });

  // 2. 20번 조회 완료 시 새 URL 생성
  if (currentUrl === null || redirectCount >= CLICKS_PER_URL) {
    const createRes = http.post(
      `${BASE_URL}/api/urls`,
      JSON.stringify({ originalUrl: `https://example.com/load/${__VU}/${Date.now()}` }),
      { headers: { 'Content-Type': 'application/json' } }
    );

    const createSuccess = check(createRes, {
      'create status 201': (r) => r.status === 201,
      'create has shortUrl': (r) => r.json('shortUrl') !== undefined,
    });

    if (createSuccess && createRes.status === 201) {
      currentUrl = createRes.json('shortUrl');
      redirectCount = 0;
    }
  }

  // 3. Redirect (URL이 있을 때만)
  if (currentUrl) {
    const redirectRes = http.get(`${BASE_URL}/r/${currentUrl}`, { redirects: 0 });
    check(redirectRes, {
      'redirect status 302': (r) => r.status === 302,
    });
    redirectCount++;
  }

  sleep(1);
}
