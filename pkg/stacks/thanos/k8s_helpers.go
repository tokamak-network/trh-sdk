// Package thanos provides the ThanosStack deployment and management logic.
// Kubernetes helper methods are split across domain files:
//   - k8s_pvc_pv_helpers.go     — PVC and PV operations
//   - k8s_monitoring_helpers.go — AlertManager, PrometheusRule, Secret
//   - k8s_ingress_service_helpers.go — Ingress, Service, PV volume handles
//   - k8s_namespace_helpers.go  — Namespace operations
//   - k8s_resource_helpers.go   — Pods, ConfigMaps, generic resources
package thanos
