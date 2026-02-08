# TuneLink

> URL Shortener Service - A high-performance URL shortening service built with Go + EKS + Terraform

![Demo](docs/images/tunelink.gif)

**Live Demo:** https://hearttune.link

---

## Highlights

| Metric | Result |
|--------|--------|
| Throughput | **6,500+ req/s** |
| Response Time (p95) | **109ms** |
| Concurrent Users | **1,000 VUs** |
| Error Rate | **0%** |

**93.5% response time improvement** with Redis cache (687ms -> 45ms)

---

## Tech Stack

**Backend**
- Go 1.21 + Gin Framework
- GORM (MySQL)
- Redis (Caching + Click Counter)

**Frontend**
- React 18 + TypeScript
- Vite + TailwindCSS

**Infrastructure**
- AWS EKS (Kubernetes)
- RDS MySQL + ElastiCache Redis
- ALB + ACM (HTTPS)
- Terraform (IaC)

**CI/CD & Monitoring**
- GitHub Actions
- Prometheus + Grafana
- k6 (Load Testing)

---

## Architecture

![AWS Architecture](docs/images/aws-vpc-resource-map.png)

```
                         Internet
                            │
                            ▼
                    ┌──────────────┐
                    │   Route 53   │
                    │ hearttune.link│
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │  ALB (HTTPS) │
                    └──────┬───────┘
                           │
        ┌──────────────────┴──────────────────┐
        │              EKS Cluster             │
        │                                      │
        │   ┌─────────┐      ┌─────────┐      │
        │   │   API   │      │   Web   │      │
        │   │  (Go)   │      │ (React) │      │
        │   └────┬────┘      └─────────┘      │
        │        │                            │
        └────────┼────────────────────────────┘
                 │
        ┌────────┴────────┐
        │                 │
   ┌────▼────┐      ┌─────▼─────┐
   │  MySQL  │      │   Redis   │
   │  (RDS)  │      │(ElastiCache)│
   └─────────┘      └───────────┘
```

---

## Key Features

### 1. k6 Load Testing Environment

Built a Kubernetes-native load testing environment.

```
┌─────────────────────────────────────────────────────────┐
│                      EKS Cluster                         │
│                                                          │
│  ┌────────────────────┐    ┌────────────────────────┐   │
│  │  General Node Group │    │  Loadtest Node Group    │   │
│  │                    │    │   (Isolated via Taint)  │   │
│  │  ┌─────┐ ┌─────┐  │    │  ┌──────────────────┐  │   │
│  │  │ API │ │ Web │  │    │  │  k6 Runner Pods  │  │   │
│  │  └─────┘ └─────┘  │    │  └────────┬─────────┘  │   │
│  └────────────────────┘    └───────────┼────────────┘   │
│                                        │                 │
│                                        ▼                 │
│                               https://hearttune.link     │
└─────────────────────────────────────────────────────────┘
```

**Implementation Details:**
- **k6-operator**: Test definitions via Kubernetes CRD -> GitOps friendly
- **Node Isolation**: Taint/Toleration separates k6 Pods from API Pods -> Accurate performance measurement
- **On-Demand Nodes**: Nodes activated only during tests -> Cost savings
- **External Path Testing**: Tests run through the same path as real users (ALB -> API)

---

### 2. Prometheus + Grafana Monitoring

![Grafana Dashboard](docs/images/stress-test-20260203-2243-grafana.png)

**Implementation Details:**
- **kube-prometheus-stack**: Integrated installation of Prometheus + Grafana + AlertManager via Helm
- **k6 Metrics Integration**: Real-time visualization of load test results via Prometheus Remote Write
- **Custom Dashboards**: Automatic dashboard provisioning via ConfigMap (managed by Terraform)
- **External Access**: ALB Ingress exposed at `grafana.hearttune.link`

---

### 3. Bastion Host (Private Resource Access)

![DataGrip SSH Tunnel](docs/images/datagrip-ssh-tunnel-bastion.png)

Configured a Bastion Host for secure access to RDS/Redis in Private Subnets.

**Implementation Details:**
- **Elastic IP**: Fixed IP maintained even after instance restart
- **SSH Tunnel**: Direct access to RDS/Redis via SSH Tunnel from DataGrip
- **Least Privilege**: Security Group allowing only Bastion -> RDS/Redis

![RDS Query](docs/images/datagrip-mysql-rds-query.png)

