# 부하테스트 결과

## Stress Test - 2026-02-02 22:09

### 테스트 설정
| 항목 | 값 |
|------|-----|
| 테스트 유형 | Stress Test |
| 목적 | 1000 VUs 고부하 안정성 테스트 |
| 부하 패턴 | 0 → 1000 VUs (1분) → 유지 (5분) |
| Duration | 6분 (생성 4분 + redirect 2분) |
| 생성:조회 비율 | 1:100 (URL당 100번 조회) |
| Parallelism | 4 Runners (각 250 VUs) |
| 실행 시간 | 2026-02-02 22:09 KST |
| DB 초기화 | 테스트 전 수행 |

### 결과 요약
| 지표 | 결과 | 기대값 | 상태 |
|------|------|--------|------|
| 총 요청 수 | 818,135 | - | - |
| 평균 RPS | ~2,267 req/s | - | - |
| 성공률 | 78.30% | >99% | **FAIL** |
| p(95) Latency | 469ms | <500ms | PASS |
| Error Rate | 21.69% | <1% | **FAIL** |
| Max VUs | 1,000 | 1,000 | PASS |

### 상세 메트릭 (4 Runners 평균)
| 메트릭 | avg | min | med | max | p(90) | p(95) |
|--------|-----|-----|-----|-----|-------|-------|
| http_req_duration | 155ms | 1.9ms | 85ms | 17.7s | 376ms | 469ms |

### Checks 결과
| Check | 성공 | 실패 | 성공률 |
|-------|------|------|--------|
| health ok | ~407,000 | 0 | 100% |
| create ok | 3,014 | 1,213 | 71% |
| redirect ok | 230,621 | 176,270 | **56%** |

### DB 통계 (테스트 후)
| 항목 | 값 |
|------|-----|
| 총 URL 수 | 3,014 |
| 총 클릭 수 | 125,379 |
| 평균 클릭 | 41.60 |
| 최소 클릭 | 13 |
| 최대 클릭 | 88 |

### 분석

#### 테스트 설계
- **4분까지 URL 생성**, 이후 2분은 redirect만 수행
- 모든 URL이 최소 100번 redirect 요청을 받도록 설계
- minClicks > 0 확인으로 모든 URL이 요청을 받았음을 검증

#### 평균 클릭수 분석
- 목표: 100회 (URL당 100번 redirect 시도)
- 실제: 41.60회
- 원인: redirect 성공률 56% (서버 부하로 인한 실패)
- 계산: 100회 시도 × 56% 성공률 ≈ 56회 (실제보다 높은 이유: 일부 URL은 100번 미만 시도)

#### 병목 지점
- **RDS db.t3.micro**: CPU/Connection 한계로 인한 높은 실패율
- **API Pod 2개**: 1000 VUs 동시 처리에 부족

#### 원인 분석 (로그 확인)

API Pod 로그에서 핵심 에러 발견:

```
Error 1040: Too many connections
Error 1040 (08004): Too many connections
```

**Root Cause**: DB Connection Pool 미설정
- API 코드에서 GORM Connection Pool 설정이 없음
- 각 Pod가 무제한으로 DB 연결 생성 시도
- RDS db.t3.micro의 max_connections (~66-87) 초과
- 연결 한도 도달 시 즉시 에러 반환 → redirect 실패

**추가 발견**:
- SLOW SQL 경고 (200ms 이상): SELECT/UPDATE 쿼리 지연
- 일부 쿼리 1000ms+ 소요 (정상 시 ~10ms)

### 해결 방안

#### 즉시 적용 (코드 수정)
Connection Pool 설정 추가 (`apps/api/internal/infrastructure/config.go`):

```go
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(25)               // Pod당 최대 25개 연결
sqlDB.SetMaxIdleConns(10)               // 유휴 연결 10개 유지
sqlDB.SetConnMaxLifetime(5 * time.Minute) // 연결 수명 5분
```

#### 추가 개선 (인프라)
1. **RDS 업그레이드**: db.t3.small 이상 (max_connections 증가)
2. **API Pod 증가**: 2 → 4개
3. **RDS Proxy 도입**: 대규모 스케일 시 고려

### 다음 테스트
1. Connection Pool 설정 후 동일 조건 재테스트
2. 성공률 99% 이상 달성 여부 확인

### Grafana 대시보드
- URL: https://grafana.hearttune.link
- Dashboard: k6 Load Testing (ID: 19665)

---

## 테스트 이력

| 날짜 | 테스트 | 결과 | 비고 |
|------|--------|------|------|
| 2026-02-02 22:09 | Stress (1000 VUs, 6분) | FAIL | 에러율 22%, avgClicks 41.60 |
