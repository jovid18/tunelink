# TuneLink Deployment Guide

> This document records the entire process from having only an AWS account to deploying on EKS.
> Check the checkbox upon completion of each step.

---

## Current Progress

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 1: Local Development | ✅ Complete | Running locally with Docker |
| Phase 2: AWS Infrastructure | ✅ Complete | Infrastructure provisioning with Terraform + HTTPS |
| Phase 3: CI/CD | ✅ Complete | GitHub Actions |

**Deployment URL:** https://hearttune.link

---

## Phase 2: AWS Infrastructure Provisioning

### Step 1: AWS Initial Setup

#### 1.1 Create IAM User

- [x] Log in to AWS Console (root account)
- [x] IAM > Users > Create User
  - Username: `tunelink-operator`
  - AWS Management Console access: Optional
- [x] Set permissions (choose one of the following)

**Option A: AdministratorAccess (for learning, simple)**
```
Attach AdministratorAccess policy
```

**Option B: Least Privilege (recommended for production)**
```
Required policies:
- AmazonVPCFullAccess
- AmazonEC2FullAccess
- AmazonEKSClusterPolicy
- AmazonEKSWorkerNodePolicy
- AmazonRDSFullAccess
- AmazonElastiCacheFullAccess
- AmazonEC2ContainerRegistryFullAccess
- AmazonS3FullAccess (for Terraform state)
- AmazonDynamoDBFullAccess (for Terraform lock)
- IAMFullAccess (for EKS IRSA)
```

#### 1.2 Create Access Key

- [x] IAM > Users > tunelink-operator > Security credentials
- [x] Create access key > Select CLI
- [x] Save Access Key ID and Secret Access Key (shown only once!)

```
Access Key ID: AKIA...
Secret Access Key: xxxxxxxxxxxxxxxx
```

#### 1.3 Install AWS CLI

**macOS:**
```bash
brew install awscli
```

**Verify installation:**
```bash
aws --version
# aws-cli/2.x.x ...
```

#### 1.4 Configure AWS CLI Profile

- [x] Run the following command

```bash
aws configure --profile tunelink
```

Input values:
```
AWS Access Key ID: [Access Key ID saved above]
AWS Secret Access Key: [Secret Access Key saved above]
Default region name: ap-northeast-2
Default output format: json
```

- [x] Verify configuration

```bash
aws sts get-caller-identity --profile tunelink
```

Expected output:
```json
{
    "UserId": "AIDA...",
    "Account": "123456789012",
    "Arn": "arn:aws:iam::123456789012:user/tunelink-deployer"
}
```

#### 1.5 Set Environment Variables (Optional)

To avoid appending `--profile tunelink` every time:

```bash
# Add to ~/.zshrc or ~/.bashrc
export AWS_PROFILE=tunelink
export AWS_REGION=ap-northeast-2
```

```bash
source ~/.zshrc
```

---

### Step 2: Install Terraform and Configure Backend

#### 2.1 Install Terraform

**macOS:**
```bash
brew install terraform
```

**Verify installation:**
```bash
terraform --version
# Terraform v1.x.x
```

#### 2.2 Create S3 Bucket for Terraform State

> Terraform saves infrastructure state as a file. To work with multiple people or use it in CI/CD, it must be stored in S3.

- [x] Create S3 bucket

```bash
# Bucket name must be globally unique
# Adding your account ID or random string is recommended
aws s3 mb s3://tunelink-terraform-state-058264445568 --region ap-northeast-2
```

- [x] Enable bucket versioning (allows recovery if state is accidentally deleted)

```bash
aws s3api put-bucket-versioning \
  --bucket tunelink-terraform-state-{YOUR_ACCOUNT_ID} \
  --versioning-configuration Status=Enabled
```

#### 2.3 Create DynamoDB Table for Terraform Lock

> Prevents simultaneous terraform apply executions (Lock)

- [x] Create DynamoDB table

```bash
aws dynamodb create-table \
  --table-name tunelink-terraform-lock \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region ap-northeast-2
```

