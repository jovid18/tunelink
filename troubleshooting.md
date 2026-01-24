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
