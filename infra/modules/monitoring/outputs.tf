output "namespace" {
  description = "Monitoring namespace"
  value       = kubernetes_namespace.monitoring.metadata[0].name
}

output "grafana_service_name" {
  description = "Grafana service name for port-forward"
  value       = "prometheus-grafana"
}

output "prometheus_service_name" {
  description = "Prometheus service name for port-forward"
  value       = "prometheus-kube-prometheus-prometheus"
}

output "grafana_port_forward_command" {
  description = "Command to access Grafana via port-forward"
  value       = "kubectl port-forward svc/prometheus-grafana -n monitoring 3000:80"
}

output "prometheus_port_forward_command" {
  description = "Command to access Prometheus via port-forward"
  value       = "kubectl port-forward svc/prometheus-kube-prometheus-prometheus -n monitoring 9090:9090"
}

output "grafana_url" {
  description = "Grafana URL (if ingress enabled)"
  value       = var.ingress_enabled ? "https://${var.grafana_host}" : "Use port-forward: kubectl port-forward svc/prometheus-grafana -n monitoring 3000:80"
}

output "grafana_alb_dns" {
  description = "Grafana ALB DNS name"
  value       = var.ingress_enabled ? try(kubernetes_ingress_v1.grafana[0].status[0].load_balancer[0].ingress[0].hostname, "ALB provisioning...") : null
}
