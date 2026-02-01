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
const CREATE_RATIO = 20; // 1:20 비율 (생성 1회당 리다이렉트 20회)

// VU별 생성된 URL 저장
const createdUrls = [];

export default function () {
  // 1. Health check (lightweight)
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
      JSON.stringify({ originalUrl: `https://example.com/stress/${__VU}/${__ITER}` }),
      { headers: { 'Content-Type': 'application/json' } }
    );

    const createSuccess = check(createRes, {
      'create ok': (r) => r.status === 201,
    });

    if (createSuccess && createRes.status === 201) {
      const shortUrl = createRes.json('shortUrl');
      createdUrls.push(shortUrl);

      // 생성 직후 리다이렉트 테스트
      const redirectRes = http.get(`${BASE_URL}/r/${shortUrl}`, { redirects: 0 });
      check(redirectRes, {
        'redirect ok': (r) => r.status === 302,
      });
    }
  } else {
    // 기존 URL로 리다이렉트만
    const randomUrl = createdUrls[Math.floor(Math.random() * createdUrls.length)];
    const redirectRes = http.get(`${BASE_URL}/r/${randomUrl}`, { redirects: 0 });
    check(redirectRes, {
      'redirect ok': (r) => r.status === 302,
    });
  }

  sleep(0.5);
}
