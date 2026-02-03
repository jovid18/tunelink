# Troubleshooting Guide

## Terraform

### Security Group Rule이 apply할 때마다 재생성되는 문제

**날짜:** 2026-01-25

**증상:**

- `terraform apply` 실행 시 `aws_security_group_rule` 리소스가 매번 새로 생성됨
- 예: `module.bastion.aws_security_group_rule.rds_from_bastion` 가 반복 생성

**원인:**

- Security Group에서 **inline rule** (`ingress {}`, `egress {}` 블록)과 **aws_security_group_rule** 리소스를 동시에 사용
- Terraform이 apply 시 inline rule만 유지하고 외부에서 추가된 rule을 삭제함
- 다음 apply에서 삭제된 rule이 다시 생성되는 무한 반복

**문제 코드 예시:**

```hcl
# RDS 모듈 - inline rule 사용
resource "aws_security_group" "rds" {
  ingress {  # inline rule
    from_port   = 3306
    ...
  }
}

# Bastion 모듈 - 별도 리소스로 같은 SG에 rule 추가
resource "aws_security_group_rule" "rds_from_bastion" {
  security_group_id = var.rds_security_group_id  # 충돌!
  ...
}
```

**해결 방법:**

- Security Group에서 inline rule을 제거하고, 모든 rule을 `aws_security_group_rule` 리소스로 분리

```hcl
# Security Group (rule 없이)
resource "aws_security_group" "rds" {
  name        = "rds-sg"
  description = "Security group for RDS"
  vpc_id      = var.vpc_id
  tags = { Name = "rds-sg" }
}

# 별도 리소스로 분리
resource "aws_security_group_rule" "rds_ingress_vpc" {
  type              = "ingress"
  from_port         = 3306
  to_port           = 3306
  protocol          = "tcp"
  cidr_blocks       = ["10.0.0.0/16"]
  security_group_id = aws_security_group.rds.id
}
```

**참고:**

- Terraform 공식 문서에서도 inline rule과 aws_security_group_rule 혼용을 권장하지 않음
- 여러 모듈에서 같은 Security Group에 rule을 추가해야 하는 경우 반드시 `aws_security_group_rule` 사용

---

### Security Group Rule 분리 후 InvalidPermission.Duplicate 에러

**날짜:** 2026-01-25

**증상:**

- inline rule을 `aws_security_group_rule`로 분리한 후 apply 시 에러 발생

```
Error: InvalidPermission.Duplicate: the specified rule "peer: 10.0.0.0/16, TCP, from port: 3306, to port: 3306, ALLOW" already exists
```

**원인:**

- AWS에는 이미 해당 rule이 존재하지만, Terraform state에는 새로운 리소스로 인식
- inline rule로 생성된 rule이 AWS에 남아있고, 새 리소스로 동일한 rule을 생성하려고 시도

**해결 방법:**

- `terraform import`로 기존 AWS rule을 Terraform state에 가져오기

```bash
# Security Group Rule import 형식
# {sg_id}_{type}_{protocol}_{from_port}_{to_port}_{cidr}

# ingress rule import
terraform import module.rds.aws_security_group_rule.rds_ingress_vpc \
  sg-060140f0813bf330b_ingress_tcp_3306_3306_10.0.0.0/16

# egress rule import (all traffic)
terraform import module.rds.aws_security_group_rule.rds_egress_all \
  sg-060140f0813bf330b_egress_all_0_0_0.0.0.0/0
```

**import 후 확인:**

```bash
terraform plan
# "No changes." 출력되면 성공
```

**참고:**

- Security Group Rule의 import ID 형식: `{sg_id}_{type}_{protocol}_{from_port}_{to_port}_{source}`
- source가 CIDR이면 그대로, Security Group이면 해당 SG ID 사용

---

### Bastion Host IP가 재시작 시 변경되는 문제

**날짜:** 2026-02-01

**증상:**

- Bastion EC2 인스턴스 재시작 후 Public IP가 변경됨
- DataGrip SSH Tunnel 연결 실패
- 문서에 기록된 IP와 실제 IP 불일치

**원인:**

- EC2 인스턴스에 자동 할당된 Public IP는 인스턴스 중지/시작 시 변경됨
- Elastic IP를 사용하지 않으면 IP가 고정되지 않음

**해결 방법:**

- Bastion 모듈에 Elastic IP 추가

