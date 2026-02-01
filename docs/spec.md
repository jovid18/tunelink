# TuneLink - URL Shortener

## 서비스 개요

| 항목 | 결정 |
|------|------|
| 서비스 | URL Shortener (링크 단축 서비스) |
| 인증 | 없음 (MVP 단계) |
| 진행 방식 | 로컬 개발환경 먼저 → AWS 배포 |

### 핵심 기능 (MVP)

- 긴 URL → 짧은 URL 생성
- 짧은 URL 접속 시 원본 URL로 리다이렉트
- (선택) 클릭 수 통계

### API 엔드포인트

| Method | Path | 설명 |
|--------|------|------|
| GET | `/health` | 헬스체크 |
| POST | `/api/urls` | URL 단축 생성 |
| GET | `/r/{shortUrl}` | 리다이렉트 |

### DB 스키마

**테이블: `urls`**
| 컬럼 | 타입 | 설명 |
|------|------|------|
| id | BIGINT | PK |
| short_url | VARCHAR(10) | 단축 코드 |
| original_url | TEXT | 원본 URL |
| clicks | BIGINT | 클릭 수 |
| created_at | TIMESTAMP | 생성일 |

---

## 기술 스택

| 영역 | 기술 |
|------|------|
| Backend | Go + Gin |
| Frontend | React + Vite |
| Database | MySQL |
| Cache | Redis |
| Container | Docker |
| Orchestration | Kubernetes (EKS) |
| IaC | Terraform (Helm 없음) |
| CI/CD | GitHub Actions |

---

## 레포 구조 (모노레포)

```
tunelink/
├── apps/
│   ├── api/                     # Go + Gin API 서버
│   │   ├── cmd/
│   │   │   └── api/
│   │   │       └── main.go
│   │   ├── internal/
│   │   │   ├── handler/         # HTTP 핸들러
│   │   │   ├── service/         # 비즈니스 로직
│   │   │   ├── repository/      # DB 접근
│   │   │   ├── model/           # 도메인 모델
│   │   │   └── config/          # 설정
│   │   ├── Dockerfile
│   │   ├── go.mod
│   │   └── go.sum
│   │
│   └── web/                     # React + Vite 프론트엔드
│       ├── src/
│       │   ├── components/
│       │   ├── pages/
│       │   ├── hooks/
│       │   ├── api/
│       │   └── main.tsx
│       ├── Dockerfile           # nginx로 정적 파일 서빙
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
│           ├── eks/             # EKS 클러스터
│           ├── rds/             # MySQL (RDS)
│           ├── elasticache/     # Redis (ElastiCache)
│           ├── ecr/             # 컨테이너 레지스트리
│           ├── alb_controller/  # AWS Load Balancer Controller
│           ├── k8s_base/        # namespace, configmap, secret
│           ├── k8s_api/         # api deployment, service
│           └── k8s_web/         # web deployment, service, ingress
│
├── .github/
│   └── workflows/
│       ├── api.yml              # api 변경 시 빌드/푸시/배포
│       ├── web.yml              # web 변경 시 빌드/푸시/배포
│       └── infra.yml            # infra 변경 시 plan/apply
│
├── scripts/
│   ├── local-dev.sh             # 로컬 개발 환경 셋업
│   └── init-tf.sh               # Terraform 초기화
│
├── docs/
├── Makefile
├── docker-compose.yml           # 로컬 개발용 (MySQL, Redis)
└── README.md
```

---

## 개발 흐름

### Phase 1: 로컬 개발 환경

```
[개발자 로컬]
    │
    ├── docker-compose up        # MySQL + Redis 로컬 실행
    ├── apps/api → go run        # API 서버 (localhost:8080)
    └── apps/web → npm run dev   # Vite dev server (localhost:5173)
```

**목표**: API/Web 기능 개발, DB 스키마 설계

---

### Phase 2: AWS 인프라 프로비저닝

