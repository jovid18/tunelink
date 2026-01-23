# TuneLink 배포 가이드

> 이 문서는 AWS 계정만 있는 상태에서 EKS 배포까지의 전체 과정을 기록합니다.
> 각 단계 완료 시 체크박스를 표시하세요.

---

## 현재 진행 상황

| Phase | 상태 | 설명 |
|-------|------|------|
| Phase 1: 로컬 개발 | ✅ 완료 | Docker로 로컬 실행 |
| Phase 2: AWS 인프라 | ✅ 완료 | Terraform으로 인프라 구축 + HTTPS |
| Phase 3: CI/CD | ✅ 완료 | GitHub Actions |

**배포 URL:** https://hearttune.link

---

## Phase 2: AWS 인프라 프로비저닝

### Step 1: AWS 초기 설정

#### 1.1 IAM 사용자 생성

- [x] AWS Console 로그인 (root 계정)
- [x] IAM > Users > Create User
  - 사용자 이름: `tunelink-operator`
  - AWS Management Console 액세스: 선택사항
- [x] 권한 설정 (아래 중 선택)

**옵션 A: AdministratorAccess (학습용, 간편)**
```
AdministratorAccess 정책 연결
```

**옵션 B: 최소 권한 (프로덕션 권장)**
```
필요한 정책들:
- AmazonVPCFullAccess
- AmazonEC2FullAccess
- AmazonEKSClusterPolicy
- AmazonEKSWorkerNodePolicy
- AmazonRDSFullAccess
- AmazonElastiCacheFullAccess
- AmazonEC2ContainerRegistryFullAccess
- AmazonS3FullAccess (Terraform state용)
- AmazonDynamoDBFullAccess (Terraform lock용)
- IAMFullAccess (EKS IRSA용)
```

#### 1.2 Access Key 생성

- [x] IAM > Users > tunelink-operator > Security credentials
- [x] Create access key > CLI 선택
- [x] Access Key ID와 Secret Access Key 저장 (한 번만 보임!)

```
Access Key ID: AKIA...
Secret Access Key: xxxxxxxxxxxxxxxx
```

#### 1.3 AWS CLI 설치

**macOS:**
```bash
brew install awscli
```

**설치 확인:**
```bash
aws --version
# aws-cli/2.x.x ...
```

#### 1.4 AWS CLI 프로파일 설정

- [x] 아래 명령어 실행

```bash
aws configure --profile tunelink
```

입력값:
```
AWS Access Key ID: [위에서 저장한 Access Key ID]
AWS Secret Access Key: [위에서 저장한 Secret Access Key]
Default region name: ap-northeast-2
Default output format: json
```

- [x] 설정 확인

```bash
aws sts get-caller-identity --profile tunelink
```

예상 출력:
```json
{
    "UserId": "AIDA...",
    "Account": "123456789012",
    "Arn": "arn:aws:iam::123456789012:user/tunelink-deployer"
}
```

#### 1.5 환경변수 설정 (선택)

매번 `--profile tunelink` 안 붙이려면:

```bash
# ~/.zshrc 또는 ~/.bashrc에 추가
export AWS_PROFILE=tunelink
export AWS_REGION=ap-northeast-2
```

```bash
source ~/.zshrc
```

---

### Step 2: Terraform 설치 및 백엔드 설정

#### 2.1 Terraform 설치

**macOS:**
```bash
brew install terraform
```

**설치 확인:**
```bash
terraform --version
# Terraform v1.x.x
```

#### 2.2 Terraform State용 S3 버킷 생성

> Terraform은 인프라 상태를 파일로 저장함. 여러 명이 작업하거나 CI/CD에서 쓰려면 S3에 저장해야 함.

- [x] S3 버킷 생성

```bash
# 버킷 이름은 전 세계적으로 유일해야 함
# 본인 계정 ID나 랜덤 문자열 추가 권장
aws s3 mb s3://tunelink-terraform-state-058264445568 --region ap-northeast-2
```

- [x] 버킷 버전 관리 활성화 (실수로 state 날려도 복구 가능)

```bash
aws s3api put-bucket-versioning \
  --bucket tunelink-terraform-state-{YOUR_ACCOUNT_ID} \
  --versioning-configuration Status=Enabled
```

#### 2.3 Terraform Lock용 DynamoDB 테이블 생성