#### 2.4 Verify Backend Configuration

Verify creation:
```bash
# Check S3 bucket
aws s3 ls | grep tunelink

# Check DynamoDB table
aws dynamodb list-tables --region ap-northeast-2
```

---

### Step 3: Create Terraform Project Structure

#### 3.1 Create Directory Structure

- [x] Create folders with the following structure

```
infra/
└── terraform/
    ├── envs/
    │   └── dev/
    │       ├── main.tf          # Module invocations
    │       ├── variables.tf     # Variable definitions
    │       ├── outputs.tf       # Output values
    │       ├── terraform.tfvars # Variable values (do not commit to git)
    │       ├── backend.tf       # S3 backend configuration
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

Creation command:
```bash
cd /Users/joseonghyeon/tunelink

mkdir -p infra/terraform/envs/dev
mkdir -p infra/terraform/modules/{vpc,ecr,rds,elasticache,eks,alb_controller,k8s_base,k8s_api,k8s_web}
```

---

### Step 4: Write Terraform Modules

> Each module is written and applied in dependency order

#### Dependency Graph
```
VPC ─┬─→ ECR (independent)
     ├─→ RDS (requires VPC)
     ├─→ ElastiCache (requires VPC)
     └─→ EKS (requires VPC)
              │
              └─→ ALB Controller (requires EKS)
                       │
                       └─→ K8s Base (requires ALB Controller)
                                │
                                ├─→ K8s API
                                └─→ K8s Web
```

---

#### 4.1 VPC Module

**Purpose:** Create VPC, Subnet, Internet Gateway, NAT Gateway, Route Table

- [x] Write `infra/terraform/modules/vpc/main.tf`
- [x] Write `infra/terraform/modules/vpc/variables.tf`
- [x] Write `infra/terraform/modules/vpc/outputs.tf`
- [x] Add VPC module invocation in dev environment
- [x] Verify VPC creation with `terraform apply`

**VPC Structure:**
```
VPC (10.0.0.0/16)
├── Public Subnet 1 (10.0.1.0/24) - ap-northeast-2a
├── Public Subnet 2 (10.0.2.0/24) - ap-northeast-2c
├── Private Subnet 1 (10.0.11.0/24) - ap-northeast-2a
├── Private Subnet 2 (10.0.12.0/24) - ap-northeast-2c
├── Internet Gateway
├── NAT Gateway (located in Public Subnet)
└── Route Tables
```

**Network Architecture:**
```
                            Internet
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

Routing:
┌─────────────────┬────────────────────────────────┐
│  Public RT      │  0.0.0.0/0 → Internet Gateway  │
├─────────────────┼────────────────────────────────┤
│  Private RT     │  0.0.0.0/0 → NAT Gateway       │
└─────────────────┴────────────────────────────────┘

Traffic Flow:
• External → ALB → Public Subnet → EKS (Private)
• EKS (Private) → NAT Gateway → Internet (external API calls, etc.)
```

**Verify after apply:**
```bash
# Check VPC
aws ec2 describe-vpcs --filters "Name=tag:Name,Values=tunelink-*" --query 'Vpcs[*].[VpcId,Tags[?Key==`Name`].Value|[0]]' --output table

# Check Subnets
aws ec2 describe-subnets --filters "Name=tag:Name,Values=tunelink-*" --query 'Subnets[*].[SubnetId,AvailabilityZone,CidrBlock,Tags[?Key==`Name`].Value|[0]]' --output table
```

---

#### 4.2 ECR Module

**Purpose:** Create Docker image repositories (api, web)

- [x] Write `infra/terraform/modules/ecr/main.tf`
- [x] Write `infra/terraform/modules/ecr/variables.tf`
- [x] Write `infra/terraform/modules/ecr/outputs.tf`
- [x] Add ECR module invocation in dev environment
- [x] Verify ECR creation with `terraform apply`

**Verify after apply:**
```bash
aws ecr describe-repositories --query 'repositories[*].[repositoryName,repositoryUri]' --output table
```

**Image push test (after ECR creation):**
```bash
# ECR login
aws ecr get-login-password --region ap-northeast-2 | docker login --username AWS --password-stdin {ACCOUNT_ID}.dkr.ecr.ap-northeast-2.amazonaws.com

