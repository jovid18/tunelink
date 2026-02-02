---
name: k6-load-test
description: k6 부하테스트 실행 및 결과 문서화. Smoke/Load/Stress/Breakpoint 테스트 중 선택하여 실행하고, 결과를 test-result.md에 정리한 뒤 리소스를 정리합니다.
allowed-tools:
  - Bash
  - Read
  - Write
  - Edit
  - AskUserQuestion
---

# k6 Load Test Skill

EKS 클러스터에서 k6 부하테스트를 실행하고 결과를 문서화합니다.

## 워크플로우

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                           /k6-load-test                                      │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌───────────┐  │
│  │  Step 0  │──▶│  Step 1  │──▶│  Step 2  │──▶│  Step 3  │──▶│  Step 4   │  │
│  │DB 초기화 │   │테스트선택│   │테스트실행│   │DB 통계   │   │결과 문서화│  │
│  │  확인    │   │          │   │  & 모니터│   │  수집    │   │  & 정리   │  │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘   └───────────┘  │
│       │              │              │              │               │         │
│       ▼              ▼              ▼              ▼               ▼         │
│  [Yes/No 질문] [사용자 선택] [TestRun 생성]  [API 호출]    [test-result.md] │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

## Test API Endpoints

테스트 데이터 관리를 위한 API:

| Endpoint | Method | 설명 |
|----------|--------|------|
| `/api/test/stats` | GET | URL 통계 조회 (count, clicks 합계/평균/최소/최대) |
| `/api/test/urls` | DELETE | 모든 URL 삭제 (테스트 데이터 초기화) |

## 실행 단계

### Step 0: DB 초기화 확인

테스트 시작 전에 DB 데이터 초기화 여부를 확인합니다.

**AskUserQuestion으로 질문:**
```
question: "테스트 전에 DB 데이터를 초기화할까요?"
header: "DB 초기화"
options:
  - label: "예, 초기화"
    description: "DELETE /api/test/urls 호출하여 urls 테이블 초기화"
  - label: "아니오, 유지"
    description: "기존 데이터 유지 (결과 분석 시 주의 필요)"
```

**"예" 선택 시:**
```bash
curl -X DELETE https://hearttune.link/api/test/urls
```

> 어떤 응답이든 테스트는 진행됩니다.

### Step 1: 테스트 유형 선택

AskUserQuestion 도구로 사용자에게 테스트 유형을 선택하게 합니다:

```
question: "어떤 부하테스트를 실행할까요?"
header: "테스트 선택"
options:
  - label: "Smoke Test (Recommended)"
    description: "5 VUs, 30초 - 기본 동작 확인용. 빠르게 시스템 정상 동작 검증"
  - label: "Load Test"
    description: "20→50 VUs, 7분 - 일반적인 부하 상황 테스트"
  - label: "Stress Test"
    description: "100→200 VUs, 16분 - 높은 부하에서 시스템 안정성 테스트"
  - label: "Breakpoint Test"
    description: "10→500 RPS, ~9분 - 시스템 한계점 탐색 (에러 15% 또는 p95>10s 시 자동 중단)"
```

### Step 2: 테스트 실행

선택된 테스트에 따라 kubectl로 TestRun 리소스를 생성합니다.

**중요: 테스트 전 loadtest 노드 활성화 필요**
```bash
# 1. infra/main.tf에서 loadtest_node_desired_size = 1 로 변경
# 2. terraform apply -target=module.eks
# 3. 노드 Ready 확인: kubectl get nodes -l role=loadtest
```

**Smoke Test:**
```bash
kubectl apply -f - <<EOF
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: smoke-test-$(date +%Y%m%d-%H%M%S)
  namespace: tunelink
spec:
  parallelism: 4
  script:
    configMap:
      name: k6-test-scripts
      file: smoke-test.js
  arguments: --out experimental-prometheus-rw
  runner:
    nodeSelector:
      role: loadtest
    tolerations:
      - key: "role"
        operator: "Equal"
        value: "loadtest"
        effect: "NoSchedule"
    env:
      - name: K6_PROMETHEUS_RW_SERVER_URL
        value: http://prometheus-kube-prometheus-prometheus.monitoring.svc.cluster.local:9090/api/v1/write
EOF
```