> 동시에 terraform apply 실행 방지 (Lock)

- [x] DynamoDB 테이블 생성

```bash
aws dynamodb create-table \
  --table-name tunelink-terraform-lock \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region ap-northeast-2
```

#### 2.4 백엔드 설정 확인

생성 확인:
```bash
# S3 버킷 확인
aws s3 ls | grep tunelink

# DynamoDB 테이블 확인
aws dynamodb list-tables --region ap-northeast-2
```

---

### Step 3: Terraform 프로젝트 구조 생성

#### 3.1 디렉토리 구조 생성

- [x] 아래 구조로 폴더 생성

```
infra/
└── terraform/
    ├── envs/
    │   └── dev/
    │       ├── main.tf          # 모듈 호출
    │       ├── variables.tf     # 변수 정의
    │       ├── outputs.tf       # 출력값
    │       ├── terraform.tfvars # 변수 값 (git에 올리지 않음)
    │       ├── backend.tf       # S3 백엔드 설정
    │       └── providers.tf     # AWS, Kubernetes provider
    │
    └── modules/
        ├── vpc/
        ├── ecr/
        ├── rds/
        ├── elasticache/
        ├── eks/
        ├── alb_controller/
        ├── k8s_base/
        ├── k8s_api/
        └── k8s_web/
```

생성 명령어:
```bash
cd /Users/joseonghyeon/tunelink

mkdir -p infra/terraform/envs/dev
mkdir -p infra/terraform/modules/{vpc,ecr,rds,elasticache,eks,alb_controller,k8s_base,k8s_api,k8s_web}
```

---

### Step 4: Terraform 모듈 작성

> 각 모듈은 의존성 순서대로 작성 및 적용

#### 의존성 그래프
```
VPC ─┬─→ ECR (독립)
     ├─→ RDS (VPC 필요)
     ├─→ ElastiCache (VPC 필요)
     └─→ EKS (VPC 필요)
              │
              └─→ ALB Controller (EKS 필요)
                       │
                       └─→ K8s Base (ALB Controller 필요)
                                │
                                ├─→ K8s API
                                └─→ K8s Web
```

---

#### 4.1 VPC 모듈

**목적:** VPC, Subnet, Internet Gateway, NAT Gateway, Route Table 생성

- [x] `infra/terraform/modules/vpc/main.tf` 작성
- [x] `infra/terraform/modules/vpc/variables.tf` 작성
- [x] `infra/terraform/modules/vpc/outputs.tf` 작성
- [x] dev 환경에서 VPC 모듈 호출 추가
- [x] `terraform apply` 로 VPC 생성 확인

**VPC 구조:**
```
VPC (10.0.0.0/16)
├── Public Subnet 1 (10.0.1.0/24) - ap-northeast-2a
├── Public Subnet 2 (10.0.2.0/24) - ap-northeast-2c
├── Private Subnet 1 (10.0.11.0/24) - ap-northeast-2a
├── Private Subnet 2 (10.0.12.0/24) - ap-northeast-2c
├── Internet Gateway
├── NAT Gateway (Public Subnet에 위치)
└── Route Tables
```

**네트워크 아키텍처:**
```
                            인터넷
                               │
                               ▼
                    ┌──────────────────┐
                    │  Internet Gateway │
                    │   (tunelink-igw)  │
                    └────────┬─────────┘
                             │
        ┌────────────────────┴────────────────────┐
        │                   VPC                    │
        │             10.0.0.0/16                  │
        │                                          │
        │  ┌─────────────────────────────────────┐ │
        │  │         Public Subnets              │ │
        │  │  ┌───────────────┬───────────────┐  │ │
        │  │  │  10.0.1.0/24  │  10.0.2.0/24  │  │ │
        │  │  │     (2a)      │     (2c)      │  │ │
        │  │  │               │               │  │ │
        │  │  │ ┌───────────┐ │               │  │ │
        │  │  │ │    NAT    │ │               │  │ │
        │  │  │ │  Gateway  │ │               │  │ │
        │  │  │ └─────┬─────┘ │               │  │ │
        │  │  └───────┼───────┴───────────────┘  │ │
        │  └──────────┼──────────────────────────┘ │
        │             │                            │
        │             ▼                            │
        │  ┌─────────────────────────────────────┐ │
        │  │        Private Subnets              │ │
        │  │  ┌───────────────┬───────────────┐  │ │
        │  │  │ 10.0.11.0/24  │ 10.0.12.0/24  │  │ │
        │  │  │     (2a)      │     (2c)      │  │ │
        │  │  │               │               │  │ │
        │  │  │  [EKS Nodes]  │  [EKS Nodes]  │  │ │
        │  │  │  [RDS]        │  [RDS]        │  │ │
        │  │  │  [Redis]      │  [Redis]      │  │ │
        │  │  └───────────────┴───────────────┘  │ │
        │  └─────────────────────────────────────┘ │
        └──────────────────────────────────────────┘

라우팅:
┌─────────────────┬────────────────────────────────┐
│  Public RT      │  0.0.0.0/0 → Internet Gateway  │
├─────────────────┼────────────────────────────────┤
│  Private RT     │  0.0.0.0/0 → NAT Gateway       │
└─────────────────┴────────────────────────────────┘

트래픽 흐름:
• 외부 → ALB → Public Subnet → EKS (Private)
• EKS (Private) → NAT Gateway → 인터넷 (외부 API 호출 등)
```

