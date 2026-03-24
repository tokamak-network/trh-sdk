package thanos

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
)

type InstallStatus int32

const (
	InstallPending   InstallStatus = iota
	InstallRunning
	InstallCompleted
	InstallFailed
)

type ToolReadiness struct {
	results  map[string]chan error
	statuses map[string]*atomic.Int32
	once     sync.Once
	logger   *zap.SugaredLogger
	arch     string
}

func NewToolReadiness(logger *zap.SugaredLogger, arch string) *ToolReadiness {
	tools := []string{"terraform", "aws-cli", "kubectl", "helm"}
	results := make(map[string]chan error, len(tools))
	statuses := make(map[string]*atomic.Int32, len(tools))
	for _, t := range tools {
		results[t] = make(chan error, 1)
		statuses[t] = &atomic.Int32{}
	}
	return &ToolReadiness{
		results:  results,
		statuses: statuses,
		logger:   logger,
		arch:     arch,
	}
}

func (tr *ToolReadiness) Start(ctx context.Context) {
	tr.once.Do(func() {
		if err := dependencies.CheckDiskSpace("/app/storage", dependencies.MinDiskSpaceMB); err != nil {
			tr.logger.Errorf("[tool-install] %v", err)
			for _, ch := range tr.results {
				ch <- err
			}
			return
		}

		tr.logger.Infof("[tool-install] Starting parallel installation: terraform@%s, aws-cli@%s, kubectl@%s, helm@%s",
			dependencies.TerraformVersion, dependencies.AwsCLIVersion, dependencies.KubectlVersion, dependencies.HelmVersion)

		type toolEntry struct {
			name    string
			install func(context.Context, *zap.SugaredLogger, string) error
		}
		tools := []toolEntry{
			{"terraform", dependencies.InstallTerraform},
			{"aws-cli", dependencies.InstallAwsCLI},
			{"kubectl", dependencies.InstallKubectl},
			{"helm", dependencies.InstallHelm},
		}

		for _, t := range tools {
			go func(name string, install func(context.Context, *zap.SugaredLogger, string) error) {
				tr.statuses[name].Store(int32(InstallRunning))
				installCtx, cancel := context.WithTimeout(ctx, time.Duration(dependencies.InstallTimeoutSeconds)*time.Second)
				defer cancel()

				var err error
				defer func() {
					if err != nil {
						tr.statuses[name].Store(int32(InstallFailed))
					} else {
						tr.statuses[name].Store(int32(InstallCompleted))
					}
					tr.results[name] <- err
				}()

				err = install(installCtx, tr.logger, tr.arch)
			}(t.name, t.install)
		}
	})
}

func (tr *ToolReadiness) WaitFor(ctx context.Context, tools ...string) error {
	for _, tool := range tools {
		ch, ok := tr.results[tool]
		if !ok {
			return fmt.Errorf("unknown tool: %s", tool)
		}
		select {
		case err := <-ch:
			ch <- err
			if err != nil {
				return fmt.Errorf("%s: %w", tool, err)
			}
		case <-ctx.Done():
			return fmt.Errorf("context canceled while waiting for %s: %w", tool, ctx.Err())
		}
	}
	tr.logger.Infof("[tool-install] All requested tools ready: %v", tools)
	return nil
}

func (tr *ToolReadiness) Status() map[string]InstallStatus {
	result := make(map[string]InstallStatus, len(tr.statuses))
	for name, v := range tr.statuses {
		result[name] = InstallStatus(v.Load())
	}
	return result
}
