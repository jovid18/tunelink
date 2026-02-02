import http from 'k6/http';
import { check, sleep } from 'k6';

// Breakpoint Test: 시스템 한계점 탐색
// ramping-arrival-rate로 RPS를 점진적으로 증가시켜 한계점 도달 시 자동 중단
export const options = {
  scenarios: {
    breakpoint: {
      executor: 'ramping-arrival-rate',
      startRate: 10,
      timeUnit: '1s',
      preAllocatedVUs: 500,
      maxVUs: 1000,
      stages: [
        { duration: '1m', target: 50 },
        { duration: '1m', target: 100 },
        { duration: '1m', target: 200 },
        { duration: '1m', target: 300 },
        { duration: '1m', target: 400 },
        { duration: '1m', target: 500 },
        { duration: '1m', target: 0 },
      ],
    },
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
  thresholds: {
    http_req_failed: [{ threshold: 'rate<0.15', abortOnFail: true, delayAbortEval: '30s' }],
    http_req_duration: [{ threshold: 'p(95)<10000', abortOnFail: true, delayAbortEval: '30s' }],
  },
};

const BASE_URL = 'https://hearttune.link';
const CLICKS_PER_URL = 100;

// VU별 URL 관리
let currentUrl = null;
let redirectCount = 0;

export default function () {
  // 1. Health check
  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health ok': (r) => r.status === 200,
  });

  // 2. 100번 완료 또는 URL 없으면 새 URL 생성
  if (currentUrl === null || redirectCount >= CLICKS_PER_URL) {
    const createRes = http.post(
      `${BASE_URL}/api/urls`,
      JSON.stringify({ originalUrl: `https://example.com/breakpoint/${__VU}/${Date.now()}` }),
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

  // 3. Redirect
  if (currentUrl) {
    const redirectRes = http.get(`${BASE_URL}/r/${currentUrl}`, { redirects: 0 });
    check(redirectRes, {
      'redirect ok': (r) => r.status === 302,
    });
    redirectCount++;
  }

  sleep(0.1);
}

export function handleSummary(data) {
  const metrics = data.metrics;

  console.log('\n========== BREAKPOINT TEST SUMMARY ==========');
  console.log(`Total Requests: ${metrics.http_reqs?.values?.count || 'N/A'}`);
  console.log(`Error Rate: ${((metrics.http_req_failed?.values?.rate || 0) * 100).toFixed(2)}%`);
  console.log(`Avg Response Time: ${(metrics.http_req_duration?.values?.avg || 0).toFixed(2)}ms`);
  console.log(`p(95) Response Time: ${(metrics.http_req_duration?.values?.['p(95)'] || 0).toFixed(2)}ms`);
  console.log(`Max VUs: ${metrics.vus_max?.values?.value || 'N/A'}`);
  console.log('==============================================\n');

  return {
    stdout: JSON.stringify(data, null, 2),
  };
}