**apply 후 확인:**
```bash
# VPC 확인
aws ec2 describe-vpcs --filters "Name=tag:Name,Values=tunelink-*" --query 'Vpcs[*].[VpcId,Tags[?Key==`Name`].Value|[0]]' --output table

# Subnet 확인
aws ec2 describe-subnets --filters "Name=tag:Name,Values=tunelink-*" --query 'Subnets[*].[SubnetId,AvailabilityZone,CidrBlock,Tags[?Key==`Name`].Value|[0]]' --output table
```

---

#### 4.2 ECR 모듈

**목적:** Docker 이미지 저장소 생성 (api, web)

- [x] `infra/terraform/modules/ecr/main.tf` 작성
- [x] `infra/terraform/modules/ecr/variables.tf` 작성
- [x] `infra/terraform/modules/ecr/outputs.tf` 작성
- [x] dev 환경에서 ECR 모듈 호출 추가
- [x] `terraform apply` 로 ECR 생성 확인

**apply 후 확인:**
```bash
aws ecr describe-repositories --query 'repositories[*].[repositoryName,repositoryUri]' --output table
```

**이미지 푸시 테스트 (ECR 생성 후):**
```bash
# ECR 로그인
aws ecr get-login-password --region ap-northeast-2 | docker login --username AWS --password-stdin {ACCOUNT_ID}.dkr.ecr.ap-northeast-2.amazonaws.com

# API 이미지 빌드 & 푸시
cd /Users/joseonghyeon/tunelink/apps/api
docker build -t tunelink-api .
docker tag tunelink-api:latest {ACCOUNT_ID}.dkr.ecr.ap-northeast-2.amazonaws.com/tunelink-api:latest
docker push {ACCOUNT_ID}.dkr.ecr.ap-northeast-2.amazonaws.com/tunelink-api:latest

# Web 이미지 빌드 & 푸시
cd /Users/joseonghyeon/tunelink/apps/web
docker build -t tunelink-web .
docker tag tunelink-web:latest {ACCOUNT_ID}.dkr.ecr.ap-northeast-2.amazonaws.com/tunelink-web:latest
docker push {ACCOUNT_ID}.dkr.ecr.ap-northeast-2.amazonaws.com/tunelink-web:latest
```

---

#### 4.3 RDS 모듈 (MySQL)

**목적:** MySQL 데이터베이스 생성

- [x] `infra/terraform/modules/rds/main.tf` 작성
- [x] `infra/terraform/modules/rds/variables.tf` 작성
- [x] `infra/terraform/modules/rds/outputs.tf` 작성
- [x] dev 환경에서 RDS 모듈 호출 추가
- [x] `terraform apply` 로 RDS 생성 확인

**RDS 설정:**
```
Engine: MySQL 8.0
Instance: db.t3.micro (프리티어 가능)
Storage: 20GB gp2
Multi-AZ: No (dev 환경)
Subnet: Private Subnet
```

**apply 후 확인:**
```bash
aws rds describe-db-instances --query 'DBInstances[*].[DBInstanceIdentifier,Endpoint.Address,DBInstanceStatus]' --output table
```

---

#### 4.4 ElastiCache 모듈 (Redis) - ⏸️ SKIP

**목적:** Redis 캐시 생성

