# 부하테스트

## 도구 선정: k6

### 후보군 비교

| 도구    | 평가     | 비고                    |
| ------- | -------- | ----------------------- |
| k6      | ⭐⭐⭐⭐ | **선정**                |
| Locust  | ⭐⭐⭐   | 채용 요건에 명시됨      |
| Vegeta  | ⭐⭐⭐   | Go 프로젝트와 스택 일치 |
| Gatling | ⭐⭐     | Scala 러닝커브          |
| JMeter  | ⭐       | 레거시, GUI 무거움      |

### k6 선정 근거

1. **업계 표준** - 현재 가장 많이 사용되는 부하테스트 도구
2. **Grafana 연동** - 프로젝트에 이미 Prometheus + Grafana 모니터링 구축됨, 같은 회사 제품이라 연동 최적화
3. **JavaScript 스크립트** - 프론트엔드(React) 개발자도 쉽게 작성 가능
4. **K8s 친화적** - k6-operator로 클러스터 내 분산 부하테스트 가능
5. **현대적 설계** - CLI 기반, CI/CD 파이프라인 통합 용이

## 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│                        EKS Cluster                               │
│                                                                  │
│  ┌────────────────────────┐    ┌────────────────────────┐       │
│  │   일반 노드 그룹        │    │  Loadtest 노드 그룹     │       │
│  │   (tunelink-dev-node)  │    │  (tunelink-dev-loadtest)│       │
│  │                        │    │  taint: role=loadtest   │       │
│  │  ┌─────┐  ┌─────┐     │    │                        │       │
│  │  │ API │  │ Web │     │    │  ┌──────────────────┐  │       │
│  │  └──┬──┘  └─────┘     │    │  │  k6 Runner Pods  │  │       │
│  │     │                 │    │  │  (nodeSelector:  │  │       │
│  │     │ HTTP            │    │  │   role=loadtest) │  │       │
│  │     │                 │    │  └────────┬─────────┘  │       │
│  └─────┼─────────────────┘    └───────────┼────────────┘       │
│        │                                  │                     │
│        └──────────── K8s 내부 네트워크 ────┘                     │
│                                                                  │
│  ┌─────────────────────────┐                                    │
│  │  k6-operator-system     │  watches TestRun CRD               │
│  │  ┌───────────────────┐  │                                    │
│  │  │   k6-operator     │──┼──────────────────────────────────▶ │
│  │  └───────────────────┘  │                                    │
│  └─────────────────────────┘                                    │
└─────────────────────────────────────────────────────────────────┘
```

### 노드 격리

k6 Runner Pod는 **전용 loadtest 노드**에서 실행됩니다. 이를 통해:
- API Pod와 리소스 경쟁 없음 → **정확한 성능 측정**
- 테스트 시에만 노드 활성화 → **비용 절약**

자세한 내용은 [loadtest-node.md](./loadtest-node.md) 참조.

### 컴포넌트

| 컴포넌트                  | Namespace            | 생명주기                        |
| ------------------------- | -------------------- | ------------------------------- |
| k6-operator               | `k6-operator-system` | 항상 실행 (가벼움, ~64Mi)       |
| k6-test-scripts ConfigMap | `tunelink`           | 항상 존재                       |
| k6 Runner Pods            | `tunelink`           | 테스트 시작 → 완료 후 자동 삭제 |

---

## 테스트 대상 API

| Method | Path           | 설명          | Request Body               |
| ------ | -------------- | ------------- | -------------------------- |
| GET    | `/health`      | 헬스체크      | -                          |
| POST   | `/api/urls`    | URL 단축 생성 | `{ "originalUrl": "..." }` |
| GET    | `/r/:shortUrl` | 리다이렉트    | -                          |

**K8s 내부 서비스 주소:**

```
http://tunelink-dev-api.tunelink.svc.cluster.local
```

---

## 테스트 스크립트

경로: `infra/modules/k6_operator/scripts/`

### smoke-test.js

- **목적**: 기본 동작 확인
- **부하**: 5 VUs, 30초
- **시나리오**: Health check만
- **Threshold**: p(95) < 500ms, 에러율 < 1%

### load-test.js

- **목적**: 일반적인 부하 상황 테스트
- **부하**: 20 → 50 VUs, 7분
- **시나리오**: Health → URL 생성 → 리다이렉트
- **Threshold**: p(95) < 1000ms, 에러율 < 5%

### stress-test.js

- **목적**: 시스템 한계 테스트
- **부하**: 100 → 200 VUs, 16분
- **시나리오**: Health → URL 생성 → 리다이렉트
- **Threshold**: 없음 (한계 측정 목적)

### breakpoint-test.js

- **목적**: 시스템 한계점(Breaking Point) 탐색
- **부하**: 10 → 500 RPS (점진 증가), 최대 1000 VUs
- **시나리오**: Health → URL 생성 → 리다이렉트
- **Threshold**: 에러율 < 15%, p(95) < 10초 (초과 시 자동 중단)
- **특징**: `ramping-arrival-rate` executor 사용, 일정 RPS 유지하며 부하 증가

---

## 테스트 실행 방법

### Claude Code 스킬 사용 (권장)

```bash
# Claude Code에서 간편하게 실행
/k6-load-test
```

스킬이 테스트 유형 선택 → 실행 → 결과 문서화 → 리소스 정리까지 자동으로 처리합니다.

---

### 수동 실행

### 0. Loadtest 노드 활성화 (테스트 전 필수)

```bash
# infra/main.tf에서 loadtest_node_desired_size = 1 로 변경 후
cd /Users/joseonghyeon/tunelink/infra
terraform apply -target=module.eks

