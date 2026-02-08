# TuneLink - URL Shortener

## Service Overview

| Item      | Decision                             |
| --------- | ------------------------------------ |
| Service   | URL Shortener (Link shortening service) |
| Auth      | None (MVP stage)                     |
| Approach  | Local dev environment first → AWS deployment |

### Core Features (MVP)

- Long URL → Short URL generation
- Redirect to original URL when accessing short URL
- (Optional) Click count statistics

### API Endpoints

| Method | Path            | Description      |
| ------ | --------------- | ---------------- |
| GET    | `/health`       | Health check     |
| POST   | `/api/urls`     | Create short URL |
| GET    | `/r/{shortUrl}` | Redirect         |

### DB Schema

**Table: `urls`**
| Column | Type | Description |
|------|------|------|
| id | BIGINT | PK |
| short_url | VARCHAR(10) | Short code |
| original_url | TEXT | Original URL |
| clicks | BIGINT | Click count |
| created_at | TIMESTAMP | Created date |

---

## Tech Stack

| Area          | Technology            |
| ------------- | --------------------- |
| Backend       | Go + Gin              |
| Frontend      | React + Vite          |
| Database      | MySQL                 |
| Cache         | Redis                 |
| Container     | Docker                |
| Orchestration | Kubernetes (EKS)      |
| IaC           | Terraform (No Helm)   |
| CI/CD         | GitHub Actions        |

---

## Repo Structure (Monorepo)

```
tunelink/
├── apps/
│   ├── api/                     # Go + Gin API server
│   │   ├── cmd/
│   │   │   └── api/
│   │   │       └── main.go
│   │   ├── internal/
│   │   │   ├── handler/         # HTTP handlers
│   │   │   ├── service/         # Business logic
│   │   │   ├── repository/      # DB access
│   │   │   ├── model/           # Domain models
│   │   │   └── config/          # Configuration
│   │   ├── Dockerfile
│   │   ├── go.mod
│   │   └── go.sum
│   │
│   └── web/                     # React + Vite frontend
│       ├── src/
│       │   ├── components/
│       │   ├── pages/
│       │   ├── hooks/
│       │   ├── api/
│       │   └── main.tsx
│       ├── Dockerfile           # Serve static files with nginx
│       ├── nginx.conf
│       ├── package.json
│       └── vite.config.ts
│
├── infra/
│   └── terraform/
│       ├── envs/
│       │   ├── dev/
│       │   │   ├── main.tf
│       │   │   ├── variables.tf
│       │   │   ├── outputs.tf
│       │   │   └── terraform.tfvars
│       │   └── prod/
│       │       └── ...
│       │
│       └── modules/
│           ├── vpc/             # VPC, Subnet, IGW, NAT
│           ├── eks/             # EKS cluster
│           ├── rds/             # MySQL (RDS)
│           ├── elasticache/     # Redis (ElastiCache)
│           ├── ecr/             # Container registry
│           ├── alb_controller/  # AWS Load Balancer Controller
│           ├── k8s_base/        # namespace, configmap, secret
│           ├── k8s_api/         # api deployment, service
│           └── k8s_web/         # web deployment, service, ingress
│
├── .github/
│   └── workflows/
│       ├── api.yml              # Build/push/deploy on api changes
│       ├── web.yml              # Build/push/deploy on web changes
│       └── infra.yml            # Plan/apply on infra changes
│
├── scripts/
│   ├── local-dev.sh             # Local dev environment setup
│   └── init-tf.sh               # Terraform initialization
│
├── docs/
├── Makefile
├── docker-compose.yml           # For local development (MySQL, Redis)
└── README.md
```

---

## Development Flow

### Phase 1: Local Development Environment

```
[Developer Local]
    │
    ├── docker-compose up        # Run MySQL + Redis locally
    ├── apps/api → go run        # API server (localhost:8080)
    └── apps/web → npm run dev   # Vite dev server (localhost:5173)
```

**Goal**: API/Web feature development, DB schema design

