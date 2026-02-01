# TuneLink 모니터링 방안

## 현재 인프라 현황
- **클라우드**: AWS (ap-northeast-2)
- **K8s**: EKS 1.29 (SPOT 인스턴스, 1~3 노드)
- **서비스**: API(2 replicas), Web(2 replicas)
- **DB**: RDS MySQL 8.0 (db.t3.micro)
- **모니터링**: Prometheus + Grafana (Terraform 모듈로 관리)

---

## 모니터링 옵션 비교

### 옵션 1: AWS CloudWatch Container Insights (빠른 시작)

**장점**
- AWS 네이티브, 설정 간단
- EKS와 즉시 통합
- 추가 인프라 불필요

**단점**
- 비용 발생 (로그/메트릭 수집량 기반)
- 커스터마이징 제한적

**설정 방법**
```bash
# CloudWatch 에이전트 설치
aws eks create-addon \
  --cluster-name tunelink-dev \
  --addon-name amazon-cloudwatch-observability \
  --region ap-northeast-2
```

**수집 가능한 지표**
- Pod CPU/Memory 사용량
- 노드 리소스 사용량
- 컨테이너 재시작 횟수
- 네트워크 I/O

**예상 비용**: 월 $10~30 (dev 환경 규모)

---

### 옵션 2: Prometheus + Grafana (추천 - 가성비)

**장점**
- 오픈소스, 무료
- 커스터마이징 자유로움
- K8s 표준 모니터링 스택
- 풍부한 커뮤니티 대시보드

**단점**
- 클러스터 내 리소스 소비
- 초기 설정 필요

**설치 방법 (Helm)**
```bash
# kube-prometheus-stack 설치 (Prometheus + Grafana + AlertManager)
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.retention=7d \
  --set prometheus.prometheusSpec.resources.requests.memory=256Mi \
  --set prometheus.prometheusSpec.resources.requests.cpu=100m \
  --set grafana.adminPassword=your-secure-password
```

**Terraform 모듈 예시**
```hcl
# modules/monitoring/main.tf
resource "helm_release" "prometheus" {
  name             = "prometheus"
  repository       = "https://prometheus-community.github.io/helm-charts"
  chart            = "kube-prometheus-stack"
  namespace        = "monitoring"
  create_namespace = true

  values = [
    file("${path.module}/values.yaml")
  ]
}
```

**주요 대시보드**
- Node Exporter (노드 리소스)
- Kubernetes Pods (Pod 상태/리소스)
- CoreDNS (DNS 성능)
- 커스텀 API 대시보드

**예상 리소스**: CPU 200m~500m, Memory 512Mi~1Gi

---

### 옵션 3: AWS Managed Prometheus + Grafana

**장점**
- 관리형 서비스 (운영 부담 없음)
- 스케일 자동 처리
- HA 기본 제공

**단점**
- 비용 높음
- 오버스펙 (dev 환경에는)

**예상 비용**: 월 $50~100+

---

### 옵션 4: 경량 솔루션 (Metrics Server만)

**장점**
- 최소 리소스
- kubectl top 명령어 사용 가능
- HPA 지원

**단점**
- 히스토리 저장 안 됨
- 대시보드 없음

**설치**
```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

---

## 추천 조합 (Dev 환경)

### Phase 1: 즉시 적용
1. **Metrics Server** 설치 → `kubectl top pods/nodes` 사용
2. **RDS CloudWatch 알람** 설정 (CPU > 80%, Storage < 20%)

### Phase 2: 기본 모니터링
1. **Prometheus + Grafana** 설치 (kube-prometheus-stack)
2. 기본 대시보드로 클러스터/Pod 모니터링

### Phase 3: 고급 모니터링 (선택)
1. 애플리케이션 메트릭 수집 (API response time, error rate)
2. 로그 수집 (Loki 또는 CloudWatch Logs)
3. 알람 설정 (Slack/Discord 연동)

---

## 확인하고 싶은 핵심 지표

### Pod/Container
| 지표 | 설명 | 중요도 |
|------|------|--------|
| CPU 사용량 | 리소스 적정성 확인 | 높음 |
| Memory 사용량 | OOM 예방 | 높음 |
| Restart 횟수 | 안정성 지표 | 높음 |
| Pod 상태 | Running/Pending/Failed | 높음 |

### Node
| 지표 | 설명 | 중요도 |
|------|------|--------|
| Node CPU/Memory | 노드 용량 계획 | 중간 |
| Disk 사용량 | 스토리지 이슈 예방 | 중간 |

### Application (API 서버)
| 지표 | 설명 | 중요도 |
|------|------|--------|
| Request latency (p50, p95, p99) | 응답 속도 | 높음 |
| Error rate (4xx, 5xx) | 서비스 품질 | 높음 |
| Request per second | 트래픽 패턴 | 중간 |

### Database (RDS)
| 지표 | 설명 | 중요도 |
|------|------|--------|
| CPU Utilization | 쿼리 부하 | 높음 |
| Database Connections | 커넥션 풀 관리 | 중간 |
| Free Storage Space | 스토리지 관리 | 중간 |

---

## 빠른 시작: kubectl 명령어

Metrics Server 설치 후 바로 사용 가능:

```bash
# Pod 리소스 확인
kubectl top pods -n tunelink

# 노드 리소스 확인
kubectl top nodes

# Pod 상태 확인
kubectl get pods -n tunelink -o wide

# Pod 로그 확인
kubectl logs -n tunelink -l app=tunelink-api --tail=100

