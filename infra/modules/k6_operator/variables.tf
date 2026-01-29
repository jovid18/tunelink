variable "environment" {
  description = "Environment name"
  type        = string
}

variable "chart_version" {
  description = "k6-operator Helm chart version"
  type        = string
  default     = "4.2.0"
}

variable "app_namespace" {
  description = "Namespace where the app is deployed (for test scripts ConfigMap)"
  type        = string
  default     = "tunelink"
}

variable "operator_cpu_request" {
  description = "CPU request for k6-operator"
  type        = string
  default     = "50m"
}

variable "operator_memory_request" {
  description = "Memory request for k6-operator"
  type        = string
  default     = "64Mi"
}

variable "operator_cpu_limit" {
  description = "CPU limit for k6-operator"
  type        = string
  default     = "100m"
}

variable "operator_memory_limit" {
  description = "Memory limit for k6-operator"
  type        = string
  default     = "128Mi"
}
