variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "ap-northeast-2"
}

variable "aws_profile" {
  description = "AWS CLI profile"
  type        = string
  default     = "tunelink"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "dev"
}

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "tunelink"
}

# VPC
variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

# RDS
variable "db_username" {
  description = "Database master username"
  type        = string
  sensitive   = true
}

variable "db_password" {
  description = "Database master password"
  type        = string
  sensitive   = true
}

# Kubernetes
variable "api_image_tag" {
  description = "API Docker image tag"
  type        = string
  default     = "latest"
}

variable "web_image_tag" {
  description = "Web Docker image tag"
  type        = string
  default     = "latest"
}

variable "base_url" {
  description = "Base URL for short URLs (e.g., https://tunelink.example.com)"
  type        = string
  default     = "https://hearttune.link"
}

# Domain & SSL
variable "domain_name" {
  description = "Domain name for the application"
  type        = string
  default     = "hearttune.link"
}

variable "certificate_arn" {
  description = "ACM certificate ARN for HTTPS"
  type        = string
  default     = "arn:aws:acm:ap-northeast-2:058264445568:certificate/d2dd8141-7efd-4aed-bd94-c39402abd721"
}