> **2026-01-23: 일단 스킵**
> - API 코드가 Redis 없이도 동작하도록 설계됨 (캐시 미스 시 DB fallback)
> - dev 환경에서는 트래픽이 적어 캐시 불필요
> - 필요 시 나중에 추가 가능 (cache.t3.micro 프리티어 무료)

- [ ] `infra/terraform/modules/elasticache/main.tf` 작성
- [ ] `infra/terraform/modules/elasticache/variables.tf` 작성
- [ ] `infra/terraform/modules/elasticache/outputs.tf` 작성
- [ ] dev 환경에서 ElastiCache 모듈 호출 추가
- [ ] `terraform apply` 로 ElastiCache 생성 확인

**ElastiCache 설정:**
```
Engine: Redis 7.x
Node Type: cache.t3.micro
Num Nodes: 1
Subnet: Private Subnet
```

**apply 후 확인:**
```bash
aws elasticache describe-cache-clusters --query 'CacheClusters[*].[CacheClusterId,CacheNodeType,CacheClusterStatus]' --output table
```

---

#### 4.4.1 Bastion Host 모듈

**목적:** Private RDS 접근용 SSH 터널링 서버

- [x] `infra/terraform/modules/bastion/main.tf` 작성
- [x] `infra/terraform/modules/bastion/variables.tf` 작성
- [x] `infra/terraform/modules/bastion/outputs.tf` 작성
- [x] EC2 Key Pair 생성 (`tunelink-bastion`)
- [x] dev 환경에서 Bastion 모듈 호출 추가
- [x] `terraform apply` 로 Bastion 생성 확인

**Bastion 설정:**
```
Instance Type: t4g.micro (ARM, 프리티어)
AMI: Amazon Linux 2023
Subnet: Public Subnet
Key Pair: tunelink-bastion
```

**접속 정보:**
```
Host: 3.36.60.248
User: ec2-user
Key: ~/.ssh/tunelink-bastion.pem
```

**DataGrip에서 RDS 접속 (SSH Tunnel):**
```
[SSH/SSL 탭]
✅ Use SSH tunnel
Host: 3.36.60.248
Port: 22
User: ec2-user
Auth type: Key pair
Private key: ~/.ssh/tunelink-bastion.pem

[General 탭]
Host: tunelink-dev-mysql.cxm4yimyycnl.ap-northeast-2.rds.amazonaws.com
Port: 3306
User: tunelink_admin
Password: (terraform.tfvars 참고)
Database: tunelink
```

**SSH 접속 테스트:**
```bash
ssh -i ~/.ssh/tunelink-bastion.pem ec2-user@3.36.60.248
```

---

#### 4.5 EKS 모듈

**목적:** Kubernetes 클러스터 생성

- [x] `infra/terraform/modules/eks/main.tf` 작성
- [x] `infra/terraform/modules/eks/variables.tf` 작성
- [x] `infra/terraform/modules/eks/outputs.tf` 작성
- [x] dev 환경에서 EKS 모듈 호출 추가
- [x] `terraform apply` 로 EKS 생성 확인 (10-15분 소요)

**EKS 설정:**
```
Kubernetes Version: 1.29 (또는 최신)
Node Group:
  - capacity_type: SPOT (On-Demand 대비 60-70% 저렴)
  - Instance Types: [t3.small, t3.medium, t3a.small, t3a.medium]  # 여러 타입 지정 (가용성 높임)
  - Desired: 2
  - Min: 1
  - Max: 3
  - Subnet: Private Subnet
```

**Spot Instance 주의사항:**
- AWS가 용량 필요하면 2분 전 경고 후 회수 가능
- 여러 인스턴스 타입 지정하면 회수 확률 낮아짐
- Stateless 앱(API, Web)은 Spot에 적합
- 필요시 Node Termination Handler 설치 고려

**apply 후 확인:**
```bash
# EKS 클러스터 확인
aws eks describe-cluster --name tunelink-dev --query 'cluster.[name,status,endpoint]' --output table

# kubeconfig 업데이트
aws eks update-kubeconfig --name tunelink-dev --region ap-northeast-2

# kubectl 연결 확인
kubectl get nodes
kubectl get ns
```

---

#### 4.6 ALB Controller 모듈

**목적:** AWS Load Balancer Controller 설치 (Ingress용)