# Build & push API image
cd /Users/joseonghyeon/tunelink/apps/api
docker build -t tunelink-api .
docker tag tunelink-api:latest {ACCOUNT_ID}.dkr.ecr.ap-northeast-2.amazonaws.com/tunelink-api:latest
docker push {ACCOUNT_ID}.dkr.ecr.ap-northeast-2.amazonaws.com/tunelink-api:latest

# Build & push Web image
cd /Users/joseonghyeon/tunelink/apps/web
docker build -t tunelink-web .
docker tag tunelink-web:latest {ACCOUNT_ID}.dkr.ecr.ap-northeast-2.amazonaws.com/tunelink-web:latest
docker push {ACCOUNT_ID}.dkr.ecr.ap-northeast-2.amazonaws.com/tunelink-web:latest
```

---

#### 4.3 RDS Module (MySQL)

**Purpose:** Create MySQL database

- [x] Write `infra/terraform/modules/rds/main.tf`
- [x] Write `infra/terraform/modules/rds/variables.tf`
- [x] Write `infra/terraform/modules/rds/outputs.tf`
- [x] Add RDS module invocation in dev environment
- [x] Verify RDS creation with `terraform apply`

**RDS Configuration:**
```
Engine: MySQL 8.0
Instance: db.t3.micro (Free Tier eligible)
Storage: 20GB gp2
Multi-AZ: No (dev environment)
Subnet: Private Subnet
```

**Verify after apply:**
```bash
aws rds describe-db-instances --query 'DBInstances[*].[DBInstanceIdentifier,Endpoint.Address,DBInstanceStatus]' --output table
```

---

#### 4.4 ElastiCache Module (Redis) - ✅ Complete

**Purpose:** Create Redis cache

> **2026-02-03: Implementation complete**
> - Reduced DB load with URL lookup caching
> - Click count increments handled via Redis INCR with batch synchronization
> - 93.5% response time improvement confirmed in Stress Test

- [x] Write `infra/terraform/modules/elasticache/main.tf`
- [x] Write `infra/terraform/modules/elasticache/variables.tf`
- [x] Write `infra/terraform/modules/elasticache/outputs.tf`
- [x] Add ElastiCache module invocation in dev environment
- [x] Verify ElastiCache creation with `terraform apply`

**ElastiCache Configuration:**
```
Engine: Redis 7.x
Node Type: cache.t3.micro
Num Nodes: 1
Subnet: Private Subnet
```

**Verify after apply:**
```bash
aws elasticache describe-cache-clusters --query 'CacheClusters[*].[CacheClusterId,CacheNodeType,CacheClusterStatus]' --output table
```

---

#### 4.4.1 Bastion Host Module

**Purpose:** SSH tunneling server for accessing Private RDS

- [x] Write `infra/terraform/modules/bastion/main.tf`
- [x] Write `infra/terraform/modules/bastion/variables.tf`
- [x] Write `infra/terraform/modules/bastion/outputs.tf`
- [x] Create EC2 Key Pair (`tunelink-bastion`)
- [x] Add Bastion module invocation in dev environment
- [x] Verify Bastion creation with `terraform apply`

**Bastion Configuration:**
```
Instance Type: t4g.micro (ARM, Free Tier)
AMI: Amazon Linux 2023
Subnet: Public Subnet
Key Pair: tunelink-bastion
```

**Connection Information:**
```
Host: 3.36.215.254
User: ec2-user
Key: ~/.ssh/tunelink-bastion.pem
```

**Connecting to RDS from DataGrip (SSH Tunnel):**
```
[SSH/SSL Tab]
✅ Use SSH tunnel
Host: 3.36.215.254
Port: 22
User: ec2-user
Auth type: Key pair
Private key: ~/.ssh/tunelink-bastion.pem

