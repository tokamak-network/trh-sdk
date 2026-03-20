# 발견 사항 & 공유 자료

## 2026-03-09 — 환경 스캔: helm/doctl call site 목록

### helm 21개 call site
- pkg/utils/helm.go:17 — ListReleases (helm list --namespace X -q)
- pkg/utils/helm.go:81 — InstallOrUpgrade (helm upgrade --install ...)
- pkg/utils/helm.go:96 — Uninstall (helm uninstall releaseName --namespace X)
- pkg/stacks/thanos/deploy_chain.go:329 — repo add (helm repo add)
- pkg/stacks/thanos/deploy_chain.go:341 — repo search (helm search repo)
- pkg/stacks/thanos/monitoring.go:85 — dependency update
- pkg/stacks/thanos/monitoring.go:103 — install
- pkg/stacks/thanos/monitoring.go:313 — uninstall
- pkg/stacks/thanos/bridge.go:142 — install
- pkg/stacks/thanos/bridge.go:199 — upgrade
- pkg/stacks/thanos/block_explorer.go:219 — install
- pkg/stacks/thanos/block_explorer.go:292 — upgrade
- pkg/stacks/thanos/block_explorer.go:334 — uninstall
- pkg/stacks/thanos/uptime_service.go:187 — install/upgrade
- pkg/stacks/thanos/uptime_service.go:267 — uninstall
- pkg/stacks/thanos/destroy_chain.go:72 — uninstall
- pkg/stacks/thanos/destroy_chain.go:170 — uninstall
- pkg/stacks/thanos/update_network.go:168 — upgrade
- pkg/stacks/thanos/cross_trade.go:1000 — install
- pkg/stacks/thanos/cross_trade.go:1059 — upgrade
- pkg/dependencies/dependencies.go:24 — version check (helm version)

### doctl 4개 call site
- pkg/cloud-provider/digitalocean/digitalocean.go:20 — account get (ValidateToken)
- pkg/cloud-provider/digitalocean/digitalocean.go:31 — compute region list (ListRegions)
- pkg/cloud-provider/digitalocean/digitalocean.go:58 — kubernetes cluster kubeconfig save (GetKubeconfig)
- pkg/dependencies/dependencies.go:101 — version check (doctl version)

### Phase 1 완료 상태
- pkg/runner/k8s.go — K8sRunner 인터페이스
- pkg/runner/runner.go — ToolRunner 팩토리
- pkg/runner/native_k8s.go — NativeK8sRunner
- pkg/runner/shellout.go — ShellOutK8sRunner
- pkg/runner/mock/mock_k8s.go — mock

### 의존성 (go.mod에 추가 필요)
- helm.sh/helm/v3 v3.14.0
- github.com/digitalocean/godo v1.109.0

---

# DEAD_ENDS (시도했으나 실패한 접근)
(초기 상태 — 없음)
