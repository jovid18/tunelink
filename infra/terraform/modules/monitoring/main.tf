locals {
  name_prefix = "${var.project_name}-${var.environment}"
  namespace   = "monitoring"
}

# Monitoring Namespace
resource "kubernetes_namespace" "monitoring" {
  metadata {
    name = local.namespace

    labels = {
      name        = local.namespace
      environment = var.environment
    }
  }
}

# Prometheus + Grafana Stack (kube-prometheus-stack)
resource "helm_release" "prometheus_stack" {
  name       = "prometheus"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "kube-prometheus-stack"
  namespace  = kubernetes_namespace.monitoring.metadata[0].name
  version    = var.chart_version

  # Prometheus 설정
  set {
    name  = "prometheus.prometheusSpec.retention"
    value = var.prometheus_retention
  }

  set {
    name  = "prometheus.prometheusSpec.resources.requests.memory"
    value = var.prometheus_memory_request
  }

  set {
    name  = "prometheus.prometheusSpec.resources.requests.cpu"
    value = var.prometheus_cpu_request
  }

  set {
    name  = "prometheus.prometheusSpec.resources.limits.memory"
    value = var.prometheus_memory_limit
  }

  set {
    name  = "prometheus.prometheusSpec.resources.limits.cpu"
    value = var.prometheus_cpu_limit
  }

  # Grafana 설정
  set {
    name  = "grafana.adminPassword"
    value = var.grafana_admin_password
  }

  set {
    name  = "grafana.resources.requests.memory"
    value = var.grafana_memory_request
  }

  set {
    name  = "grafana.resources.requests.cpu"
    value = var.grafana_cpu_request
  }

  set {
    name  = "grafana.resources.limits.memory"
    value = var.grafana_memory_limit
  }

  set {
    name  = "grafana.resources.limits.cpu"
    value = var.grafana_cpu_limit
  }

  # Service Type (ClusterIP - port-forward로 접속)
  set {
    name  = "grafana.service.type"
    value = "ClusterIP"
  }

  # AlertManager 비활성화 (dev 환경)
  set {
    name  = "alertmanager.enabled"
    value = var.alertmanager_enabled
  }

  # Node Exporter - 노드 메트릭 수집
  set {
    name  = "nodeExporter.enabled"
    value = "true"
  }

  # kube-state-metrics - K8s 오브젝트 메트릭
  set {
    name  = "kubeStateMetrics.enabled"
    value = "true"
  }

  # SPOT 인스턴스를 위한 tolerations (모든 노드에서 실행 가능)
  values = [
    yamlencode({
      prometheus = {
        prometheusSpec = {
          storageSpec = var.enable_persistence ? {
            volumeClaimTemplate = {
              spec = {
                accessModes = ["ReadWriteOnce"]
                resources = {
                  requests = {
                    storage = var.prometheus_storage_size
                  }
                }
              }
            }
          } : null
        }
      }
      grafana = {
        persistence = {
          enabled = var.enable_persistence
          size    = var.grafana_storage_size
        }
      }
    })
  ]

  timeout = 600

  depends_on = [kubernetes_namespace.monitoring]
}

# Grafana Ingress (ALB)
resource "kubernetes_ingress_v1" "grafana" {
  count = var.ingress_enabled ? 1 : 0

  metadata {
    name      = "${local.name_prefix}-grafana-ingress"
    namespace = kubernetes_namespace.monitoring.metadata[0].name

    annotations = {
      "kubernetes.io/ingress.class"                = "alb"
      "alb.ingress.kubernetes.io/scheme"           = "internet-facing"
      "alb.ingress.kubernetes.io/target-type"      = "ip"
      "alb.ingress.kubernetes.io/healthcheck-path" = "/api/health"
      "alb.ingress.kubernetes.io/listen-ports"     = "[{\"HTTP\": 80}, {\"HTTPS\": 443}]"
      "alb.ingress.kubernetes.io/certificate-arn"  = var.certificate_arn
      "alb.ingress.kubernetes.io/ssl-redirect"     = "443"
    }
  }

  spec {
    rule {
      host = var.grafana_host

      http {
        path {
          path      = "/"
          path_type = "Prefix"

          backend {
            service {
              name = "prometheus-grafana"
              port {
                number = 80
              }
            }
          }
        }
      }
    }
  }

  depends_on = [helm_release.prometheus_stack]
}
