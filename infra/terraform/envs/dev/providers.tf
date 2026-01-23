terraform {
  required_version = ">= 1.0.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11"
    }
  }
}

provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile != "" ? var.aws_profile : null

  default_tags {
    tags = {
      Project     = "tunelink"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

# Kubernetes provider (EKS 클러스터 연결)
provider "kubernetes" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_ca_certificate)
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    args        = var.aws_profile != "" ? ["eks", "get-token", "--cluster-name", module.eks.cluster_name, "--profile", var.aws_profile] : ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
  }
}

# Helm provider (Helm 차트 설치용)
provider "helm" {
  kubernetes {
    host                   = module.eks.cluster_endpoint
    cluster_ca_certificate = base64decode(module.eks.cluster_ca_certificate)
    exec {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      args        = var.aws_profile != "" ? ["eks", "get-token", "--cluster-name", module.eks.cluster_name, "--profile", var.aws_profile] : ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
    }
  }
}
