# Load Testing

## Tool Selection: k6

### Candidate Comparison

| Tool    | Rating   | Notes                              |
| ------- | -------- | ---------------------------------- |
| k6      | ⭐⭐⭐⭐ | **Selected**                       |
| Locust  | ⭐⭐⭐   | Python-based, provides Web UI      |
| Vegeta  | ⭐⭐⭐   | Stack matches Go project           |
| Gatling | ⭐⭐     | Scala learning curve               |
| JMeter  | ⭐       | Legacy, heavy GUI                  |

### Rationale for Selecting k6

1. **Industry Standard** - Currently the most widely used load testing tool
2. **Grafana Integration** - Prometheus + Grafana monitoring already set up in the project; same company's product ensures optimal integration
3. **JavaScript Scripts** - Frontend (React) developers can easily write test scripts
4. **K8s Friendly** - Distributed load testing within the cluster via k6-operator
5. **Modern Design** - CLI-based, easy CI/CD pipeline integration

## Architecture

```
                          ┌─────────────────┐
                          │   Internet      │
                          │ hearttune.link  │
                          └────────┬────────┘
                                   │ HTTPS
                                   ▼
┌──────────────────────────────────────────────────────────────────┐
│                         EKS Cluster                              │
│                                                                  │
│                    ┌─────────────────┐                           │
│                    │  Ingress / ALB  │                           │
│                    └────────┬────────┘                           │
│                             │                                    │
│  ┌────────────────────────┐ │  ┌────────────────────────┐       │
│  │   General Node Group   │ │  │  Loadtest Node Group    │       │
│  │   (tunelink-dev-node)  │ │  │  (tunelink-dev-loadtest)│       │
│  │                        │ │  │  taint: role=loadtest   │       │
│  │  ┌─────┐  ┌─────┐     │ │  │                        │       │
│  │  │ API │  │ Web │     │ │  │  ┌──────────────────┐  │       │
│  │  └──┬──┘  └─────┘     │ │  │  │  k6 Runner Pods  │  │       │
│  │     ▲                 │ │  │  │  (nodeSelector:  │  │       │
│  │     │                 │ │  │  │   role=loadtest) │  │       │
│  │     │                 │ │  │  └────────┬─────────┘  │       │
│  └─────┼─────────────────┘ │  └───────────┼────────────┘       │
│        │                   │              │                     │
│        └───────────────────┘              │ External path       │
│                                           │ (hearttune.link)    │
│                                           ▼                     │
│  ┌─────────────────────────┐       ┌──────────────┐            │
│  │  k6-operator-system     │       │   Internet   │            │
│  │  ┌───────────────────┐  │       └──────────────┘            │
│  │  │   k6-operator     │──┼─── watches TestRun CRD ──────────▶│
│  │  └───────────────────┘  │                                    │
│  └─────────────────────────┘                                    │
└──────────────────────────────────────────────────────────────────┘
```

> **Note**: k6 tests through the external URL (`https://hearttune.link`), so the load test follows the same path as real users (Ingress/ALB → API).

### Node Isolation

k6 Runner Pods run on **dedicated loadtest nodes**. This ensures:
- No resource contention with API Pods → **Accurate performance measurement**
- Nodes activated only during tests → **Cost savings**

For details, see [loadtest-node.md](./loadtest-node.md).

### Components

| Component                 | Namespace            | Lifecycle                                       |
| ------------------------- | -------------------- | ----------------------------------------------- |
| k6-operator               | `k6-operator-system` | Always running (lightweight, ~64Mi)             |
| k6-test-scripts ConfigMap | `tunelink`           | Always present                                  |
| k6 Runner Pods            | `tunelink`           | Created at test start → auto-deleted on completion |

---

## Target APIs

| Method | Path           | Description       | Request Body               |
| ------ | -------------- | ----------------- | -------------------------- |
| GET    | `/health`      | Health check      | -                          |
| POST   | `/api/urls`    | Create short URL  | `{ "originalUrl": "..." }` |
| GET    | `/r/:shortUrl` | Redirect          | -                          |

**External Service URL:**

```
https://hearttune.link
```

---

## Test Scripts

Path: `infra/modules/k6_operator/scripts/`

### Common Design Principles

All tests use the **per-vu-iterations** executor to guarantee an exact number of requests:
- Each VU executes a fixed number of iterations
- Each iteration creates 1 URL + N redirects
- **Count-based** testing (not time-based) for predictable results

### smoke-test.js

- **Purpose**: Verify basic functionality
- **Executor**: `per-vu-iterations`
- **Load**: 5 VUs × 1 iteration = **5 URLs**
- **Scenario**: Health → URL creation → 10 redirects
- **Expected Result**: 5 URLs, 10 clicks each
- **Threshold**: p(95) < 500ms, error rate < 1%

### load-test.js

- **Purpose**: Test under normal load conditions
- **Executor**: `per-vu-iterations`
- **Load**: 100 VUs × 3 iterations = **300 URLs**
- **Scenario**: Health → URL creation → 100 redirects
- **Expected Result**: 300 URLs, 100 clicks each (30,000 total redirect requests)
- **Threshold**: p(95) < 1000ms, error rate < 5%

### stress-test.js