---

### Phase 2: AWS Infrastructure Provisioning

```
[Terraform Apply Order]

1. VPC Module
   └── VPC, Public/Private Subnet, IGW, NAT Gateway

2. ECR Module
   └── api, web image repositories

3. RDS Module
   └── MySQL (Private Subnet)

4. ElastiCache Module
   └── Redis (Private Subnet)

5. EKS Module
   └── EKS Cluster + Node Group (Private Subnet)

6. ALB Controller Module
   └── AWS Load Balancer Controller (IRSA)

7. K8s Base Module
   └── Namespace, ConfigMap, Secret

8. K8s API/Web Module
   └── Deployment, Service, Ingress
```

**Dependency Graph:**

```
VPC ─┬─→ RDS
     ├─→ ElastiCache
     ├─→ EKS ─→ ALB Controller ─→ K8s Base ─┬─→ K8s API
     │                                       └─→ K8s Web
     └─→ ECR
```

---

### Phase 3: CI/CD Pipeline

#### On API Changes (apps/api/\*\*)

```
Push → GitHub Actions
         │
         ├── Go Build & Test
         ├── Docker Build
         ├── ECR Push (tag: commit SHA)
         └── Terraform Apply
             └── k8s_api module (update image_tag variable)
```

#### On Web Changes (apps/web/\*\*)

```
Push → GitHub Actions
         │
         ├── npm install & build
         ├── Docker Build (nginx)
         ├── ECR Push (tag: commit SHA)
         └── Terraform Apply
             └── k8s_web module (update image_tag variable)
```

#### On Infra Changes (infra/\*\*)

```
Push → GitHub Actions
         │
         ├── Terraform Plan (comment on PR)
         └── Terraform Apply (on merge to main)
```

---

## Network Architecture

```
                    ┌─────────────────────────────────────────────┐
                    │                    VPC                       │
                    │                                              │
   Internet ───────►│  ┌──────────────────────────────────────┐   │
                    │  │         Public Subnet                 │   │
                    │  │  ┌─────────────┐                      │   │
                    │  │  │ ALB (Ingress)│                     │   │
                    │  │  └──────┬──────┘                      │   │
                    │  └─────────┼────────────────────────────┘   │
                    │            │                                 │
                    │  ┌─────────▼────────────────────────────┐   │
                    │  │         Private Subnet                │   │
                    │  │                                       │   │
                    │  │  ┌─────────┐      ┌─────────┐         │   │
                    │  │  │   API   │◄────►│   Web   │         │   │
                    │  │  │  (Pod)  │      │  (Pod)  │         │   │
                    │  │  └────┬────┘      └─────────┘         │   │
                    │  │       │                               │   │
                    │  │  ┌────▼────┐      ┌─────────┐         │   │
                    │  │  │  MySQL  │      │  Redis  │         │   │
                    │  │  │  (RDS)  │      │(ElastiC)│         │   │
                    │  │  └─────────┘      └─────────┘         │   │
                    │  └───────────────────────────────────────┘   │
                    └─────────────────────────────────────────────┘
```

---

## Ingress Routing (Path-based)

```
example.com/          → Web Service (React)
example.com/api/*     → API Service (Go)
```

**Terraform Resource Example:**

```hcl
resource "kubernetes_ingress_v1" "main" {
  metadata {
    name      = "main-ingress"
    namespace = "tunelink"
    annotations = {
      "kubernetes.io/ingress.class"               = "alb"
      "alb.ingress.kubernetes.io/scheme"          = "internet-facing"
      "alb.ingress.kubernetes.io/target-type"     = "ip"
    }
  }

  spec {
    rule {
      http {
        path {
          path      = "/api"
          path_type = "Prefix"
          backend {
            service {
              name = "api"
              port { number = 80 }
            }
          }
        }
        path {
          path      = "/"
          path_type = "Prefix"
          backend {
            service {
              name = "web"
              port { number = 80 }
            }
          }
        }
      }
    }
  }
}
```
