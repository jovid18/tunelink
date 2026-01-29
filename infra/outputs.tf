# VPC Outputs
output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "public_subnet_ids" {
  description = "Public subnet IDs"
  value       = module.vpc.public_subnet_ids
}

output "private_subnet_ids" {
  description = "Private subnet IDs"
  value       = module.vpc.private_subnet_ids
}

# ECR Outputs
output "ecr_repository_urls" {
  description = "ECR repository URLs"
  value       = module.ecr.repository_urls
}

# RDS Outputs
output "rds_endpoint" {
  description = "RDS endpoint"
  value       = module.rds.endpoint
}

output "rds_address" {
  description = "RDS address"
  value       = module.rds.address
}

# EKS Outputs
output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = module.eks.cluster_endpoint
}

# ALB Outputs
output "alb_dns_name" {
  description = "ALB DNS name"
  value       = module.k8s_web.alb_dns_name
}

# Bastion Outputs
output "bastion_public_ip" {
  description = "Bastion host public IP"
  value       = module.bastion.public_ip
}

output "bastion_public_dns" {
  description = "Bastion host public DNS"
  value       = module.bastion.public_dns
}

# Monitoring Outputs
output "grafana_url" {
  description = "Grafana URL"
  value       = module.monitoring.grafana_url
}

output "grafana_alb_dns" {
  description = "Grafana ALB DNS (for Route53 CNAME)"
  value       = module.monitoring.grafana_alb_dns
}

output "prometheus_port_forward" {
  description = "Command to access Prometheus"
  value       = module.monitoring.prometheus_port_forward_command
}
