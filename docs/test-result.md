# 부하테스트 결과

## Stress Test - 2026-02-02 20:01

### 테스트 설정
| 항목 | 값 |
|------|-----|
| 테스트 유형 | Stress Test |
| 목적 | 1000 VUs 고부하 안정성 테스트 |
| 부하 패턴 | 0 → 1000 VUs (1분) → 유지 (3분) → 0 (1분) |
| Duration | 5분 |
| 생성:조회 비율 | 1:100 |
| Parallelism | 4 Runners (각 250 VUs) |
| 실행 시간 | 2026-02-02 19:55 KST |

### 결과 요약
| 지표 | 결과 | 기대값 | 상태 |
|------|------|--------|------|
| 총 요청 수 | 576,845 | - | - |
| 평균 RPS | ~1,920 req/s | - | - |
| 성공률 | 76.87% | >99% | **FAIL** |
| p(95) Latency | 460ms | <500ms | PASS |
| Error Rate | 23.13% | <1% | **FAIL** |
| Max VUs | 1,000 | 1,000 | PASS |

### 상세 메트릭 (4 Runners 평균)
| 메트릭 | avg | min | med | max | p(90) | p(95) |
|--------|-----|-----|-----|-----|-------|-------|
| http_req_duration | 167ms | 1.8ms | 103ms | 18.4s | 377ms | 460ms |

### Checks 결과
| Check | 성공 | 실패 | 성공률 |
|-------|------|------|--------|
| health ok | 71,729 | 0 | 100% |
| create ok | 2,685 | 865 | 76% |
| redirect ok | 153,672 | 132,543 | **53%** |

### 분석

#### 문제점
1. **Redirect 실패율 높음 (47%)**: 1000 VUs 동시 접속 시 시스템 과부하
2. **에러율 23%**: 예상 threshold(1%) 대비 매우 높음
3. **Max latency 18초**: 일부 요청에서 심각한 지연 발생

#### 병목 추정
- **RDS db.t3.micro**: CPU/Connection 한계 (micro 인스턴스 제한)
- **API Pod**: 3개 replica로는 1000 VUs 처리 불가

### 권장사항
1. **현재 인프라 안정 처리량**: ~200-300 VUs (이전 Breakpoint 결과 기반)
2. **1000 VUs 지원을 위한 개선**:
   - RDS 인스턴스 업그레이드 (db.t3.small 이상)
   - API Pod replica 증가 (3 → 6)
   - HPA 설정 검토
3. **다음 테스트**: 500 VUs로 재테스트하여 정확한 한계점 파악

### Grafana 대시보드
- URL: https://grafana.hearttune.link
- Dashboard: k6 Load Testing (ID: 19665)

---

## Breakpoint Test - 2026-02-01 20:49

### 테스트 설정
| 항목 | 값 |
|------|-----|
| 테스트 유형 | Breakpoint Test |
| 목적 | 시스템 한계점 탐색 |
| 부하 패턴 | 10 → 500 RPS (점진 증가) |
| 예상 Duration | 9분 30초 |
| 실제 Duration | 9분 30초 (정상 완료) |
| Parallelism | 3 Runners |
| 실행 시간 | 2026-02-01 20:38 KST |

### 결과 요약
| 지표 | 결과 | Threshold | 상태 |
|------|------|-----------|------|
| 총 요청 수 | 212,497 | - | - |
| 평균 RPS | ~373 req/s | - | - |
| 성공률 | 99.96% | >85% | PASS |
| p(95) Latency | 211ms | <10,000ms | PASS |
| Error Rate | 0.04% | <15% | PASS |
| Max VUs | ~500 | 1,000 | - |

### 상세 메트릭 (Runner 1 기준)
| 메트릭 | avg | min | med | max | p(90) | p(95) |
|--------|-----|-----|-----|-----|-------|-------|
| http_req_duration | 87ms | 0.4ms | 94ms | 1,340ms | 196ms | 211ms |

### 부하 단계별 진행
| 단계 | 목표 RPS | 실제 도달 여부 |
|------|---------|---------------|
| 1분 | 20 RPS | O |
| 2분 | 50 RPS | O |
| 3분 | 100 RPS | O |
| 4분 | 150 RPS | O |
| 5분 | 200 RPS | O |
| 6분 | 300 RPS | O |
| 7분 | 400 RPS | O (일부) |
| 8분 | 500 RPS | 제한적 |

### 한계점 분석
| 항목 | 값 |
|------|-----|
| 테스트 종료 | 정상 완료 (시간 만료) |
| 한계점 도달 여부 | 미도달 (threshold 이내) |
| 최대 안정 RPS | ~400 RPS |
| 최대 VUs | ~500 |
| 병목 추정 | 노드 리소스 (k6 Runner Pod 4번째 스케줄링 실패) |

### 인프라 제약사항 (해결됨)
> **업데이트 (2026-02-01)**: 아래 제약사항은 전용 loadtest 노드 그룹 추가로 해결되었습니다.

테스트 중 발견된 제약사항 (당시):
- **노드 리소스 부족**: k6 Runner Pod 4개 중 1개가 Pending (Insufficient memory, Too many pods)
- **실제 병렬성**: 요청된 parallelism=4 중 3만 실행됨

**개선 사항**:
- 전용 loadtest 노드 그룹 추가 (`t3.xlarge`, 4 vCPU, 16GB)
- 일반 노드와 분리하여 리소스 경쟁 없이 테스트 가능
- 테스트 시에만 노드 활성화 (`loadtest_node_desired_size = 1`)
- 자세한 내용: [loadtest-node.md](./loadtest-node.md)

### 권장사항
1. **현재 인프라로 안정적 처리 가능**: ~400 RPS
2. **더 높은 부하 테스트 필요시**:
   - loadtest 노드 그룹으로 분리된 환경에서 재테스트 권장
   - parallelism=4로 전체 테스트 가능
3. **실제 한계 측정을 위해**:
   - RDS db.t3.micro가 병목일 가능성 높음
   - API replicas 증가 테스트 권장

### Grafana 대시보드
- URL: https://grafana.hearttune.link
- Dashboard: k6 Load Testing (ID: 19665)
- 폴더: k6 Load Testing

---

## 테스트 이력

| 날짜 | 테스트 | 결과 | 비고 |
|------|--------|------|------|
| 2026-02-02 | Stress (1000 VUs) | FAIL | 에러율 23%, 시스템 과부하 |
| 2026-02-01 | Breakpoint | PASS | 한계점 미도달, 노드 리소스 제한 |
| 2026-01-29 | Smoke | PASS | p(95)=1.81ms |
