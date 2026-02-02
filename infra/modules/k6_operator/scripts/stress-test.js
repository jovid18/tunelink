import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '1m', target: 1000 },  // ramp up to 1000 users
    { duration: '3m', target: 1000 },  // stay at 1000 users (생성 가능)
    { duration: '2m', target: 1000 },  // stay at 1000 users (redirect만)
  ],
};

const BASE_URL = 'https://hearttune.link';
const CLICKS_PER_URL = 100; // URL 1개당 100번 조회 후 새 URL 생성

// 시간 기반 URL 생성 제한
const CREATE_CUTOFF_SEC = 4 * 60; // 4분까지만 URL 생성, 이후 2분은 redirect만

// VU별 현재 URL과 조회 카운트
let currentUrl = null;
let redirectCount = 0;
const testStartTime = Date.now();

export default function () {
  // 1. Health check (lightweight)
  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health ok': (r) => r.status === 200,
  });

  // 2. 시간 체크 - 4분 이후에는 새 URL 생성 안 함
  const elapsedSec = (Date.now() - testStartTime) / 1000;
  const canCreateNewUrl = elapsedSec < CREATE_CUTOFF_SEC;

  // 3. 100번 조회 완료 시 새 URL 생성 (시간 내에만)
  if (currentUrl === null || (redirectCount >= CLICKS_PER_URL && canCreateNewUrl)) {
    const createRes = http.post(
      `${BASE_URL}/api/urls`,
      JSON.stringify({ originalUrl: `https://example.com/stress/${__VU}/${Date.now()}` }),
      { headers: { 'Content-Type': 'application/json' } }
    );

    const createSuccess = check(createRes, {
      'create ok': (r) => r.status === 201,
    });

    if (createSuccess && createRes.status === 201) {
      currentUrl = createRes.json('shortUrl');
      redirectCount = 0;
    }
  }

  // 4. Redirect (URL이 있을 때만)
  if (currentUrl) {
    const redirectRes = http.get(`${BASE_URL}/r/${currentUrl}`, { redirects: 0 });
    check(redirectRes, {
      'redirect ok': (r) => r.status === 302,
    });
    redirectCount++;
  }

  sleep(0.5);
}
