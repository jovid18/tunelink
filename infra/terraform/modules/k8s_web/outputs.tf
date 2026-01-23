output "deployment_name" {
  description = "Deployment name"
  value       = kubernetes_deployment.web.metadata[0].name
}

output "service_name" {
  description = "Service name"
  value       = kubernetes_service.web.metadata[0].name
}

output "ingress_name" {
  description = "Ingress name"
  value       = kubernetes_ingress_v1.main.metadata[0].name
}

output "alb_dns_name" {
  description = "ALB DNS name (available after ALB creation)"
  value       = try(kubernetes_ingress_v1.main.status[0].load_balancer[0].ingress[0].hostname, "pending")
}