```
[Terraform Apply 순서]

1. VPC 모듈
   └── VPC, Public/Private Subnet, IGW, NAT Gateway

2. ECR 모듈
   └── api, web 이미지 저장소

3. RDS 모듈
   └── MySQL (Private Subnet)

4. ElastiCache 모듈
   └── Redis (Private Subnet)

5. EKS 모듈
   └── EKS Cluster + Node Group (Private Subnet)

6. ALB Controller 모듈
   └── AWS Load Balancer Controller (IRSA)

7. K8s Base 모듈
   └── Namespace, ConfigMap, Secret

8. K8s API/Web 모듈
   └── Deployment, Service, Ingress
```

**의존성 그래프:**
```
VPC ─┬─→ RDS
     ├─→ ElastiCache
     ├─→ EKS ─→ ALB Controller ─→ K8s Base ─┬─→ K8s API
     │                                       └─→ K8s Web
     └─→ ECR
```

---

### Phase 3: CI/CD 파이프라인

#### API 변경 시 (apps/api/**)
```
Push → GitHub Actions
         │
         ├── Go Build & Test
         ├── Docker Build
         ├── ECR Push (tag: commit SHA)
         └── Terraform Apply
             └── k8s_api 모듈 (image_tag 변수 업데이트)
```

#### Web 변경 시 (apps/web/**)
```
Push → GitHub Actions
         │
         ├── npm install & build
         ├── Docker Build (nginx)
         ├── ECR Push (tag: commit SHA)
         └── Terraform Apply
             └── k8s_web 모듈 (image_tag 변수 업데이트)
```

#### Infra 변경 시 (infra/**)
```
Push → GitHub Actions
         │
         ├── Terraform Plan (PR에 코멘트)
         └── Terraform Apply (main 머지 시)
```

---

## 네트워크 구조

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

## Ingress 라우팅 (Path 기반)

```
example.com/          → Web Service (React)
example.com/api/*     → API Service (Go)
```

**Terraform 리소스 예시:**
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

---

## 개발 순서 (권장)

### Step 1: 앱 개발 (로컬)
- [ ] Go API 기본 구조 (health check, CORS)
- [ ] MySQL 스키마 설계 & 마이그레이션
- [ ] React 기본 구조
- [ ] docker-compose로 로컬 통합 테스트

### Step 2: 컨테이너화
- [ ] API Dockerfile 작성
- [ ] Web Dockerfile 작성 (nginx)
- [ ] 로컬에서 컨테이너 빌드/실행 테스트

### Step 3: Terraform 모듈 개발
- [ ] VPC 모듈
- [ ] ECR 모듈
- [ ] RDS 모듈
- [ ] ElastiCache 모듈
- [ ] EKS 모듈
- [ ] ALB Controller 모듈
- [ ] K8s 모듈 (base, api, web)

### Step 4: 수동 배포 테스트
- [ ] ECR에 이미지 푸시
- [ ] terraform apply로 전체 인프라 + 앱 배포
- [ ] 동작 확인

### Step 5: CI/CD 구축
- [ ] GitHub Actions 워크플로우 작성
- [ ] PR 시 plan, merge 시 apply 자동화

---

## 환경 변수 / 시크릿 관리

| 항목 | 저장 위치 | K8s 주입 방식 |
|------|----------|--------------|
| DB Host/Port | Terraform output | ConfigMap |
| DB User/Password | AWS Secrets Manager | External Secrets 또는 Terraform kubernetes_secret |
| Redis Host/Port | Terraform output | ConfigMap |
| API URL (for Web) | 빌드 시 주입 | Vite env |

---

## 예상 비용 (dev 환경 기준)

| 리소스 | 스펙 | 월 예상 |
|--------|------|---------|
| EKS Control Plane | - | ~$73 |
| EKS Node (t3.medium x2) | 2 vCPU, 4GB | ~$60 |
| RDS (db.t3.micro) | 1 vCPU, 1GB | ~$15 |
| ElastiCache (cache.t3.micro) | 1 vCPU, 0.5GB | ~$12 |
| ALB | - | ~$20 |
| NAT Gateway | - | ~$32 |
| ECR | 저장량에 따라 | ~$1 |
| **합계** | | **~$213/월** |

> 비용 절감 팁: dev는 NAT Gateway 대신 NAT Instance, 또는 Public Subnet에 노드 배치 고려

---

## 다음 단계

1. 이 구조 확정되면 → 폴더/파일 생성
2. Phase 1부터 순차적으로 진행
3. 각 단계별로 PR + 리뷰