# Pod 상세 정보 (이벤트 포함)
kubectl describe pod -n tunelink -l app=tunelink-api
```

---

## Terraform으로 Prometheus + Grafana 설치

모니터링 모듈이 이미 구성되어 있습니다.

### 1. 적용
```bash
cd infra/terraform/envs/dev

# Plan 확인
terraform plan

# 적용
terraform apply
```

### 2. Grafana 접속
```bash
# port-forward로 로컬에서 접속
kubectl port-forward svc/prometheus-grafana -n monitoring 3000:80

# 브라우저에서 http://localhost:3000 접속
# - Username: admin
# - Password: terraform.tfvars의 grafana_admin_password 값
```

### 3. Prometheus 접속
```bash
kubectl port-forward svc/prometheus-kube-prometheus-prometheus -n monitoring 9090:9090

# http://localhost:9090 접속
```

### 4. 기본 제공 대시보드
Grafana에 로그인하면 아래 대시보드가 자동 설치됨:
- **Kubernetes / Compute Resources / Cluster** - 클러스터 전체 리소스
- **Kubernetes / Compute Resources / Pod** - Pod별 리소스
- **Kubernetes / Compute Resources / Namespace (Pods)** - 네임스페이스별 리소스
- **Node Exporter / Nodes** - 노드 시스템 메트릭

### 모듈 구조
```
infra/modules/monitoring/
├── main.tf       # Helm release + ConfigMap 정의
├── variables.tf  # 설정 변수
├── outputs.tf    # 출력값 (port-forward 명령어 등)
└── dashboards/
    └── k6-prometheus.json  # k6 대시보드 (커스텀, 버그 수정됨)
```

---

## 설치 완료 내역 (2026-01-23)

### 설치된 구성요소
```bash
# monitoring 네임스페이스에 설치됨
AWS_PROFILE=tunelink kubectl get pods -n monitoring
```

| Pod | 역할 |
|-----|------|
| prometheus-grafana-* | Grafana 대시보드 |
| prometheus-kube-prometheus-prometheus-* | Prometheus 서버 |
| prometheus-kube-prometheus-operator-* | Prometheus Operator |
| prometheus-kube-state-metrics-* | K8s 상태 메트릭 수집 |
| prometheus-prometheus-node-exporter-* | 노드 메트릭 수집 |
| alertmanager-prometheus-kube-prometheus-alertmanager-* | 알람 매니저 |

### Grafana 접속 정보
- **URL**: http://grafana.hearttune.link
- **Username**: admin
- **Password**: terraform.tfvars의 `grafana_admin_password` 값

### 설치 방법 (Helm)
```bash
# 1. Helm repo 추가
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# 2. 설치
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.retention=7d \
  --set grafana.adminPassword=<password>
```

### Ingress 설정
```bash
# Grafana Ingress 생성 (ALB)
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tunelink-dev-grafana-ingress
  namespace: monitoring
  annotations:
    kubernetes.io/ingress.class: alb
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/healthcheck-path: /api/health
EOF
```

### Route53 설정
```bash
# grafana.hearttune.link → ALB 연결 (Alias A 레코드)
# Hosted Zone: Z0012757YHB5B2GHM6YO (hearttune.link)
aws route53 change-resource-record-sets \
  --hosted-zone-id Z0012757YHB5B2GHM6YO \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "grafana.hearttune.link",
        "Type": "A",
        "AliasTarget": {
          "HostedZoneId": "ZWKZPGTI48KDX",
          "DNSName": "<ALB DNS Name>",
          "EvaluateTargetHealth": true
        }
      }
    }]
  }'
```

### 추천 대시보드
| 대시보드 | 용도 |
|---------|------|
| Kubernetes / Compute Resources / Cluster | 클러스터 전체 CPU/메모리 |
| Kubernetes / Compute Resources / Node | 노드별 CPU/메모리 |
| Kubernetes / Compute Resources / Pod | Pod별 리소스 |
| Node Exporter / Nodes | EC2 노드 상세 (CPU, 디스크, 네트워크) |
| k6 Load Testing / k6-prometheus | k6 부하테스트 결과 시각화 (커스텀 대시보드) |

### k6 부하테스트 연동 (2026-01-29 추가)

k6 테스트 결과를 Grafana에서 시각화하기 위한 설정이 완료됨.

**아키텍처**
```
k6 Runner Pod → Prometheus Remote Write → Prometheus → Grafana
```

**설정 내용**
1. Prometheus: `enableRemoteWriteReceiver = true`
2. k6 TestRun: `--out experimental-prometheus-rw` 옵션으로 메트릭 전송
3. Grafana: 커스텀 대시보드 ConfigMap으로 자동 프로비저닝
   - 원본(gnetId: 19665)의 버그 수정 버전 사용
   - 파일: `infra/modules/monitoring/dashboards/k6-prometheus.json`

---

## 다음 단계

1. [x] Prometheus + Grafana Terraform 모듈 추가
2. [x] Helm으로 kube-prometheus-stack 설치
3. [x] Grafana Ingress 생성 (ALB)
4. [x] Route53 DNS 등록 (grafana.hearttune.link)
5. [x] Grafana 접속 확인
6. [x] k6 부하테스트 대시보드 연동 (커스텀 대시보드, ConfigMap 관리)
7. [ ] (선택) 애플리케이션 메트릭 endpoint 추가 (/metrics)
8. [ ] (선택) Slack/Discord 알람 연동
