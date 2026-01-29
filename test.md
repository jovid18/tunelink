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
┌─────────────────────────────────────────────────────────────┐
│                      EKS Cluster                            │
│                                                             │
│  ┌─────────────────────────┐    ┌────────────────────────┐  │
│  │  k6-operator-system     │    │    tunelink namespace  │  │
│  │  ┌───────────────────┐  │    │                        │  │
│  │  │   k6-operator     │  │    │  ┌─────┐    ┌─────┐    │  │
│  │  │   (controller)    │  │    │  │ API │    │ Web │    │  │
│  │  └─────────┬─────────┘  │    │  └──▲──┘    └─────┘    │  │
│  └────────────┼────────────┘    │     │                  │  │
│               │                 │     │ HTTP             │  │
│               │ watches         │     │                  │  │
│               ▼                 │  ┌──┴───────────────┐  │  │
│  ┌─────────────────────────────┐│  │  k6 Runner Pods  │  │  │
│  │       TestRun CRD           │├──│  (테스트 시 생성) │  │  │
│  └─────────────────────────────┘│  └──────────────────┘  │  │
│                                 └────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

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

경로: `infra/terraform/modules/k6_operator/scripts/`

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

---

## 테스트 실행 방법

### 1. k6-operator 설치 (최초 1회)

```bash
cd /Users/joseonghyeon/tunelink/infra/terraform/envs/dev
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

```bash
# Smoke Test 실행
kubectl apply -f - <<EOF
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: smoke-test
  namespace: tunelink
spec:
  parallelism: 1
  script:
    configMap:
      name: k6-test-scripts
      file: smoke-test.js
EOF

# Load Test 실행
kubectl apply -f - <<EOF
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: load-test
  namespace: tunelink
spec:
  parallelism: 2
  script:
    configMap:
      name: k6-test-scripts
      file: load-test.js
EOF

# Stress Test 실행
kubectl apply -f - <<EOF
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: stress-test
  namespace: tunelink
spec:
  parallelism: 4
  script:
    configMap:
      name: k6-test-scripts
      file: stress-test.js
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
# 완료된 테스트 삭제
kubectl delete testrun smoke-test -n tunelink
kubectl delete testrun load-test -n tunelink
kubectl delete testrun stress-test -n tunelink
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
- **Dashboard ID**: 19665 (k6 Prometheus)
- **폴더**: "k6 Load Testing"
- terraform apply 시 자동 설치됨

### 수동 대시보드 Import (필요시)
```bash
# Grafana 접속 (port-forward)
kubectl port-forward svc/prometheus-grafana -n monitoring 3000:80

# 브라우저에서 http://localhost:3000 접속
# Dashboard > Import > ID: 19665
```

---

## 진행 상황

- [x] k6 도구 선정 및 근거 문서화
- [x] k6-operator Terraform 모듈 작성
- [x] API 엔드포인트별 테스트 스크립트 작성
- [x] k6-operator 배포 (terraform apply)
- [x] Smoke Test 실행 및 결과 확인
- [ ] Load Test 실행 및 결과 확인
- [x] Grafana 대시보드 연동 (Dashboard ID: 19665)
- [ ] 부하테스트 결과 문서화

### Smoke Test 결과 (2026-01-29)

| 항목 | 결과 |
|------|------|
| VUs | 5 |
| Duration | 30s |
| Total Requests | 150 |
| Success Rate | 100% |
| p(95) Latency | 1.81ms |
| Error Rate | 0.00% |
| Threshold | 모두 통과 |