[General Tab]
Host: tunelink-dev-mysql.cxm4yimyycnl.ap-northeast-2.rds.amazonaws.com
Port: 3306
User: tunelink_admin
Password: (refer to terraform.tfvars)
Database: tunelink
```

**SSH Connection Test:**
```bash
ssh -i ~/.ssh/tunelink-bastion.pem ec2-user@3.36.215.254
```

---

#### 4.5 EKS Module

**Purpose:** Create Kubernetes cluster

- [x] Write `infra/terraform/modules/eks/main.tf`
- [x] Write `infra/terraform/modules/eks/variables.tf`
- [x] Write `infra/terraform/modules/eks/outputs.tf`
- [x] Add EKS module invocation in dev environment
- [x] Verify EKS creation with `terraform apply` (takes 10-15 minutes)

**EKS Configuration:**
```
Kubernetes Version: 1.29 (or latest)
Node Group:
  - capacity_type: SPOT (60-70% cheaper compared to On-Demand)
  - Instance Types: [t3.small, t3.medium, t3a.small, t3a.medium]  # Multiple types specified (increases availability)
  - Desired: 2
  - Min: 1
  - Max: 3
  - Subnet: Private Subnet
```

**Spot Instance Notes:**
- AWS can reclaim instances with 2-minute warning when capacity is needed
- Specifying multiple instance types reduces reclamation probability
- Stateless apps (API, Web) are well-suited for Spot
- Consider installing Node Termination Handler if needed

**Verify after apply:**
```bash
# Check EKS cluster
aws eks describe-cluster --name tunelink-dev --query 'cluster.[name,status,endpoint]' --output table

# Update kubeconfig
aws eks update-kubeconfig --name tunelink-dev --region ap-northeast-2

# Verify kubectl connection
kubectl get nodes
kubectl get ns
```

---

#### 4.6 ALB Controller Module

**Purpose:** Install AWS Load Balancer Controller (for Ingress)

> A controller that automatically creates ALBs when Ingress resources are created in EKS

- [x] Write `infra/terraform/modules/alb_controller/main.tf` (IRSA + Helm or kubectl)
- [x] Write `infra/terraform/modules/alb_controller/variables.tf`
- [x] Write `infra/terraform/modules/alb_controller/outputs.tf`
- [x] Add ALB Controller module invocation in dev environment
- [x] Verify installation with `terraform apply`

**Requirements:**
1. IRSA (IAM Roles for Service Accounts) configuration
2. AWS Load Balancer Controller IAM Policy
3. Controller Deployment

**Verify after apply:**
```bash
kubectl get deployment -n kube-system aws-load-balancer-controller
kubectl get pods -n kube-system | grep aws-load-balancer
```

---

#### 4.7 K8s Base Module

**Purpose:** Create Namespace, ConfigMap, Secret

- [x] Write `infra/terraform/modules/k8s_base/main.tf`
- [x] Write `infra/terraform/modules/k8s_base/variables.tf`
- [x] Write `infra/terraform/modules/k8s_base/outputs.tf`
- [x] Add K8s Base module invocation in dev environment
- [x] Verify resource creation with `terraform apply`

**Resources to Create:**
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

**Verify after apply:**
```bash
kubectl get ns tunelink
kubectl get configmap -n tunelink
kubectl get secret -n tunelink
```

---

#### 4.8 K8s API Module

**Purpose:** Create API Deployment, Service

- [x] Write `infra/terraform/modules/k8s_api/main.tf`
- [x] Write `infra/terraform/modules/k8s_api/variables.tf`
- [x] Write `infra/terraform/modules/k8s_api/outputs.tf`
- [x] Add K8s API module invocation in dev environment
- [x] Verify deployment with `terraform apply`

**Resources to Create:**
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

**Verify after apply:**
```bash
kubectl get deployment -n tunelink
kubectl get pods -n tunelink
kubectl get svc -n tunelink
kubectl logs -n tunelink -l app=api
```

---

#### 4.9 K8s Web Module

**Purpose:** Create Web Deployment, Service, Ingress

- [x] Write `infra/terraform/modules/k8s_web/main.tf`
- [x] Write `infra/terraform/modules/k8s_web/variables.tf`
- [x] Write `infra/terraform/modules/k8s_web/outputs.tf`
- [x] Add K8s Web module invocation in dev environment
- [x] Verify deployment with `terraform apply`

**Resources to Create:**
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

**Verify after apply:**
```bash
kubectl get ingress -n tunelink
kubectl describe ingress -n tunelink main-ingress

