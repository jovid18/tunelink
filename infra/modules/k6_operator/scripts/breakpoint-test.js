import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    breakpoint: {
      executor: 'ramping-arrival-rate',
      startRate: 10,           // 초당 10 요청으로 시작
      timeUnit: '1s',
      preAllocatedVUs: 500,    // 최대 500 VUs 준비
      maxVUs: 1000,            // 절대 한계
      stages: [
        { duration: '1m', target: 20 },    // 초당 20 요청
        { duration: '1m', target: 50 },    // 초당 50 요청
        { duration: '1m', target: 100 },   // 초당 100 요청
        { duration: '1m', target: 150 },   // 초당 150 요청
        { duration: '1m', target: 200 },   // 초당 200 요청
        { duration: '1m', target: 300 },   // 초당 300 요청
        { duration: '1m', target: 400 },   // 초당 400 요청
        { duration: '1m', target: 500 },   // 초당 500 요청
        { duration: '1m', target: 0 },     // ramp down
      ],
    },
  },
  thresholds: {
    // 에러율 15% 미만이면 통과, 초과시 실패 및 중단 (한계점 도달)
    http_req_failed: [{ threshold: 'rate<0.15', abortOnFail: true, delayAbortEval: '30s' }],
    // p(95) 응답시간 10초 미만이면 통과, 초과시 실패 및 중단
    http_req_duration: [{ threshold: 'p(95)<10000', abortOnFail: true, delayAbortEval: '30s' }],
  },
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
      JSON.stringify({ originalUrl: `https://example.com/breakpoint/${__VU}/${__ITER}` }),
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
    if (createdUrls.length > 0) {
      const randomUrl = createdUrls[Math.floor(Math.random() * createdUrls.length)];
      const redirectRes = http.get(`${BASE_URL}/r/${randomUrl}`, { redirects: 0 });
      check(redirectRes, {
        'redirect ok': (r) => r.status === 302,
      });
    }
  }

  // 부하를 높이기 위해 sleep 최소화
  sleep(0.1);
}

export function handleSummary(data) {
  // 테스트 종료 시 한계점 정보 출력
  const metrics = data.metrics;

  console.log('\n========== BREAKPOINT TEST SUMMARY ==========');
  console.log(`Total Requests: ${metrics.http_reqs?.values?.count || 'N/A'}`);
  console.log(`Error Rate: ${((metrics.http_req_failed?.values?.rate || 0) * 100).toFixed(2)}%`);
  console.log(`Avg Response Time: ${(metrics.http_req_duration?.values?.avg || 0).toFixed(2)}ms`);
  console.log(`p(95) Response Time: ${(metrics.http_req_duration?.values?.['p(95)'] || 0).toFixed(2)}ms`);
  console.log(`p(99) Response Time: ${(metrics.http_req_duration?.values?.['p(99)'] || 0).toFixed(2)}ms`);
  console.log(`Max Response Time: ${(metrics.http_req_duration?.values?.max || 0).toFixed(2)}ms`);
  console.log(`Max VUs: ${metrics.vus_max?.values?.value || 'N/A'}`);
  console.log('==============================================\n');

  return {
    stdout: JSON.stringify(data, null, 2),
  };
}
