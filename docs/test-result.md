# 부하테스트 결과

## 테스트 방식 (2026-02-03 업데이트)

### per-vu-iterations 방식 채택

기존 시간 기반(ramping-vus) 방식에서 **횟수 기반(per-vu-iterations)** 방식으로 변경했습니다.

**변경 이유:**
- 시간 기반 방식은 VU 시작 시점 차이로 인해 각 URL의 redirect 횟수가 불균일
- 횟수 기반 방식은 각 URL이 정확히 N번 redirect 요청을 받도록 보장

**새로운 테스트 설계:**
```
각 VU가:
1. URL 생성
2. 해당 URL에 100번 redirect
3. iteration 완료
4. (iterations 수만큼 반복)
```

**Stress Test 기준:**
- 1000 VUs × 3 iterations = 3,000 URLs
- 각 URL × 100 redirect = 300,000 redirect 요청
- k6에서 100% 성공 시 → DB에 300,000 clicks 기록 예상

### 검증 방법

테스트 후 `/api/test/stats` API로 DB 통계 확인:
```json
{
  "totalUrls": 3000,      // 예상: 3000
  "totalClicks": 300000,  // 예상: 300000 (실제는 DB 동시성에 따라 다를 수 있음)
  "avgClicks": 100,       // 예상: 100
  "minClicks": 100,       // 예상: 100
  "maxClicks": 100        // 예상: 100
}
```

**참고:** k6에서 redirect 요청이 100% 성공해도 DB 기록은 동시성 이슈로 누락될 수 있습니다.
이 차이는 캐싱(Redis) 도입 전후 비교에 유용합니다.

---

## 테스트 이력

| 날짜 | 테스트 | 결과 | 비고 |
|------|--------|------|------|
| (예정) | Stress (1000 VUs, 3 iter) | - | Connection Pool 롤백 후 재테스트 |

---

## Grafana 대시보드

- URL: https://grafana.hearttune.link
- Dashboard: k6 Load Testing (ID: 19665)