---

### 4. Redis Cache + Click Synchronization

Introduced a Redis caching strategy to reduce DB load and improve response speed.

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│    API      │────▶│   Redis     │
└─────────────┘     └──────┬──────┘     └──────┬──────┘
                           │                    │
                           │ cache miss         │ INCR clicks
                           ▼                    │
                    ┌─────────────┐             │
                    │    MySQL    │◀────────────┘
                    │    (RDS)    │    batch sync
                    └─────────────┘
```

**Implementation Details:**
- **URL Lookup Caching**: Cache-Aside pattern (TTL: 1h)
- **Click Counter**: Redis INCR (atomic) -> Background worker syncs to DB
- **Graceful Degradation**: Falls back to DB when Redis is unavailable

---

### 5. CI/CD Pipeline

Built an automated build/deploy pipeline with GitHub Actions.

```
Push to main
     │
     ├─── apps/api/** changed
     │         │
     │         ▼
     │    Go Test → Docker Build → ECR Push → kubectl rollout
     │
     └─── apps/web/** changed
               │
               ▼
          npm build → Docker Build → ECR Push → kubectl rollout
```

**Implementation Details:**
- **Path-Based Triggers**: Workflows run only when `apps/api/**` or `apps/web/**` are changed
- **Image Tagging**: Tagged with Git SHA for easy rollback
- **Zero-Downtime Deployment**: Rolling Update via `kubectl rollout restart`

---

## Performance Optimization

### Before and After Redis Cache

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Avg Response Time | 687ms | 45ms | **-93.5%** |
| p(95) Response Time | 2.47s | 109ms | **-95.6%** |
| Throughput | 1,187 req/s | 6,504 req/s | **+448%** |

### Problems Solved

| Problem | Cause | Solution |
|---------|-------|----------|
| Too many connections | Connection Pool not configured | Limited to 25 connections per Pod |
| 50% click count loss | Read-Modify-Write Race Condition | Atomic SQL update (`clicks = clicks + 1`) |
| Response delay under high load | DB query on every request | Redis caching + click batch synchronization |

---

## Quick Start

### Local Development Environment

```bash
# 1. Start infrastructure (MySQL, Redis)
docker-compose up -d

# 2. Run API server
cd apps/api
go run cmd/api/main.go

# 3. Run Web
cd apps/web
npm install && npm run dev
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| POST | `/api/urls` | Create shortened URL |
| GET | `/r/{shortUrl}` | Redirect |

---

## Project Structure

```
tunelink/
├── apps/
│   ├── api/                 # Go API server
│   │   ├── cmd/api/         # Entry point
│   │   └── internal/        # Business logic
│   └── web/                 # React frontend
│
├── infra/
│   └── terraform/
│       ├── modules/         # Reusable Terraform modules
│       │   ├── vpc/         # VPC, Subnet, NAT Gateway
│       │   ├── eks/         # EKS Cluster + Node Groups
│       │   ├── rds/         # MySQL (RDS)
│       │   ├── elasticache/ # Redis (ElastiCache)
│       │   ├── bastion/     # Bastion Host + Elastic IP
│       │   ├── monitoring/  # Prometheus + Grafana
│       │   ├── k6_operator/ # k6 Load Testing
│       │   └── ...
│       └── envs/dev/        # Environment-specific configuration
│
├── .github/workflows/       # CI/CD Pipeline
└── docs/                    # Project documentation
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [spec.md](docs/spec.md) | Project planning and design |
| [deploy.md](docs/deploy.md) | AWS deployment guide (end-to-end) |
| [monitoring.md](docs/monitoring.md) | Prometheus + Grafana setup |
| [test.md](docs/test.md) | k6 load testing strategy |
| [test-result.md](docs/test-result.md) | Load test results and improvement history |
| [troubleshooting.md](docs/troubleshooting.md) | Troubleshooting guide |

---

## Cost (Dev Environment)

![AWS Cost](docs/images/aws-cost-explorer.png)

| Resource | Monthly Cost |
|----------|-------------|
| EKS Control Plane | $73 |
| EC2 Nodes (Spot) | $10 |
| RDS (db.t3.micro) | $15 |
| ElastiCache | $12 |
| ALB | $20 |
| NAT Gateway | $32 |
| **Total** | **~$162/month** |

> **~70% cost savings** compared to On-Demand by using Spot Instances

---

## License

MIT
