package thanos

import "context"

// RunAAOperatorFromConfig runs the AA operator using the stack's deployed config.
// This is a stub implementation for cross-trade feature branch compatibility.
func (s *ThanosStack) RunAAOperatorFromConfig(ctx context.Context) {
	<-ctx.Done()
}