```hcl
# infra/modules/bastion/main.tf

# Elastic IP for Bastion (IP 고정)
resource "aws_eip" "bastion" {
  domain = "vpc"

  tags = {
    Name = "${local.name_prefix}-bastion-eip"
  }
}

# Associate EIP with Bastion instance
resource "aws_eip_association" "bastion" {
  instance_id   = aws_instance.bastion.id
  allocation_id = aws_eip.bastion.id
}
```

```hcl
# infra/modules/bastion/outputs.tf

output "public_ip" {
  description = "Bastion public IP (Elastic IP)"
  value       = aws_eip.bastion.public_ip
}
```

**적용:**

```bash
cd infra/terraform/envs/dev
terraform plan -target=module.bastion
terraform apply -target=module.bastion
```

**비용:**

- Elastic IP가 EC2에 연결되어 있으면: **무료**
- 연결 안 된 EIP만 시간당 ~$0.005 비용 발생

---

## Helm

### k6-operator Helm 설치 시 Namespace 충돌

**날짜:** 2026-01-25

**증상:**

- `terraform apply` 시 다양한 namespace 관련 에러 발생

```
Error: namespaces "k6-operator-system" already exists
Error: no Namespace with the name "k6-operator-system" found
Error: invalid ownership metadata; label validation error: missing key "app.kubernetes.io/managed-by": must be set to "Helm"
```

**원인:**

- Terraform의 `kubernetes_namespace`와 Helm의 `create_namespace`가 충돌
- Helm은 자신이 관리하는 namespace에 특정 레이블/어노테이션이 있어야 함
- `helm uninstall` 시 namespace도 함께 삭제되어 상태 불일치 발생

**시도했던 방법들 (실패):**

1. **Helm만 사용 (`create_namespace = true`)**
   - namespace가 이미 있으면: `already exists` 에러
   - namespace가 없으면: 성공하지만, 다른 이유로 실패 시 상태 꼬임

2. **Terraform namespace + Helm (`create_namespace = false`)**
   - Helm이 namespace ownership 검사에서 실패
   - `invalid ownership metadata` 에러

3. **kubectl로 namespace 생성 후 Helm 설치**
   - Helm chart 자체가 namespace를 생성하려고 해서 충돌

**해결 방법:**

- Terraform으로 namespace 생성하되, **Helm이 인식할 수 있는 레이블/어노테이션 추가**
- Helm chart의 namespace 생성 옵션도 비활성화

```hcl
# 1. Namespace에 Helm 레이블/어노테이션 추가
resource "kubernetes_namespace" "k6_operator" {
  metadata {
    name = "k6-operator-system"

    labels = {
      "app.kubernetes.io/managed-by" = "Helm"
    }

    annotations = {
      "meta.helm.sh/release-name"      = "k6-operator"
      "meta.helm.sh/release-namespace" = "k6-operator-system"
    }
  }
}

# 2. Helm release 설정
resource "helm_release" "k6_operator" {
  name             = "k6-operator"
  repository       = "https://grafana.github.io/helm-charts"
  chart            = "k6-operator"
  namespace        = kubernetes_namespace.k6_operator.metadata[0].name
  version          = "4.2.0"
  create_namespace = false  # Terraform이 이미 생성함

  # Helm chart의 namespace 생성도 비활성화
  set {
    name  = "namespace.create"
    value = "false"
  }

  depends_on = [kubernetes_namespace.k6_operator]
}
```

**상태가 꼬였을 때 정리 방법:**

```bash
# 1. Helm release 삭제
helm uninstall k6-operator -n k6-operator-system

# 2. Namespace 삭제
kubectl delete ns k6-operator-system

# 3. Terraform state에서 제거
terraform state rm module.k6_operator.helm_release.k6_operator
terraform state rm module.k6_operator.kubernetes_namespace.k6_operator

# 4. 다시 apply
terraform apply
```

**핵심 포인트:**

- Helm은 자신이 관리하는 리소스에 `app.kubernetes.io/managed-by=Helm` 레이블 필요
- `meta.helm.sh/release-name`, `meta.helm.sh/release-namespace` 어노테이션도 필요
- 여러 도구가 같은 리소스를 관리하려 할 때 ownership 충돌 주의

---

## Kubernetes

### MySQL "Too many connections" 에러

**날짜:** 2026-02-02

**증상:**

- 부하테스트 중 redirect 요청 실패율 급증 (44% 실패)
- API Pod 로그에 아래 에러 반복:

```
Error 1040: Too many connections
Error 1040 (08004): Too many connections
```

