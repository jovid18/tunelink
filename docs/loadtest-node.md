# Loadtest 노드 격리 설정

## 개요

k6 부하테스트 Pod를 API Pod와 격리하여 정확한 성능 측정을 위한 설정입니다.

## 아키텍처

```
                          ┌─────────────────┐
                          │   인터넷        │
                          │ hearttune.link  │
                          └────────┬────────┘
                                   │ HTTPS
                                   ▼
┌──────────────────────────────────────────────────────────────────┐
│                         EKS Cluster                              │
│                                                                  │
│                    ┌─────────────────┐                           │
│                    │  Ingress / ALB  │                           │
│                    └────────┬────────┘                           │
│                             │                                    │
│  ┌────────────────────────┐ │  ┌────────────────────────┐       │
│  │   일반 노드 그룹        │ │  │  Loadtest 노드 그룹     │       │
│  │   (tunelink-dev-node)  │ │  │  (tunelink-dev-loadtest)│       │
│  │                        │ │  │                        │       │
│  │  ┌─────┐  ┌─────┐     │ │  │  ┌─────┐               │       │
│  │  │ API │  │ Web │     │ │  │  │ k6  │               │       │
│  │  └──┬──┘  └─────┘     │ │  │  └──┬──┘               │       │
│  │     ▲                 │ │  │     │                  │       │
│  └─────┼─────────────────┘ │  └─────┼──────────────────┘       │
│        │                   │        │                           │
│        └───────────────────┘        │  외부 경로                │
│                                     │  (https://hearttune.link) │
│                                     ▼                           │
│                              ┌──────────────┐                   │
│                              │   인터넷     │                   │
│                              └──────────────┘                   │
└──────────────────────────────────────────────────────────────────┘
```

> **참고**: k6는 외부 URL(`https://hearttune.link`)을 통해 테스트하므로, 실제 사용자와 동일한 경로(Ingress/ALB → API)로 부하테스트가 진행됩니다.

## 왜 격리하나?

| 방식 | 문제점 |
|------|--------|
| 같은 노드 | k6가 CPU/Memory 사용 → API 성능 저하 → 결과 왜곡 |
| **다른 노드 (격리)** | 리소스 경쟁 없음 → 정확한 측정 |

## Terraform 설정

### 노드 그룹 구성 (`infra/modules/eks/main.tf`)

```hcl
# Loadtest 전용 노드 그룹
resource "aws_eks_node_group" "loadtest" {
  count = var.loadtest_node_enabled ? 1 : 0

  node_group_name = "${local.name_prefix}-loadtest"
  instance_types  = ["t3.xlarge", "t3a.xlarge"]  # 4 vCPU, 16GB
  capacity_type   = "SPOT"

  scaling_config {
    desired_size = var.loadtest_node_desired_size
    min_size     = 0
    max_size     = 2
  }

  labels = {
    role = "loadtest"
  }

  taint {
    key    = "role"
    value  = "loadtest"
    effect = "NO_SCHEDULE"
  }
}
```

### 변수 설정 (`infra/main.tf`)

```hcl
module "eks" {
  # ...

  # Loadtest 노드 (테스트 시에만 활성화)
  loadtest_node_enabled      = true
  loadtest_node_desired_size = 0  # 0 = 꺼짐, 1 = 켜짐
}
```

## 사용 방법

### 1. 테스트 시작 전: 노드 켜기

```bash
# infra/main.tf 수정
loadtest_node_desired_size = 1

# 적용
cd infra
terraform apply -target=module.eks

# 노드 Ready 확인 (1-2분 소요)
kubectl get nodes -l role=loadtest
```

### 2. 테스트 실행

k6 TestRun에 nodeSelector와 toleration 필요:

```yaml
spec:
  runner:
    nodeSelector:
      role: loadtest
    tolerations:
      - key: "role"
        operator: "Equal"
        value: "loadtest"
        effect: "NoSchedule"
```

### 3. 테스트 완료 후: 노드 끄기

```bash
# infra/main.tf 수정
loadtest_node_desired_size = 0

# 적용
cd infra
terraform apply -target=module.eks
```

## Taint & Toleration 설명

### Taint (노드에 설정)
```yaml
taint:
  key: role
  value: loadtest
  effect: NO_SCHEDULE
```
→ 일반 Pod는 이 노드에 스케줄링되지 않음

### Toleration (Pod에 설정)
```yaml
tolerations:
  - key: "role"
    operator: "Equal"
    value: "loadtest"
    effect: "NoSchedule"
```
→ k6 Pod만 이 노드에 스케줄링 허용

## 비용

| 상태 | 노드 수 | 비용 |
|------|--------|------|
| 평소 (꺼짐) | 0 | $0 |
| 테스트 중 (켜짐) | 1 | ~$0.05/시간 (SPOT t3.xlarge, 4 vCPU, 16GB) |

> **참고**: SPOT 가격은 가용 영역과 시간대에 따라 변동됨. On-Demand 대비 60~70% 저렴.

## 주의사항

1. **SPOT 인스턴스**: 테스트 중 노드가 종료될 수 있음 (드묾)
2. **노드 시작 시간**: 1-2분 소요
3. **테스트 후 끄기**: 비용 절약을 위해 꼭 끄기

## 관련 파일

- `infra/modules/eks/main.tf` - 노드 그룹 정의
- `infra/modules/eks/variables.tf` - 변수 정의
- `infra/main.tf` - 변수 값 설정
- `.claude/skills/k6-load-test/SKILL.md` - 테스트 스킬