> EKS에서 Ingress를 만들면 자동으로 ALB가 생성되도록 하는 컨트롤러

- [x] `infra/terraform/modules/alb_controller/main.tf` 작성 (IRSA + Helm 또는 kubectl)
- [x] `infra/terraform/modules/alb_controller/variables.tf` 작성
- [x] `infra/terraform/modules/alb_controller/outputs.tf` 작성
- [x] dev 환경에서 ALB Controller 모듈 호출 추가
- [x] `terraform apply` 로 설치 확인

**필요한 것:**
1. IRSA (IAM Roles for Service Accounts) 설정
2. AWS Load Balancer Controller IAM Policy
3. Controller Deployment

**apply 후 확인:**
```bash
kubectl get deployment -n kube-system aws-load-balancer-controller
kubectl get pods -n kube-system | grep aws-load-balancer
```

---

#### 4.7 K8s Base 모듈

**목적:** Namespace, ConfigMap, Secret 생성

- [x] `infra/terraform/modules/k8s_base/main.tf` 작성
- [x] `infra/terraform/modules/k8s_base/variables.tf` 작성
- [x] `infra/terraform/modules/k8s_base/outputs.tf` 작성
- [x] dev 환경에서 K8s Base 모듈 호출 추가
- [x] `terraform apply` 로 리소스 생성 확인

**생성할 리소스:**
```yaml
# Namespace
- name: tunelink

# ConfigMap
- DB_HOST: (RDS endpoint)
- DB_PORT: 3306
- DB_NAME: tunelink
- REDIS_HOST: (ElastiCache endpoint)
- REDIS_PORT: 6379

# Secret
- DB_USER: (from terraform.tfvars)
- DB_PASSWORD: (from terraform.tfvars)
```

**apply 후 확인:**
```bash
kubectl get ns tunelink
kubectl get configmap -n tunelink
kubectl get secret -n tunelink
```

---

#### 4.8 K8s API 모듈

**목적:** API Deployment, Service 생성

- [x] `infra/terraform/modules/k8s_api/main.tf` 작성
- [x] `infra/terraform/modules/k8s_api/variables.tf` 작성
- [x] `infra/terraform/modules/k8s_api/outputs.tf` 작성
- [x] dev 환경에서 K8s API 모듈 호출 추가
- [x] `terraform apply` 로 배포 확인

**생성할 리소스:**
```yaml
# Deployment
- image: {ECR_URI}/tunelink-api:{TAG}
- replicas: 2
- port: 8080
- envFrom: configmap, secret
- resources:
    requests: cpu 100m, memory 128Mi
    limits: cpu 500m, memory 512Mi
- livenessProbe: /health
- readinessProbe: /health

# Service
- type: ClusterIP
- port: 80 -> 8080
```

**apply 후 확인:**
```bash
kubectl get deployment -n tunelink
kubectl get pods -n tunelink
kubectl get svc -n tunelink
kubectl logs -n tunelink -l app=api
```

---

#### 4.9 K8s Web 모듈

**목적:** Web Deployment, Service, Ingress 생성

- [x] `infra/terraform/modules/k8s_web/main.tf` 작성
- [x] `infra/terraform/modules/k8s_web/variables.tf` 작성
- [x] `infra/terraform/modules/k8s_web/outputs.tf` 작성
- [x] dev 환경에서 K8s Web 모듈 호출 추가
- [x] `terraform apply` 로 배포 확인

**생성할 리소스:**
```yaml
# Deployment
- image: {ECR_URI}/tunelink-web:{TAG}
- replicas: 2
- port: 80 (nginx)

# Service
- type: ClusterIP
- port: 80

# Ingress
- annotations:
    kubernetes.io/ingress.class: alb
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
- rules:
    - path: /api/* -> api service
    - path: /* -> web service
```

**apply 후 확인:**
```bash
kubectl get ingress -n tunelink
kubectl describe ingress -n tunelink main-ingress

# ALB DNS 확인 (배포 완료까지 2-3분)
kubectl get ingress -n tunelink -o jsonpath='{.items[0].status.loadBalancer.ingress[0].hostname}'
```

---

#### 4.10 도메인 및 HTTPS 설정

**목적:** 커스텀 도메인 연결 및 SSL 인증서 적용

- [x] Route 53에서 도메인 구매/등록
  - 도메인: `hearttune.link`