**원인:**

- API 코드에서 DB Connection Pool 설정이 없음
- 각 Pod가 무제한으로 DB 연결 생성 시도
- RDS db.t3.micro 인스턴스의 max_connections (~66-87) 초과

**문제 코드:**

```go
// infrastructure/config.go
func connectDB() *gorm.DB {
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{...})
    // ❌ Connection Pool 설정 없음!
    return db
}
```

**해결 방법:**

- GORM에서 underlying sql.DB를 가져와 Connection Pool 설정 추가

```go
func connectDB() *gorm.DB {
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{...})

    // Connection Pool 설정
    sqlDB, err := db.DB()
    if err != nil {
        log.Fatal("Failed to get database instance:", err)
    }
    sqlDB.SetMaxOpenConns(25)               // Pod당 최대 25개 연결
    sqlDB.SetMaxIdleConns(10)               // 유휴 연결 10개 유지
    sqlDB.SetConnMaxLifetime(5 * time.Minute) // 연결 수명 5분

    return db
}
```

**설정값 계산:**

| 항목 | 값 | 설명 |
|------|-----|------|
| RDS max_connections | ~66-87 | db.t3.micro 기준 |
| API Pods | 2개 | 현재 replica 수 |
| Pod당 MaxOpenConns | 25 | 2 × 25 = 50 (RDS 한도 내) |
| 여유 연결 | ~16-37 | 다른 클라이언트용 (Bastion 등) |

**적용 결과 (2026-02-03 Stress Test):**

| 지표 | 적용 전 | 적용 후 |
|------|--------|--------|
| 에러율 | 28.2% | 0% |
| URL 생성 성공률 | ~40% | 100% |
| Peak RPS | 363 req/s | 733 req/s |

- Connection Pool 설정만으로 고부하 환경(1000 VUs)에서 안정성 확보
- 상세 결과: [test-result.md](./test-result.md)

**추가 옵션: RDS Proxy**

- Connection Pool을 앱이 아닌 AWS에서 중앙 관리
- Pod 스케일 아웃 시에도 RDS 연결 수 일정 유지
- 비용 발생하므로 대규모 트래픽 시 고려

---

### 클릭 카운트 동시성 문제 (Race Condition)

**날짜:** 2026-02-03

**증상:**

- Stress Test에서 클릭 수가 예상의 약 50%만 기록됨
- 예상: 300,000 클릭 (3,000 URLs × 100 redirects)
- 실제: 149,705 클릭 (~50%)
- 평균/최소/최대 클릭 수가 불균일 (avg: 49.9, min: 24, max: 86)

**원인:**

- Read → Modify → Write 패턴의 Race Condition (Lost Update)
- 동시 요청 시 여러 고루틴이 같은 값을 읽고 각각 +1 후 저장
- 결과적으로 일부 증가분 손실

**문제 코드:**

```go
// usecase.go - 이전 방식
func (uc *urlUseCase) incrementClicks(shortURL string) {
    ctx := context.Background()
    entity, err := uc.repo.FindByShortURL(ctx, shortURL)  // 1. 읽기: clicks=50
    if err != nil {
        return
    }
    entity.IncrementClicks()                              // 2. 메모리에서 +1: clicks=51
    uc.repo.Update(ctx, entity)                           // 3. 저장: clicks=51
}
// 동시에 10개 요청이 오면 모두 clicks=50을 읽고 51로 저장 → 9개 손실
```

**해결 방법:**

- SQL 레벨의 원자적 업데이트 사용 (커밋: 68f99c5)

```go
// url_repository.go - 수정된 방식
func (r *URLRepository) IncrementClicks(ctx context.Context, shortURL string) error {
    return r.db.WithContext(ctx).
        Model(&url.URL{}).
        Where("short_url = ?", shortURL).
        UpdateColumn("clicks", gorm.Expr("clicks + 1")).Error
        // SQL: UPDATE urls SET clicks = clicks + 1 WHERE short_url = ?
        // DB 레벨에서 원자적으로 처리되어 Lost Update 방지
}
```

**적용 결과 (2026-02-03 Stress Test):**

| 지표 | 적용 전 | 적용 후 |
|------|--------|--------|
| 예상 클릭 | 300,000 | 300,000 |
| 실제 클릭 | 149,705 (~50%) | 300,000 (100%) |
| 평균 클릭 | 49.9 | 100 |
| 최소/최대 | 24 / 86 | 100 / 100 |

