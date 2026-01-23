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
  default     = 80
}

variable "api_service_name" {
  description = "API service name for Ingress routing"
  type        = string
}

variable "api_service_port" {
  description = "API service port"
  type        = number
  default     = 80
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
    requests_cpu    = "50m"
    requests_memory = "64Mi"
    limits_cpu      = "200m"
    limits_memory   = "256Mi"
  }
}

variable "certificate_arn" {
  description = "ACM certificate ARN for HTTPS"
  type        = string
  default     = ""
}