# 노드 Ready 확인 (1-2분 소요)
kubectl get nodes -l role=loadtest
```

### 1. k6-operator 설치 (최초 1회)

```bash
cd /Users/joseonghyeon/tunelink/infra
terraform apply
```

### 2. 설치 확인

```bash
# k6-operator 확인
kubectl get deployment -n k6-operator-system

# 테스트 스크립트 ConfigMap 확인
kubectl get configmap k6-test-scripts -n tunelink
```

### 3. 테스트 실행

> **참고**: 모든 테스트는 loadtest 노드에서 실행됩니다 (nodeSelector + toleration 필수)

```bash
# Smoke Test 실행
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

# Load Test 실행
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

# Stress Test 실행
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

# Breakpoint Test 실행 (시스템 한계점 탐색)
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

### 4. 테스트 모니터링

```bash
# 테스트 상태 확인
kubectl get testrun -n tunelink

# 실시간 로그
kubectl logs -n tunelink -l app=k6 -f

# Runner Pod 확인
kubectl get pods -n tunelink -l app=k6
```

### 5. 테스트 정리

```bash
# 완료된 테스트 목록 확인
kubectl get testrun -n tunelink

# 특정 테스트 삭제
kubectl delete testrun <테스트명> -n tunelink

# 모든 완료된 테스트 삭제
kubectl delete testrun --all -n tunelink
```

### 6. Loadtest 노드 끄기 (테스트 완료 후)

```bash
# infra/main.tf에서 loadtest_node_desired_size = 0 로 변경 후
cd /Users/joseonghyeon/tunelink/infra
terraform apply -target=module.eks
```

---

## Grafana 연동

k6는 Prometheus Remote Write를 지원하여 기존 모니터링 스택과 연동됨.

### 아키텍처
```
k6 Runner Pod → Prometheus Remote Write → Prometheus → Grafana
```

### 설정 (이미 적용됨)

**1. Prometheus Remote Write Receiver 활성화**
```hcl
# monitoring 모듈 main.tf
prometheus.prometheusSpec.enableRemoteWriteReceiver = true
```

**2. k6 TestRun에서 Prometheus로 메트릭 전송**
```yaml
# testruns/*.yaml
spec:
  arguments: --out experimental-prometheus-rw
  runner:
    env:
      - name: K6_PROMETHEUS_RW_SERVER_URL
        value: http://prometheus-kube-prometheus-prometheus.monitoring.svc.cluster.local:9090/api/v1/write
```

**3. Grafana k6 대시보드 자동 프로비저닝**
- **방식**: ConfigMap으로 커스텀 대시보드 배포
- **파일**: `infra/modules/monitoring/dashboards/k6-prometheus.json`
- 원본(gnetId: 19665)의 버그 수정 버전 (HTTP request failures 쿼리 수정)
- terraform apply 시 자동 설치됨

### 대시보드 수정 시
```bash
# 1. Grafana UI에서 대시보드 수정
# 2. Share > Export > Save to file

# 3. JSON 파일을 dashboards 폴더로 복사
cp ~/Downloads/k6-prometheus-*.json \
   infra/modules/monitoring/dashboards/k6-prometheus.json

# 4. Terraform 적용
cd infra && terraform apply -target=module.monitoring
```

---

## 진행 상황

- [x] k6 도구 선정 및 근거 문서화
- [x] k6-operator Terraform 모듈 작성
- [x] API 엔드포인트별 테스트 스크립트 작성 (smoke, load, stress, breakpoint)
- [x] k6-operator 배포 (terraform apply)
- [x] Smoke Test 실행 및 결과 확인
- [x] Breakpoint Test 실행 및 결과 확인
- [x] Grafana 대시보드 연동 (커스텀 대시보드, ConfigMap 관리)
- [x] 부하테스트 결과 문서화 ([test-result.md](./test-result.md))
- [x] Loadtest 노드 격리 구현 ([loadtest-node.md](./loadtest-node.md))
- [ ] Load Test 재실행 (격리된 환경)
- [ ] Stress Test 실행

## 관련 문서

- [테스트 결과](./test-result.md) - 실행된 테스트 결과 기록
- [노드 격리 설정](./loadtest-node.md) - Loadtest 전용 노드 그룹 설정
- [모니터링](./monitoring.md) - Prometheus + Grafana 설정