**핵심 포인트:**

- 동시성 환경에서 카운터 증가는 반드시 **원자적 연산** 사용
- `SELECT → UPDATE`가 아닌 `UPDATE ... SET col = col + 1` 패턴
- 대안: Redis INCR, PostgreSQL RETURNING, DB Lock 등

**상세 결과:** [test-result.md](./test-result.md)

---

### Redis 캐시 통합 및 클릭 수 동기화

**날짜:** 2026-02-03

**배경:**

- DB 직접 조회 방식의 응답시간이 고부하 시 급격히 증가 (avg 687ms)
- 클릭 수 증가를 DB 원자적 업데이트로 처리해도 DB 부하 발생
- Redis 캐시 도입으로 성능 개선 필요

**구현 내용 (커밋: 339aac5, 0b67b36):**

1. **URL 조회 캐싱**
   - Redis에 URL 정보 캐싱 (TTL: 1시간)
   - 캐시 히트 시 DB 조회 스킵
   - 캐시 미스 시 DB 조회 후 캐싱

2. **클릭 수 Redis INCR + 배치 동기화**
   - 클릭 발생 시 Redis `INCR` (원자적, 빠름)
   - 백그라운드 워커가 주기적으로 DB 동기화
   - DB 부하 분산 + 정확성 보장

**아키텍처:**

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

**성능 개선 결과 (Stress Test 1000 VUs):**

| 지표 | Redis 도입 전 | Redis 도입 후 | 개선율 |
|------|-------------|--------------|--------|
| 평균 응답시간 | 687.93ms | 44.71ms | **-93.5%** |
| p(95) 응답시간 | 2.47s | 109.46ms | **-95.6%** |
| 처리량 | ~1,187 req/s | ~6,504 req/s | **+448%** |

**주의사항:**

1. **Redis 연결 실패 시**: DB fallback으로 서비스 지속 (graceful degradation)
2. **동기화 지연**: 클릭 수가 실시간이 아닌 배치로 DB에 반영됨
3. **캐시 무효화**: URL 수정 시 캐시 삭제 필요

**Redis 연결 확인:**

```bash
# EKS Pod에서 Redis 연결 테스트
kubectl run -it --rm redis-test --image=redis:7 -n tunelink -- \
  redis-cli -h tunelink-dev-redis.6d5ed3.0001.apn2.cache.amazonaws.com ping
```

**상세 결과:** [test-result.md](./test-result.md)

---

### k6 부하테스트 Pod 스케줄링 실패 (Too many pods)

**날짜:** 2026-01-25

**증상:**

- k6 TestRun 실행 시 starter Pod이 Pending 상태로 대기
- `kubectl describe pod` 시 아래 에러:

```
Warning  FailedScheduling  0/3 nodes are available: 3 Too many pods.
preemption: 0/3 nodes are available: 3 No preemption victims found for incoming pod.
```

**원인:**

- AWS EKS에서 Pod 수는 **인스턴스 타입의 ENI(Elastic Network Interface) 제한**에 의해 결정
- 작은 인스턴스 타입은 ENI당 할당 가능한 IP 수가 적음
- Pod마다 IP가 필요하므로 최대 Pod 수가 제한됨

**인스턴스별 최대 Pod 수:**
| 인스턴스 타입 | 최대 Pod 수 |
|-------------|-----------|
| t3.micro | 4 |
| t3.small | 11 |
| t3.medium | 17 |
| t3.large | 35 |
| t3.xlarge | 58 |

**현재 상태 확인:**

```bash
# 노드별 Pod 용량 확인
kubectl get nodes -o custom-columns="NAME:.metadata.name,CAPACITY:.status.capacity.pods,ALLOCATABLE:.status.allocatable.pods"

# 전체 Pod 수 확인
kubectl get pods -A --no-headers | wc -l
```

**해결 방법:**

1. **인스턴스 타입 업그레이드** - t3.medium 이상으로 변경
2. **노드 추가** - Auto Scaling Group의 desired capacity 증가
3. **불필요한 Pod 정리** - 사용하지 않는 워크로드 제거

**k6 테스트 리소스 정리:**

```bash
# TestRun 삭제 (관련 Pod 자동 정리)
kubectl delete testrun <testrun-name> -n tunelink

# 확인
kubectl get pods -n tunelink | grep k6
```

**참고:**

- AWS ENI 제한 계산: `(ENI 수 × ENI당 IP 수) - 1`
- 공식 문서: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.html