# Check ALB DNS (2-3 minutes until deployment completes)
kubectl get ingress -n tunelink -o jsonpath='{.items[0].status.loadBalancer.ingress[0].hostname}'
```

---

#### 4.10 Domain and HTTPS Setup

**Purpose:** Connect custom domain and apply SSL certificate

- [x] Purchase/register domain in Route 53
  - Domain: `hearttune.link`
- [x] Issue ACM certificate (DNS validation)
  - ARN: `arn:aws:acm:ap-northeast-2:058264445568:certificate/d2dd8141-7efd-4aed-bd94-c39402abd721`
  - Domains: `hearttune.link`, `*.hearttune.link`
- [x] Add HTTPS configuration to Terraform Ingress
  - `k8s_web/main.tf` - Modify annotations
  - `k8s_web/variables.tf` - Add certificate_arn variable
  - `envs/dev/variables.tf` - Certificate ARN, domain configuration
  - `envs/dev/main.tf` - Pass certificate_arn
- [x] Configure Route 53 DNS records
  - `hearttune.link` → ALB (A Record Alias)
  - `www.hearttune.link` → ALB (A Record Alias)
- [x] Verify HTTPS with `terraform apply`

**Configuration Structure:**
```
User → https://hearttune.link
              │
              ▼
         Route 53 (A Record Alias)
              │
              ▼
         ALB (443 + ACM Certificate)
         ├── HTTP 80 → HTTPS 443 Redirect
         └── HTTPS 443 → EKS Pods
```

**IAM Inline Policies (added):**
```
tunelink-operator user:
- ACMFullAccess (inline)
- Route53FullAccess (inline)
```

**Verify after apply:**
```bash
# Check DNS
dig hearttune.link

# Test HTTPS connection
curl -I https://hearttune.link

# Check certificate
echo | openssl s_client -servername hearttune.link -connect hearttune.link:443 2>/dev/null | openssl x509 -noout -dates
```

---

#### 4.11 k6-operator Module (Load Testing)

**Purpose:** Build Kubernetes-native load testing environment

> **Selection Rationale** (see `test.md`)
> - Industry-standard load testing tool
> - Optimized Grafana integration (same company product)
> - Test definition via K8s CRD → GitOps friendly

- [x] Write `infra/terraform/modules/k6_operator/main.tf`
- [x] Write `infra/terraform/modules/k6_operator/variables.tf`
- [x] Write `infra/terraform/modules/k6_operator/outputs.tf`
- [x] Add k6-operator module invocation in dev environment
- [x] Verify installation with `terraform apply`

**k6-operator Configuration:**
```
Namespace: k6-operator-system
CRD: TestRun (k6 test execution unit)
Helm Chart: grafana/k6-operator
```

**Architecture:**
```
┌─────────────────────────────────────────────────────────┐
│                    EKS Cluster                          │
│                                                         │
│  ┌─────────────────┐     ┌─────────────────────────┐   │
│  │  k6-operator    │     │     tunelink namespace   │   │
│  │  (controller)   │     │  ┌─────┐    ┌─────┐     │   │
│  └────────┬────────┘     │  │ API │    │ Web │     │   │
│           │              │  └──▲──┘    └──▲──┘     │   │
│           │ creates      │     │          │        │   │
│           ▼              │     │          │        │   │
│  ┌─────────────────┐     │     │          │        │   │
│  │   TestRun CRD   │─────┼─────┴──────────┘        │   │
│  │  (k6 script)    │     │   HTTP requests         │   │
│  └─────────────────┘     └─────────────────────────┘   │
│           │                                             │
│           ▼                                             │
│  ┌─────────────────┐                                   │
│  │   k6 Runner     │──────► Prometheus (metrics)       │
│  │   Pods (1~N)    │──────► Grafana (dashboard)        │
│  └─────────────────┘                                   │
└─────────────────────────────────────────────────────────┘
```

**Verify after apply:**
```bash
# Verify k6-operator installation
kubectl get deployment -n k6-operator-system