- [x] ACM 인증서 발급 (DNS 검증)
  - ARN: `arn:aws:acm:ap-northeast-2:058264445568:certificate/d2dd8141-7efd-4aed-bd94-c39402abd721`
  - 도메인: `hearttune.link`, `*.hearttune.link`
- [x] Terraform Ingress에 HTTPS 설정 추가
  - `k8s_web/main.tf` - annotations 수정
  - `k8s_web/variables.tf` - certificate_arn 변수 추가
  - `envs/dev/variables.tf` - 인증서 ARN, 도메인 설정
  - `envs/dev/main.tf` - certificate_arn 전달
- [x] Route 53 DNS 레코드 설정
  - `hearttune.link` → ALB (A Record Alias)
  - `www.hearttune.link` → ALB (A Record Alias)
- [x] `terraform apply` 로 HTTPS 적용 확인

**설정 구조:**
```
사용자 → https://hearttune.link
              │
              ▼
         Route 53 (A Record Alias)
              │
              ▼
         ALB (443 + ACM 인증서)
         ├── HTTP 80 → HTTPS 443 리다이렉트
         └── HTTPS 443 → EKS Pods
```

**IAM 인라인 정책 (추가됨):**
```
tunelink-operator 사용자:
- ACMFullAccess (인라인)
- Route53FullAccess (인라인)
```

**apply 후 확인:**
```bash
# DNS 확인
dig hearttune.link

# HTTPS 접속 테스트
curl -I https://hearttune.link

# 인증서 확인
echo | openssl s_client -servername hearttune.link -connect hearttune.link:443 2>/dev/null | openssl x509 -noout -dates
```

---

### Step 5: 전체 배포 테스트

- [ ] 커스텀 도메인 접속 확인: `https://hearttune.link`
- [ ] API health check 확인: `curl https://hearttune.link/health`
- [ ] URL 단축 기능 테스트
- [ ] 리다이렉트 기능 테스트

---

## Phase 3: CI/CD 구축

### Step 6: GitHub Actions 설정

#### 6.1 GitHub Secrets 설정

- [x] GitHub repo > Settings > Secrets and variables > Actions

추가한 Secrets:
```
AWS_ACCESS_KEY_ID: (IAM Access Key)
AWS_SECRET_ACCESS_KEY: (IAM Secret Key)
AWS_REGION: ap-northeast-2
AWS_ACCOUNT_ID: 058264445568
```

#### 6.2 API 워크플로우

- [x] `.github/workflows/api.yml` 작성

트리거: `apps/api/**` 변경 시 (main 브랜치)
```
test → build → ECR push → kubectl set image → rollout
```

#### 6.3 Web 워크플로우

- [x] `.github/workflows/web.yml` 작성

트리거: `apps/web/**` 변경 시 (main 브랜치)
```
build → ECR push → kubectl set image → rollout
```

#### 6.4 Infra 워크플로우

- [x] `.github/workflows/infra.yml` 작성

트리거: `infra/**` 변경 시
```
PR: terraform plan → 코멘트
main push: terraform apply
```

---

## 유용한 명령어 모음

### Terraform
```bash
cd /Users/joseonghyeon/tunelink/infra/terraform/envs/dev

# 초기화
terraform init

# 검증
terraform validate

# 플랜 (변경사항 미리보기)
terraform plan

# 적용
terraform apply

# 특정 모듈만 적용
terraform apply -target=module.vpc

# 삭제 (주의!)
terraform destroy
```

### kubectl
```bash
# 컨텍스트 확인
kubectl config current-context

# 전체 리소스 확인
kubectl get all -n tunelink

# 로그 확인
kubectl logs -n tunelink -l app=api -f

# Pod 접속
kubectl exec -it -n tunelink {POD_NAME} -- /bin/sh

# 재시작
kubectl rollout restart deployment/api -n tunelink
```

### AWS
```bash
# 계정 확인
aws sts get-caller-identity

# ECR 로그인
aws ecr get-login-password --region ap-northeast-2 | docker login --username AWS --password-stdin {ACCOUNT_ID}.dkr.ecr.ap-northeast-2.amazonaws.com

# EKS kubeconfig
aws eks update-kubeconfig --name tunelink-dev --region ap-northeast-2
```

---

## 비용 관리