**Load Test:**
```bash
kubectl apply -f - <<EOF
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: load-test-$(date +%Y%m%d-%H%M%S)
  namespace: tunelink
spec:
  parallelism: 4
  script:
    configMap:
      name: k6-test-scripts
      file: load-test.js
  arguments: --out experimental-prometheus-rw
  runner:
    nodeSelector:
      role: loadtest
    tolerations:
      - key: "role"
        operator: "Equal"
        value: "loadtest"
        effect: "NoSchedule"
    env:
      - name: K6_PROMETHEUS_RW_SERVER_URL
        value: http://prometheus-kube-prometheus-prometheus.monitoring.svc.cluster.local:9090/api/v1/write
EOF
```

**Stress Test:**
```bash
kubectl apply -f - <<EOF
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: stress-test-$(date +%Y%m%d-%H%M%S)
  namespace: tunelink
spec:
  parallelism: 4
  script:
    configMap:
      name: k6-test-scripts
      file: stress-test.js
  arguments: --out experimental-prometheus-rw
  runner:
    nodeSelector:
      role: loadtest
    tolerations:
      - key: "role"
        operator: "Equal"
        value: "loadtest"
        effect: "NoSchedule"
    env:
      - name: K6_PROMETHEUS_RW_SERVER_URL
        value: http://prometheus-kube-prometheus-prometheus.monitoring.svc.cluster.local:9090/api/v1/write
EOF
```

**Breakpoint Test:**
```bash
kubectl apply -f - <<EOF
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: breakpoint-test-$(date +%Y%m%d-%H%M%S)
  namespace: tunelink
spec:
  parallelism: 4
  script:
    configMap:
      name: k6-test-scripts
      file: breakpoint-test.js
  arguments: --out experimental-prometheus-rw
  runner:
    nodeSelector:
      role: loadtest
    tolerations:
      - key: "role"
        operator: "Equal"
        value: "loadtest"
        effect: "NoSchedule"
    env:
      - name: K6_PROMETHEUS_RW_SERVER_URL
        value: http://prometheus-kube-prometheus-prometheus.monitoring.svc.cluster.local:9090/api/v1/write
    resources:
      requests:
        cpu: "500m"
        memory: "512Mi"
      limits:
        cpu: "1000m"
        memory: "1Gi"
EOF
```

### Step 3: 테스트 모니터링 및 결과 수집

1. **테스트 상태 확인** (주기적으로 체크):
```bash
kubectl get testrun -n tunelink
```

2. **Runner Pod 확인**:
```bash
kubectl get pods -n tunelink -l app=k6
```

3. **테스트 완료 대기 및 로그 수집**:
```bash
# 테스트가 완료될 때까지 대기 후 로그 수집
kubectl logs -n tunelink -l app=k6 --tail=200
```

4. **결과에서 핵심 메트릭 추출**:

k6 출력에서 다음 메트릭을 파싱합니다:
- `http_req_duration` (avg, min, med, max, p90, p95)
- `http_req_failed` (에러율)
- `http_reqs` (총 요청 수)
- `vus` / `vus_max` (가상 사용자 수)
- `iterations` (총 반복 수)
- `checks` (체크 통과율)

**Breakpoint Test 추가 메트릭:**
- 테스트 중단 시점의 RPS (초당 요청 수)
- 중단 원인 (에러율 초과 or 응답시간 초과)
- 최대 VUs 도달 수

### Step 3.5: DB 통계 수집

테스트 완료 후, 리포트 작성 전에 DB 통계를 수집합니다.

**API 호출:**
```bash
curl -s https://hearttune.link/api/test/stats
```

**응답 예시:**
```json
{
  "totalUrls": 1000,
  "totalClicks": 100000,
  "avgClicks": 100,
  "minClicks": 95,
  "maxClicks": 105
}
```

이 데이터를 결과 문서에 포함합니다.

### Step 4: 결과 문서화

`test-result.md` 파일에 결과를 기록합니다. 파일이 없으면 새로 생성하고, 있으면 최신 결과를 상단에 추가합니다.

