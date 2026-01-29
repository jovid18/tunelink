locals {
  name_prefix = "${var.project_name}-${var.environment}"
  app_name    = "api"
  labels = {
    app         = local.app_name
    environment = var.environment
    managed-by  = "terraform"
  }
}

# Deployment
resource "kubernetes_deployment" "api" {
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

          # Environment from ConfigMap
          env_from {
            config_map_ref {
              name = var.config_map_name
            }
          }

          # Environment from Secret
          env_from {
            secret_ref {
              name = var.secret_name
            }
          }

          # Additional environment variables
          env {
            name  = "BASE_URL"
            value = var.base_url
          }

          env {
            name  = "PORT"
            value = tostring(var.container_port)
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
              path = "/health"
              port = var.container_port
            }
            initial_delay_seconds = 15
            period_seconds        = 10
            timeout_seconds       = 5
            failure_threshold     = 3
          }

          readiness_probe {
            http_get {
              path = "/health"
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
resource "kubernetes_service" "api" {
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
