variable "project_name" {
  description = "Project name"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "namespace" {
  description = "Kubernetes namespace"
  type        = string
}

variable "image_repository" {
  description = "Docker image repository URL"
  type        = string
}

variable "image_tag" {
  description = "Docker image tag"
  type        = string
  default     = "latest"
}

variable "replicas" {
  description = "Number of replicas"
  type        = number
  default     = 2
}

variable "container_port" {
  description = "Container port"
  type        = number
  default     = 8080
}

variable "config_map_name" {
  description = "ConfigMap name for environment variables"
  type        = string
}

variable "secret_name" {
  description = "Secret name for sensitive environment variables"
  type        = string
}

variable "base_url" {
  description = "Base URL for short URLs"
  type        = string
}

variable "resources" {
  description = "Resource limits and requests"
  type = object({
    requests_cpu    = string
    requests_memory = string
    limits_cpu      = string
    limits_memory   = string
  })
  default = {
    requests_cpu    = "100m"
    requests_memory = "128Mi"
    limits_cpu      = "500m"
    limits_memory   = "512Mi"
  }
}
