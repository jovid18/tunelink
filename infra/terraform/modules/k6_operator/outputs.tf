output "namespace" {
  description = "k6-operator namespace"
  value       = kubernetes_namespace.k6_operator.metadata[0].name
}

output "test_scripts_configmap" {
  description = "Name of the ConfigMap containing k6 test scripts"
  value       = kubernetes_config_map.k6_test_scripts.metadata[0].name
}

output "available_tests" {
  description = "List of available test scripts"
  value       = keys(kubernetes_config_map.k6_test_scripts.data)
}
