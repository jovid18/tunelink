# TuneLink Monitoring Strategy

## Current Infrastructure Overview
- **Cloud**: AWS (ap-northeast-2)
- **K8s**: EKS 1.33 (SPOT instances, 2~4 nodes)
- **Services**: API (2 replicas), Web (2 replicas)
- **DB**: RDS MySQL 8.0 (db.t3.micro)
- **Monitoring**: Prometheus + Grafana (managed via Terraform modules)

---

## Monitoring Options Comparison

### Option 1: AWS CloudWatch Container Insights (Quick Start)

**Pros**
- AWS native, simple setup
- Immediate integration with EKS
- No additional infrastructure required

**Cons**
- Incurs costs (based on log/metric ingestion volume)
- Limited customization

**Setup**
```bash
# Install CloudWatch agent
aws eks create-addon \
  --cluster-name tunelink-dev \
  --addon-name amazon-cloudwatch-observability \
  --region ap-northeast-2
```

**Available Metrics**
- Pod CPU/Memory usage
- Node resource usage
- Container restart count
- Network I/O

**Estimated Cost**: ~$10~30/month (dev environment scale)

---

### Option 2: Prometheus + Grafana (Recommended - Best Value)

**Pros**
- Open source, free
- Highly customizable
- Standard K8s monitoring stack
- Rich community dashboards

**Cons**
- Consumes cluster resources
- Requires initial setup

**Installation (Helm)**
```bash
# Install kube-prometheus-stack (Prometheus + Grafana + AlertManager)
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

**Terraform Module Example**
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

**Key Dashboards**
- Node Exporter (node resources)
- Kubernetes Pods (Pod status/resources)
- CoreDNS (DNS performance)
- Custom API dashboard

**Estimated Resources**: CPU 200m~500m, Memory 512Mi~1Gi

---

### Option 3: AWS Managed Prometheus + Grafana

**Pros**
- Managed service (no operational overhead)
- Auto-scaling
- HA built-in

**Cons**
- Higher cost
- Overkill (for dev environment)

**Estimated Cost**: ~$50~100+/month

---

### Option 4: Lightweight Solution (Metrics Server Only)

**Pros**
- Minimal resources
- Enables kubectl top command
- HPA support

**Cons**
- No history retention
- No dashboard

**Installation**
```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

---

## Recommended Combination (Dev Environment)

### Phase 1: Immediate
1. Install **Metrics Server** -> use `kubectl top pods/nodes`
2. Set up **RDS CloudWatch alarms** (CPU > 80%, Storage < 20%)

### Phase 2: Basic Monitoring
1. Install **Prometheus + Grafana** (kube-prometheus-stack)
2. Monitor cluster/Pods with default dashboards

### Phase 3: Advanced Monitoring (Optional)
1. Collect application metrics (API response time, error rate)
2. Log aggregation (Loki or CloudWatch Logs)
3. Alert configuration (Slack/Discord integration)

---

## Key Metrics to Monitor

### Pod/Container
| Metric | Description | Priority |
|--------|-------------|----------|
| CPU Usage | Verify resource adequacy | High |
| Memory Usage | Prevent OOM | High |
| Restart Count | Stability indicator | High |
| Pod Status | Running/Pending/Failed | High |

### Node
| Metric | Description | Priority |
|--------|-------------|----------|
| Node CPU/Memory | Node capacity planning | Medium |
| Disk Usage | Prevent storage issues | Medium |

### Application (API Server)
| Metric | Description | Priority |
|--------|-------------|----------|
| Request latency (p50, p95, p99) | Response speed | High |
| Error rate (4xx, 5xx) | Service quality | High |
| Request per second | Traffic patterns | Medium |

### Database (RDS)
| Metric | Description | Priority |
|--------|-------------|----------|
| CPU Utilization | Query load | High |
| Database Connections | Connection pool management | Medium |
| Free Storage Space | Storage management | Medium |

---

## Quick Start: kubectl Commands

Available immediately after installing Metrics Server:

```bash
# Check Pod resources
kubectl top pods -n tunelink

# Check node resources
kubectl top nodes

# Check Pod status
kubectl get pods -n tunelink -o wide

# Check Pod logs
kubectl logs -n tunelink -l app=tunelink-api --tail=100

# Pod details (including events)
kubectl describe pod -n tunelink -l app=tunelink-api
```

