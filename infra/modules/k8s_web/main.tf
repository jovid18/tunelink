locals {
  name_prefix = "${var.project_name}-${var.environment}"
  app_name    = "web"
  labels = {
    app         = local.app_name
    environment = var.environment
    managed-by  = "terraform"
  }
}

# Deployment
resource "kubernetes_deployment" "web" {
  metadata {
    name      = "${local.name_prefix}-${local.app_name}"
    namespace = var.namespace
    labels    = local.labels
  }

  spec {
    replicas = var.replicas

    selector {
      match_labels = {
        app = local.app_name
      }
    }

    template {
      metadata {
        labels = local.labels
      }

      spec {
        container {
          name  = local.app_name
          image = "${var.image_repository}:${var.image_tag}"

          port {
            container_port = var.container_port
          }

          resources {
            requests = {
              cpu    = var.resources.requests_cpu
              memory = var.resources.requests_memory
            }
            limits = {
              cpu    = var.resources.limits_cpu
              memory = var.resources.limits_memory
            }
          }

          liveness_probe {
            http_get {
              path = "/"
              port = var.container_port
            }
            initial_delay_seconds = 10
            period_seconds        = 10
            timeout_seconds       = 5
            failure_threshold     = 3
          }

          readiness_probe {
            http_get {
              path = "/"
              port = var.container_port
            }
            initial_delay_seconds = 5
            period_seconds        = 5
            timeout_seconds       = 3
            failure_threshold     = 3
          }
        }
      }
    }
  }
}

# Service
resource "kubernetes_service" "web" {
  metadata {
    name      = "${local.name_prefix}-${local.app_name}"
    namespace = var.namespace
    labels    = local.labels
  }

  spec {
    selector = {
      app = local.app_name
    }

    port {
      port        = 80
      target_port = var.container_port
      protocol    = "TCP"
    }

    type = "ClusterIP"
  }
}

# Ingress (ALB)
resource "kubernetes_ingress_v1" "main" {
  metadata {
    name      = "${local.name_prefix}-ingress"
    namespace = var.namespace
    labels    = local.labels

    annotations = {
      "kubernetes.io/ingress.class"                = "alb"
      "alb.ingress.kubernetes.io/scheme"           = "internet-facing"
      "alb.ingress.kubernetes.io/target-type"      = "ip"
      "alb.ingress.kubernetes.io/healthcheck-path" = "/health"
      "alb.ingress.kubernetes.io/listen-ports"     = "[{\"HTTP\": 80}, {\"HTTPS\": 443}]"
      "alb.ingress.kubernetes.io/certificate-arn"  = var.certificate_arn
      "alb.ingress.kubernetes.io/ssl-redirect"     = "443"
    }
  }

  spec {
    # API routes
    rule {
      http {
        # API endpoints
        path {
          path      = "/api"
          path_type = "Prefix"

          backend {
            service {
              name = var.api_service_name
              port {
                number = var.api_service_port
              }
            }
          }
        }

        # Health check endpoint
        path {
          path      = "/health"
          path_type = "Exact"

          backend {
            service {
              name = var.api_service_name
              port {
                number = var.api_service_port
              }
            }
          }
        }

        # Redirect endpoint
        path {
          path      = "/r"
          path_type = "Prefix"

          backend {
            service {
              name = var.api_service_name
              port {
                number = var.api_service_port
              }
            }
          }
        }

        # Web (default - catch all)
        path {
          path      = "/"
          path_type = "Prefix"

          backend {
            service {
              name = kubernetes_service.web.metadata[0].name
              port {
                number = 80
              }
            }
          }
        }
      }
    }
  }
}
