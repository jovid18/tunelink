locals {
  namespace = "k6-operator-system"
}

# k6-operator Namespace (Helm이 인식할 수 있도록 레이블/어노테이션 추가)
resource "kubernetes_namespace" "k6_operator" {
  metadata {
    name = local.namespace

    labels = {
      "app.kubernetes.io/managed-by" = "Helm"
    }

    annotations = {
      "meta.helm.sh/release-name"      = "k6-operator"
      "meta.helm.sh/release-namespace" = local.namespace
    }
  }
}

# k6-operator Helm Release
resource "helm_release" "k6_operator" {
  name             = "k6-operator"
  repository       = "https://grafana.github.io/helm-charts"
  chart            = "k6-operator"
  namespace        = kubernetes_namespace.k6_operator.metadata[0].name
  version          = var.chart_version
  create_namespace = false

  # Namespace where TestRuns will be watched (all namespaces by default)
  set {
    name  = "manager.serviceAccount.create"
    value = "true"
  }

  # Resource limits for the operator
  set {
    name  = "manager.resources.requests.cpu"
    value = var.operator_cpu_request
  }

  set {
    name  = "manager.resources.requests.memory"
    value = var.operator_memory_request
  }

  set {
    name  = "manager.resources.limits.cpu"
    value = var.operator_cpu_limit
  }

  set {
    name  = "manager.resources.limits.memory"
    value = var.operator_memory_limit
  }

  # Helm chart의 namespace 생성 비활성화
  set {
    name  = "namespace.create"
    value = "false"
  }

  timeout = 300

  depends_on = [kubernetes_namespace.k6_operator]
}

# k6 Test Script ConfigMap (in tunelink namespace for easy service discovery)
resource "kubernetes_config_map" "k6_test_scripts" {
  metadata {
    name      = "k6-test-scripts"
    namespace = var.app_namespace
  }

  data = {
    "smoke-test.js"      = file("${path.module}/scripts/smoke-test.js")
    "load-test.js"       = file("${path.module}/scripts/load-test.js")
    "stress-test.js"     = file("${path.module}/scripts/stress-test.js")
    "breakpoint-test.js" = file("${path.module}/scripts/breakpoint-test.js")
  }
}
