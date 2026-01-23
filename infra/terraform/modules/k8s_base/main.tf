locals {
  name_prefix = "${var.project_name}-${var.environment}"
}

# Namespace
resource "kubernetes_namespace" "main" {
  metadata {
    name = var.namespace

    labels = {
      name        = var.namespace
      environment = var.environment
      managed-by  = "terraform"
    }
  }
}

# ConfigMap
resource "kubernetes_config_map" "app_config" {
  metadata {
    name      = "${local.name_prefix}-config"
    namespace = kubernetes_namespace.main.metadata[0].name
  }

  data = {
    DB_HOST    = var.db_host
    DB_PORT    = var.db_port
    DB_NAME    = var.db_name
    REDIS_HOST = var.redis_host
    REDIS_PORT = var.redis_port
  }
}

# Secret
resource "kubernetes_secret" "app_secret" {
  metadata {
    name      = "${local.name_prefix}-secret"
    namespace = kubernetes_namespace.main.metadata[0].name
  }

  data = {
    DB_USER     = var.db_username
    DB_PASSWORD = var.db_password
  }

  type = "Opaque"
}
