# 부하테스트 결과

## Stress Test - 2026-02-03 00:12

### 테스트 설정
| 항목 | 값 |
|------|-----|
| 테스트 유형 | Stress Test |
| VUs | 1000 (4 runners × 250) |
| Iterations | 3 per VU (총 3000) |
| Max Duration | 10m |
| Parallelism | 4 |
| 실행 시간 | 2026-02-03 00:12:02 KST |

### 결과 요약

**4개 Runner Pod 합산 결과:**

| 지표 | 결과 | Threshold | 상태 |
|------|------|-----------|------|
| 총 요청 수 | 126,900 | - | - |
| 성공률 | 71.8% | >99% | **FAIL** |
| 에러율 | 28.2% | <1% | **FAIL** |
| p(95) Latency | ~490ms | <500ms | PASS |
| p(99) Latency | ~1.1s | - | - |

### 상세 메트릭 (Runner별 평균)

| 메트릭 | avg | min | med | max | p(90) | p(95) | p(99) |
|--------|-----|-----|-----|-----|-------|-------|-------|
| http_req_duration | ~269ms | ~2ms | ~211ms | ~18s | ~400ms | ~490ms | ~1.1s |
| http_req_duration (success) | ~261ms | ~2ms | ~219ms | ~4s | ~406ms | ~486ms | ~706ms |

### Checks 결과

| Check | Runner 1 | Runner 2 | Runner 3 | Runner 4 |
|-------|----------|----------|----------|----------|
| health ok | 100% | 100% | 100% | 100% |
| create ok | 54% | 35% | 34% | 37% |
| redirect ok | 74% | 70% | 70% | 72% |

**합산:**
- Total Checks: ~126,900
- Checks Succeeded: ~71.8%
- Checks Failed: ~28.2%

### DB 통계 (테스트 후)
| 항목 | 값 |
|------|-----|
| 총 URL 수 | 1,209 |
| 총 클릭 수 | 63,276 |
| 평균 클릭 | 52.3 |
| 최소 클릭 | 21 |
| 최대 클릭 | 100 |

### 처리량
| Runner | Requests/s | Iterations/s |
|--------|------------|--------------|
| Runner 1 | 348/s | 6.19/s |
| Runner 2 | 232/s | 6.27/s |
| Runner 3 | 225/s | 6.16/s |
| Runner 4 | 242/s | 6.16/s |
| **합계** | **~1,047/s** | **~24.8/s** |

### 분석

**문제점:**
1. **URL 생성 실패율 높음 (54-66%)**: 동시 요청 시 URL 생성 API에서 병목 발생
2. **Redirect 실패율 ~28%**: 생성되지 않은 URL에 대한 redirect 요청 실패
3. **최대 응답시간 17-18초**: 일부 요청에서 심각한 지연 발생
4. **예상 URL 수 불일치**: 3000 iterations에서 예상 1209개 URL 생성 (약 40%)

**병목 추정:**
- RDS Connection Pool 한계
- API 서버 동시 처리 용량
- 동시 URL 생성 시 충돌/타임아웃

### 권장사항
1. URL 생성 API의 동시성 처리 개선 필요
2. RDS Connection Pool 확장 검토
3. API 서버 수평 확장 (HPA 설정 조정)
4. 재시도 로직 및 에러 핸들링 강화

### Grafana 대시보드

![Stress Test 2026-02-03 00:12](./images/stress-test-20260203-0012-grafana.png)

**Grafana 메트릭 (스크린샷 기준):**
| 항목 | 값 |
|------|-----|
| HTTP requests | 126,900 |
| HTTP request failures | 35,383 (27.9%) |
| Peak RPS | 363 req/s |
| HTTP Request Duration | 1.22s |

- URL: http://grafana.hearttune.link
- Dashboard: k6 Load Testing (ID: 19665)

---