### 예상 월 비용 (dev 환경, Spot Instance 사용)
| 리소스 | 비용 |
|--------|------|
| EKS Control Plane | ~$73 |
| EC2 Nodes (t3.small x2, **Spot**) | ~$10 (On-Demand 대비 ~70% 절감) |
| RDS (db.t3.micro) | ~$15 |
| ElastiCache (cache.t3.micro) | ~$12 |
| ALB | ~$20 |
| NAT Gateway | ~$32 |
| **합계** | **~$162/월** |

> Spot 가격은 변동됨. 실시간 확인: [EC2 Spot Pricing](https://aws.amazon.com/ec2/spot/pricing/)

### 비용 절감 팁
- [ ] 사용하지 않을 때 Node Group 0으로 스케일 다운
- [ ] NAT Gateway 대신 NAT Instance 사용 고려
- [ ] RDS/ElastiCache 중지 (dev용)

### 리소스 정리 (프로젝트 종료 시)
```bash
# 역순으로 삭제
terraform destroy -target=module.k8s_web
terraform destroy -target=module.k8s_api
terraform destroy -target=module.k8s_base
terraform destroy -target=module.alb_controller
terraform destroy -target=module.eks
terraform destroy -target=module.elasticache
terraform destroy -target=module.rds
terraform destroy -target=module.ecr
terraform destroy -target=module.vpc

# 또는 전체 삭제
terraform destroy
```

---

## 트러블슈팅

### EKS 연결 안 될 때
```bash
# kubeconfig 재설정
aws eks update-kubeconfig --name tunelink-dev --region ap-northeast-2 --profile tunelink

# IAM 권한 확인
aws sts get-caller-identity
```

### Pod가 Pending 상태일 때
```bash
kubectl describe pod -n tunelink {POD_NAME}
# Events 섹션 확인
```

### ALB가 생성 안 될 때
```bash
# ALB Controller 로그 확인
kubectl logs -n kube-system -l app.kubernetes.io/name=aws-load-balancer-controller

# Ingress 이벤트 확인
kubectl describe ingress -n tunelink main-ingress
```

### RDS 연결 안 될 때
```bash
# Security Group 확인 - EKS 노드에서 RDS로 3306 포트 열려있는지
# Private Subnet에 있으므로 로컬에서 직접 연결 불가
# EKS Pod에서 테스트:
kubectl run -it --rm mysql-client --image=mysql:8 -n tunelink -- mysql -h {RDS_ENDPOINT} -u {USER} -p
```

---

## 작업 기록

> 여기에 작업 내용을 날짜별로 기록하세요

### 2026-01-23
- [x] IAM 사용자 생성 (tunelink-operator)
- [x] Access Key 생성 및 AWS CLI 프로파일 설정
- [x] Terraform 백엔드 설정 (S3 + DynamoDB)
- [x] Terraform 프로젝트 구조 생성
- [x] VPC 모듈 작성 및 apply 완료
  - VPC: vpc-0ba7bda39fb6393a5
  - Public Subnets: 10.0.1.0/24, 10.0.2.0/24
  - Private Subnets: 10.0.11.0/24, 10.0.12.0/24
  - NAT Gateway, Internet Gateway 생성
- [x] ECR, RDS, EKS, ALB Controller 모듈 작성 및 apply 완료
- [x] K8s Base, API, Web 모듈 작성 및 apply 완료
- [x] 커스텀 도메인 및 HTTPS 설정
  - 도메인 구매: hearttune.link (Route 53)
  - ACM 인증서 발급: hearttune.link, *.hearttune.link
  - IAM 인라인 정책 추가: ACMFullAccess, Route53FullAccess
  - Terraform HTTPS 설정 (Ingress annotations)
  - Route 53 A Record: hearttune.link → ALB
  - Route 53 A Record: www.hearttune.link → ALB
- [x] GitHub Actions CI/CD 설정
  - GitHub Secrets 설정 (AWS credentials)
  - `.github/workflows/api.yml` - API 빌드/배포
  - `.github/workflows/web.yml` - Web 빌드/배포
  - ~~`.github/workflows/infra.yml`~~ - 삭제 (로컬에서 관리)
- [x] Bastion Host 추가
  - EC2 Key Pair 생성: tunelink-bastion
  - Bastion EC2: 3.36.60.248
  - DataGrip SSH Tunnel로 RDS 접속 가능
