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
const CREATE_RATIO = 20; // 1:20 비율 (생성 1회당 리다이렉트 20회)

// VU별 생성된 URL 저장
const createdUrls = [];

export default function () {
  // 1. Health check
  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health ok': (r) => r.status === 200,
  });

  // 2. 1:20 비율로 생성 vs 리다이렉트 결정
  const shouldCreate = createdUrls.length === 0 || __ITER % CREATE_RATIO === 0;

  if (shouldCreate) {
    // URL 생성
    const createRes = http.post(
      `${BASE_URL}/api/urls`,
      JSON.stringify({ originalUrl: `https://example.com/test/${__VU}/${__ITER}` }),
      { headers: { 'Content-Type': 'application/json' } }
    );

    const createSuccess = check(createRes, {
      'create status 201': (r) => r.status === 201,
      'create has shortUrl': (r) => r.json('shortUrl') !== undefined,
    });

    if (createSuccess && createRes.status === 201) {
      const shortUrl = createRes.json('shortUrl');
      createdUrls.push(shortUrl);

      // 생성 직후 리다이렉트 테스트
      const redirectRes = http.get(`${BASE_URL}/r/${shortUrl}`, { redirects: 0 });
      check(redirectRes, {
        'redirect status 302': (r) => r.status === 302,
      });
    }
  } else {
    // 기존 URL로 리다이렉트만
    const randomUrl = createdUrls[Math.floor(Math.random() * createdUrls.length)];
    const redirectRes = http.get(`${BASE_URL}/r/${randomUrl}`, { redirects: 0 });
    check(redirectRes, {
      'redirect status 302': (r) => r.status === 302,
    });
  }

  sleep(1);
}