**문서 형식:**

```markdown
# 부하테스트 결과

## [테스트 유형] - YYYY-MM-DD HH:MM

### 테스트 설정
| 항목 | 값 |
|------|-----|
| 테스트 유형 | Smoke / Load / Stress / Breakpoint |
| VUs | 5 / 20-50 / 100-200 / max 1000 |
| Duration | 30s / 7m / 16m / ~9m (자동 중단) |
| 실행 시간 | YYYY-MM-DD HH:MM:SS |

### 결과 요약
| 지표 | 결과 | Threshold | 상태 |
|------|------|-----------|------|
| 총 요청 수 | XXX | - | - |
| 성공률 | XX.XX% | >99% | PASS/FAIL |
| p(95) Latency | XXms | <500ms | PASS/FAIL |
| Error Rate | X.XX% | <1% | PASS/FAIL |

### 상세 메트릭
| 메트릭 | avg | min | med | max | p(90) | p(95) |
|--------|-----|-----|-----|-----|-------|-------|
| http_req_duration | - | - | - | - | - | - |

### Checks
| Check | 통과율 |
|-------|--------|
| health status 200 | XX% |
| create ok | XX% |
| redirect ok | XX% |

### DB 통계 (테스트 후)
| 항목 | 값 |
|------|-----|
| 총 URL 수 | XXX |
| 총 클릭 수 | XXX |
| 평균 클릭 | XX.X |
| 최소 클릭 | XX |
| 최대 클릭 | XX |

### Grafana 대시보드
- URL: http://grafana.hearttune.link
- Dashboard: k6 Load Testing (ID: 19665)

---
```

**Breakpoint Test 추가 섹션:**

```markdown
### 한계점 분석 (Breakpoint Test)
| 항목 | 값 |
|------|-----|
| 중단 시점 RPS | XXX req/s |
| 중단 원인 | 에러율 15% 초과 / p(95) > 10초 |
| 최대 VUs | XXX |
| 병목 추정 | RDS / API / Network |

### 권장사항
- 현재 인프라로 안정적으로 처리 가능한 RPS: ~XXX
- 스케일업 필요 시: [구체적 권장사항]
```

### Step 5: 리소스 정리

테스트 완료 후 TestRun 리소스를 삭제합니다:

```bash
kubectl delete testrun [테스트명] -n tunelink
```

## 테스트 유형 비교

| 유형 | 부하 | 시간 | 목적 | 중단 조건 |
|------|------|------|------|----------|
| Smoke | 5 VUs | 30초 | 기본 동작 확인 | 시간 |
| Load | 20→50 VUs | 7분 | 일반 부하 테스트 | 시간 |
| Stress | 100→200 VUs | 16분 | 고부하 안정성 | 시간 |
| **Breakpoint** | **10→500 RPS** | **~9분** | **한계점 탐색** | **에러율/응답시간** |

## 에러 처리

- **kubectl 연결 실패**: AWS_PROFILE 설정 및 kubeconfig 확인 안내
- **TestRun 생성 실패**: k6-operator 상태 및 ConfigMap 존재 여부 확인
- **테스트 실패**: 로그 분석 후 원인 보고
- **Breakpoint 조기 중단**: 정상 동작 (한계점 도달), 중단 시점 메트릭 기록

## 주의사항

1. AWS_PROFILE이 설정되어 있어야 합니다 (tunelink)
2. kubeconfig가 EKS 클러스터에 연결되어 있어야 합니다
3. Stress/Breakpoint Test는 시간이 오래 걸리고 클러스터에 부하를 줄 수 있습니다
4. 테스트 중 다른 테스트를 동시에 실행하지 않는 것이 좋습니다
5. **Breakpoint Test 주의**: 시스템 한계까지 부하를 주므로 프로덕션 환경에서는 실행하지 마세요

## 참고 문서

- 부하테스트 설정: `docs/test.md`
- 모니터링 설정: `docs/monitoring.md`
- 테스트 결과: `docs/test-result.md`
- 노드 격리 설정: `docs/loadtest-node.md`
- k6 스크립트: `infra/modules/k6_operator/scripts/`
