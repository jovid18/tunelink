# Troubleshooting Guide

## Terraform

### Security Group Rule이 apply할 때마다 재생성되는 문제

**날짜:** 2025-01-25

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

**날짜:** 2025-01-25

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

**날짜:** 2025-02-01

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

**날짜:** 2025-01-25

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

### k6 부하테스트 Pod 스케줄링 실패 (Too many pods)

**날짜:** 2025-01-25

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