---

## Installing Prometheus + Grafana with Terraform

The monitoring module is already configured.

### 1. Apply
```bash
cd infra/terraform/envs/dev

# Review plan
terraform plan

# Apply
terraform apply
```

### 2. Access Grafana
```bash
# Access locally via port-forward
kubectl port-forward svc/prometheus-grafana -n monitoring 3000:80

# Open http://localhost:3000 in browser
# - Username: admin
# - Password: grafana_admin_password value from terraform.tfvars
```

### 3. Access Prometheus
```bash
kubectl port-forward svc/prometheus-kube-prometheus-prometheus -n monitoring 9090:9090

# Open http://localhost:9090
```

### 4. Pre-installed Dashboards
The following dashboards are automatically installed when you log in to Grafana:
- **Kubernetes / Compute Resources / Cluster** - Cluster-wide resources
- **Kubernetes / Compute Resources / Pod** - Per-Pod resources
- **Kubernetes / Compute Resources / Namespace (Pods)** - Per-namespace resources
- **Node Exporter / Nodes** - Node system metrics

### Module Structure
```
infra/modules/monitoring/
├── main.tf       # Helm release + ConfigMap definitions
├── variables.tf  # Configuration variables
├── outputs.tf    # Outputs (port-forward commands, etc.)
└── dashboards/
    └── k6-prometheus.json  # k6 dashboard (custom, bug-fixed)
```

---

## Installation Record (2026-01-23)

### Installed Components
```bash
# Installed in the monitoring namespace
AWS_PROFILE=tunelink kubectl get pods -n monitoring
```

| Pod | Role |
|-----|------|
| prometheus-grafana-* | Grafana dashboard |
| prometheus-kube-prometheus-prometheus-* | Prometheus server |
| prometheus-kube-prometheus-operator-* | Prometheus Operator |
| prometheus-kube-state-metrics-* | K8s state metrics collector |
| prometheus-prometheus-node-exporter-* | Node metrics collector |
| alertmanager-prometheus-kube-prometheus-alertmanager-* | Alert manager |

### Grafana Access Info
- **URL**: http://grafana.hearttune.link
- **Username**: admin
- **Password**: `grafana_admin_password` value from terraform.tfvars

### Installation Method (Helm)
```bash
# 1. Add Helm repo
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# 2. Install
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.retention=7d \
  --set grafana.adminPassword=<password>
```

### Ingress Configuration
```bash
# Create Grafana Ingress (ALB)
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

### Route53 Configuration
```bash
# grafana.hearttune.link -> ALB connection (Alias A record)
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

### Recommended Dashboards
| Dashboard | Purpose |
|-----------|---------|
| Kubernetes / Compute Resources / Cluster | Cluster-wide CPU/memory |
| Kubernetes / Compute Resources / Node | Per-node CPU/memory |
| Kubernetes / Compute Resources / Pod | Per-Pod resources |
| Node Exporter / Nodes | EC2 node details (CPU, disk, network) |
| k6 Load Testing / k6-prometheus | k6 load test result visualization (custom dashboard) |

### k6 Load Testing Integration (Added 2026-01-29)

Configuration for visualizing k6 test results in Grafana has been completed.

**Architecture**
```
k6 Runner Pod → Prometheus Remote Write → Prometheus → Grafana
```

**Configuration Details**
1. Prometheus: `enableRemoteWriteReceiver = true`
2. k6 TestRun: Sends metrics via `--out experimental-prometheus-rw` option
3. Grafana: Custom dashboard auto-provisioned via ConfigMap
   - Uses a bug-fixed version of the original (gnetId: 19665)
   - File: `infra/modules/monitoring/dashboards/k6-prometheus.json`

---

## Next Steps

1. [x] Add Prometheus + Grafana Terraform module
2. [x] Install kube-prometheus-stack via Helm
3. [x] Create Grafana Ingress (ALB)
4. [x] Register Route53 DNS (grafana.hearttune.link)
5. [x] Verify Grafana access
6. [x] Integrate k6 load testing dashboard (custom dashboard, ConfigMap managed)