# Check CRD
kubectl get crd | grep k6

# Run sample test
kubectl apply -f - <<EOF
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: smoke-test
  namespace: tunelink
spec:
  parallelism: 2
  script:
    configMap:
      name: k6-test-script
      file: script.js
EOF

# Check test status
kubectl get testrun -n tunelink
kubectl logs -n tunelink -l app=k6 -f
```

**k6 Test Script Example (ConfigMap):**
```javascript
// script.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },   // ramp up
    { duration: '1m', target: 20 },    // stay
    { duration: '30s', target: 0 },    // ramp down
  ],
};

export default function () {
  // Health check
  let res = http.get('http://api.tunelink.svc.cluster.local/health');
  check(res, { 'health ok': (r) => r.status === 200 });

  // Create shortened URL
  res = http.post(
    'http://api.tunelink.svc.cluster.local/api/urls',
    JSON.stringify({ original_url: 'https://example.com/test' }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  check(res, { 'create ok': (r) => r.status === 200 || r.status === 201 });

  sleep(1);
}
```

**Grafana Integration (leveraging existing monitoring stack):**
- k6 supports Prometheus Remote Write
- Can add k6 dashboard to existing `prometheus-grafana`
- Dashboard ID: `2587` (k6 Load Testing Results)

---

### Step 5: Full Deployment Test

- [x] Verify custom domain access: `https://hearttune.link`
- [x] Verify API health check: `curl https://hearttune.link/health`
- [x] Test URL shortening functionality
- [x] Test redirect functionality

---

## Phase 3: CI/CD Setup

### Step 6: GitHub Actions Configuration

#### 6.1 GitHub Secrets Configuration

- [x] GitHub repo > Settings > Secrets and variables > Actions

Added Secrets:
```
AWS_ACCESS_KEY_ID: (IAM Access Key)
AWS_SECRET_ACCESS_KEY: (IAM Secret Key)
AWS_REGION: ap-northeast-2
AWS_ACCOUNT_ID: 058264445568
```

#### 6.2 API Workflow

- [x] Write `.github/workflows/api.yml`

Trigger: When `apps/api/**` changes (main branch)
```
test → build → ECR push → kubectl set image → rollout
```

#### 6.3 Web Workflow

- [x] Write `.github/workflows/web.yml`

Trigger: When `apps/web/**` changes (main branch)
```
build → ECR push → kubectl set image → rollout
```

#### 6.4 Infra Workflow

- [x] Write `.github/workflows/infra.yml`

Trigger: When `infra/**` changes
```
PR: terraform plan → comment
main push: terraform apply
```

---

## Useful Command Reference

### Terraform
```bash
cd /Users/joseonghyeon/tunelink/infra/terraform/envs/dev

# Initialize
terraform init

# Validate
terraform validate

# Plan (preview changes)
terraform plan

# Apply
terraform apply

# Apply specific module only
terraform apply -target=module.vpc

# Destroy (caution!)
terraform destroy
```

### kubectl
```bash
# Check context
kubectl config current-context

# Check all resources
kubectl get all -n tunelink

# Check logs
kubectl logs -n tunelink -l app=api -f

# Access Pod shell
kubectl exec -it -n tunelink {POD_NAME} -- /bin/sh

# Restart
kubectl rollout restart deployment/api -n tunelink
```

### AWS
```bash
# Check account
aws sts get-caller-identity

# ECR login
aws ecr get-login-password --region ap-northeast-2 | docker login --username AWS --password-stdin {ACCOUNT_ID}.dkr.ecr.ap-northeast-2.amazonaws.com

# EKS kubeconfig
aws eks update-kubeconfig --name tunelink-dev --region ap-northeast-2
```

---

## Cost Management

### Estimated Monthly Cost (dev environment, using Spot Instances)
| Resource | Cost |
|----------|------|
| EKS Control Plane | ~$73 |
| EC2 Nodes (t3.small x2, **Spot**) | ~$10 (~70% savings compared to On-Demand) |
| RDS (db.t3.micro) | ~$15 |
| ElastiCache (cache.t3.micro) | ~$12 |
| ALB | ~$20 |
| NAT Gateway | ~$32 |
| **Total** | **~$162/month** |

> Spot pricing fluctuates. Check real-time pricing: [EC2 Spot Pricing](https://aws.amazon.com/ec2/spot/pricing/)

### Cost Saving Tips
- [ ] Scale down Node Group to 0 when not in use
- [ ] Consider using NAT Instance instead of NAT Gateway
- [ ] Stop RDS/ElastiCache (for dev)

### Resource Cleanup (when project ends)
```bash
# Delete in reverse order
terraform destroy -target=module.k8s_web
terraform destroy -target=module.k8s_api
terraform destroy -target=module.k8s_base
terraform destroy -target=module.alb_controller
terraform destroy -target=module.eks
terraform destroy -target=module.elasticache
terraform destroy -target=module.rds
terraform destroy -target=module.ecr
terraform destroy -target=module.vpc

# Or destroy everything
terraform destroy
```

---

## Troubleshooting

### When EKS Connection Fails
```bash
# Reconfigure kubeconfig
aws eks update-kubeconfig --name tunelink-dev --region ap-northeast-2 --profile tunelink

# Check IAM permissions
aws sts get-caller-identity
```

### When Pod is in Pending State
```bash
kubectl describe pod -n tunelink {POD_NAME}
# Check the Events section
```

### When ALB is Not Created
```bash
# Check ALB Controller logs
kubectl logs -n kube-system -l app.kubernetes.io/name=aws-load-balancer-controller

# Check Ingress events
kubectl describe ingress -n tunelink main-ingress
```

### When RDS Connection Fails
```bash
# Check Security Group - verify port 3306 is open from EKS nodes to RDS
# Since it's in a Private Subnet, direct connection from local is not possible
# Test from an EKS Pod:
kubectl run -it --rm mysql-client --image=mysql:8 -n tunelink -- mysql -h {RDS_ENDPOINT} -u {USER} -p
```

---

## Work Log

> Record work items here by date

### 2026-01-23
- [x] Created IAM user (tunelink-operator)
- [x] Created Access Key and configured AWS CLI profile
- [x] Set up Terraform backend (S3 + DynamoDB)
- [x] Created Terraform project structure
- [x] Wrote and applied VPC module
  - VPC: vpc-0ba7bda39fb6393a5
  - Public Subnets: 10.0.1.0/24, 10.0.2.0/24
  - Private Subnets: 10.0.11.0/24, 10.0.12.0/24
  - Created NAT Gateway, Internet Gateway
- [x] Wrote and applied ECR, RDS, EKS, ALB Controller modules
- [x] Wrote and applied K8s Base, API, Web modules
- [x] Custom domain and HTTPS setup
  - Domain purchase: hearttune.link (Route 53)
  - ACM certificate issued: hearttune.link, *.hearttune.link
  - Added IAM inline policies: ACMFullAccess, Route53FullAccess
  - Terraform HTTPS configuration (Ingress annotations)
  - Route 53 A Record: hearttune.link → ALB
  - Route 53 A Record: www.hearttune.link → ALB
- [x] GitHub Actions CI/CD setup
  - GitHub Secrets configured (AWS credentials)
  - `.github/workflows/api.yml` - API build/deploy
  - `.github/workflows/web.yml` - Web build/deploy
- [x] Added Bastion Host
  - EC2 Key Pair created: tunelink-bastion
  - Bastion EC2: 3.36.215.254 (Elastic IP - static)
  - RDS accessible via DataGrip SSH Tunnel
