# VPC
module "vpc" {
  source = "../../modules/vpc"

  project_name = var.project_name
  environment  = var.environment
  vpc_cidr     = var.vpc_cidr
  aws_region   = var.aws_region
}

# ECR
module "ecr" {
  source = "../../modules/ecr"

  project_name = var.project_name
  environment  = var.environment
}

# RDS (MySQL)
module "rds" {
  source = "../../modules/rds"

  project_name       = var.project_name
  environment        = var.environment
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnet_ids
  db_username        = var.db_username
  db_password        = var.db_password
  instance_class     = "db.t3.micro"  # 프리티어
}

# Bastion Host (for DB access)
module "bastion" {
  source = "../../modules/bastion"

  project_name          = var.project_name
  environment           = var.environment
  vpc_id                = module.vpc.vpc_id
  public_subnet_id      = module.vpc.public_subnet_ids[0]
  key_name              = "tunelink-bastion"
  rds_security_group_id = module.rds.security_group_id

  depends_on = [module.rds]
}

# ElastiCache - SKIP (Redis 없이도 API 동작함)
# module "elasticache" { ... }

# EKS
module "eks" {
  source = "../../modules/eks"

  project_name       = var.project_name
  environment        = var.environment
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnet_ids

  kubernetes_version  = "1.33"
  node_instance_types = ["t3.small", "t3.medium", "t3a.small", "t3a.medium"]
  node_capacity_type  = "SPOT"
  node_desired_size   = 3
  node_min_size       = 2
  node_max_size       = 4
}

# AWS Load Balancer Controller
module "alb_controller" {
  source = "../../modules/alb_controller"

  project_name      = var.project_name
  environment       = var.environment
  cluster_name      = module.eks.cluster_name
  oidc_provider_arn = module.eks.oidc_provider_arn
  oidc_provider_url = module.eks.oidc_provider_url
  vpc_id            = module.vpc.vpc_id

  depends_on = [module.eks]
}

# Kubernetes Base (Namespace, ConfigMap, Secret)
module "k8s_base" {
  source = "../../modules/k8s_base"

  project_name = var.project_name
  environment  = var.environment
  namespace    = "tunelink"

  db_host     = module.rds.address
  db_port     = "3306"
  db_name     = "tunelink"
  db_username = var.db_username
  db_password = var.db_password

  # Redis - 현재 미사용
  redis_host = ""
  redis_port = "6379"

  depends_on = [module.alb_controller]
}

# Kubernetes API Deployment
module "k8s_api" {
  source = "../../modules/k8s_api"

  project_name     = var.project_name
  environment      = var.environment
  namespace        = module.k8s_base.namespace
  image_repository = module.ecr.repository_urls["api"]
  image_tag        = var.api_image_tag
  replicas         = 2
  config_map_name  = module.k8s_base.config_map_name
  secret_name      = module.k8s_base.secret_name
  base_url         = var.base_url

  depends_on = [module.k8s_base]
}

# Kubernetes Web Deployment + Ingress
module "k8s_web" {
  source = "../../modules/k8s_web"

  project_name     = var.project_name
  environment      = var.environment
  namespace        = module.k8s_base.namespace
  image_repository = module.ecr.repository_urls["web"]
  image_tag        = var.web_image_tag
  replicas         = 2
  api_service_name = module.k8s_api.service_name
  api_service_port = module.k8s_api.service_port
  certificate_arn  = var.certificate_arn

  depends_on = [module.k8s_api]
}

# Monitoring (Prometheus + Grafana)
module "monitoring" {
  source = "../../modules/monitoring"

  project_name           = var.project_name
  environment            = var.environment
  grafana_admin_password = var.grafana_admin_password

  # Prod 설정
  prometheus_retention      = "15d"
  prometheus_memory_request = "512Mi"
  prometheus_memory_limit   = "1Gi"
  grafana_memory_request    = "256Mi"
  grafana_memory_limit      = "512Mi"

  # Persistence 비활성화 (EBS CSI Driver 설치 후 활성화)
  enable_persistence = false

  # Ingress 설정
  ingress_enabled = true
  grafana_host    = "grafana.${var.domain_name}"
  certificate_arn = var.certificate_arn

  # AlertManager 활성화
  alertmanager_enabled = true

  depends_on = [module.eks]
}

# k6-operator (Load Testing)
module "k6_operator" {
  source = "../../modules/k6_operator"

  environment   = var.environment
  app_namespace = module.k8s_base.namespace

  depends_on = [module.eks, module.k8s_base]
}