- **Purpose**: Test stability under high load
- **Executor**: `per-vu-iterations`
- **Load**: 1000 VUs × 3 iterations = **3,000 URLs**
- **Scenario**: Health → URL creation → 100 redirects
- **Expected Result**: 3,000 URLs, 100 clicks each (300,000 total redirect requests)
- **Threshold**: None (purpose is to measure limits)

### breakpoint-test.js

- **Purpose**: Find the system's breaking point
- **Executor**: `ramping-arrival-rate` (RPS-based)
- **Load**: 10 → 500 RPS (gradual increase), max 1000 VUs
- **Scenario**: Health → URL creation → 100 redirects (repeated)
- **Threshold**: Error rate < 15%, p(95) < 10s (auto-abort if exceeded)
- **Characteristics**: Maintains constant RPS while increasing load, auto-aborts upon reaching the breaking point

---

## How to Run Tests

### Using Claude Code Skill (Recommended)

```bash
# Run conveniently from Claude Code
/k6-load-test
```

The skill automatically handles test type selection → execution → result documentation → resource cleanup.

---

### Manual Execution

### 0. Activate Loadtest Node (Required Before Testing)

```bash
# Change loadtest_node_desired_size = 1 in infra/main.tf, then
cd /Users/joseonghyeon/tunelink/infra
terraform apply -target=module.eks

# Verify node is Ready (takes 1-2 minutes)
kubectl get nodes -l role=loadtest
```

### 1. Install k6-operator (One-Time Setup)

```bash
cd /Users/joseonghyeon/tunelink/infra
terraform apply
```

### 2. Verify Installation

```bash
# Check k6-operator
kubectl get deployment -n k6-operator-system

# Check test scripts ConfigMap
kubectl get configmap k6-test-scripts -n tunelink
```

### 3. Run Tests

> **Note**: All tests run on the loadtest node (nodeSelector + toleration required)

```bash
# Run Smoke Test
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

# Run Load Test
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

# Run Stress Test
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

# Run Breakpoint Test (Find System Breaking Point)
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

### 4. Monitor Tests

```bash
# Check test status
kubectl get testrun -n tunelink

# Real-time logs
kubectl logs -n tunelink -l app=k6 -f

# Check Runner Pods
kubectl get pods -n tunelink -l app=k6
```

### 5. Clean Up Tests

```bash
# List completed tests
kubectl get testrun -n tunelink

# Delete a specific test
kubectl delete testrun <test-name> -n tunelink

# Delete all completed tests
kubectl delete testrun --all -n tunelink
```

### 6. Deactivate Loadtest Node (After Testing)

```bash
# Change loadtest_node_desired_size = 0 in infra/main.tf, then
cd /Users/joseonghyeon/tunelink/infra
terraform apply -target=module.eks
```

---

## Grafana Integration

k6 supports Prometheus Remote Write, enabling integration with the existing monitoring stack.

### Architecture
```
k6 Runner Pod → Prometheus Remote Write → Prometheus → Grafana
```

### Configuration (Already Applied)

**1. Enable Prometheus Remote Write Receiver**
```hcl
# monitoring module main.tf
prometheus.prometheusSpec.enableRemoteWriteReceiver = true
```

**2. Send Metrics from k6 TestRun to Prometheus**
```yaml
# testruns/*.yaml
spec:
  arguments: --out experimental-prometheus-rw
  runner:
    env:
      - name: K6_PROMETHEUS_RW_SERVER_URL
        value: http://prometheus-kube-prometheus-prometheus.monitoring.svc.cluster.local:9090/api/v1/write
```

**3. Auto-Provisioned Grafana k6 Dashboard**
- **Method**: Deploy custom dashboard via ConfigMap
- **File**: `infra/modules/monitoring/dashboards/k6-prometheus.json`
- Bug-fixed version of the original (gnetId: 19665) (HTTP request failures query fixed)
- Automatically installed on terraform apply

### Modifying the Dashboard
```bash
# 1. Edit the dashboard in Grafana UI
# 2. Share > Export > Save to file

# 3. Copy JSON file to dashboards folder
cp ~/Downloads/k6-prometheus-*.json \
   infra/modules/monitoring/dashboards/k6-prometheus.json

# 4. Apply with Terraform
cd infra && terraform apply -target=module.monitoring
```

---

## Progress

- [x] k6 tool selection and rationale documentation
- [x] k6-operator Terraform module creation
- [x] Test scripts for each API endpoint (smoke, load, stress, breakpoint)
- [x] k6-operator deployment (terraform apply)
- [x] Smoke Test execution and result verification
- [x] Breakpoint Test execution and result verification
- [x] Grafana dashboard integration (custom dashboard, ConfigMap management)
- [x] Load test result documentation ([test-result.md](./test-result.md))
- [x] Loadtest node isolation implementation ([loadtest-node.md](./loadtest-node.md))
- [x] Stress Test execution (2026-02-03) - **100% success, click count accuracy 100%**

## Related Documents

- [Test Results](./test-result.md) - Recorded results of executed tests
- [Node Isolation Setup](./loadtest-node.md) - Loadtest dedicated node group configuration
- [Monitoring](./monitoring.md) - Prometheus + Grafana configuration
