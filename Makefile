.PHONY: update-drb-contracts
update-drb-contracts:
	@if [ -z "$(VERSION)" ]; then \
	  echo "Usage: make update-drb-contracts VERSION=<semver>"; exit 1; \
	fi
	@echo "Verifying npm package exists..."
	@npm view @tokamak-network/commit-reveal2-contracts@$(VERSION) version >/dev/null || \
	  (echo "Package @tokamak-network/commit-reveal2-contracts@$(VERSION) not found on npm"; exit 1)
	@echo "Updating drbNpmTag to $(VERSION)..."
	@sed -i.bak -E 's|drbNpmTag\s*=\s*"[^"]+"|drbNpmTag = "$(VERSION)"|' \
	  pkg/stacks/thanos/drb_genesis.go && rm pkg/stacks/thanos/drb_genesis.go.bak
	@echo "Running DRB unit tests..."
	@GOTOOLCHAIN=auto go test ./pkg/stacks/thanos/ -run "TestMaybeInjectDRB|TestPatchGenesisWithDRB|TestDerivePeerID|TestDeriveDRBAccounts|TestResolveDRBNpmTag" -v
	@echo ""
	@echo "✅ drbNpmTag updated to $(VERSION). Review changes with 'git diff' and commit."
