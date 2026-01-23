output "deployment_name" {
  description = "Deployment name"
  value       = kubernetes_deployment.api.metadata[0].name
}

output "service_name" {
  description = "Service name"
  value       = kubernetes_service.api.metadata[0].name
}

output "service_port" {
  description = "Service port"
  value       = kubernetes_service.api.spec[0].port[0].port
}
