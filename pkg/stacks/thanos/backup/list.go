package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ListRecoveryPoints queries recovery points and returns parsed entries
func ListRecoveryPoints(ctx context.Context, region, arn, limit string) ([]types.RecoveryPoint, error) {
	if strings.TrimSpace(limit) == "" {
		limit = "20"
	}
	jsonQuery := fmt.Sprintf("reverse(sort_by(RecoveryPoints,&CreationDate))[:%s].{Vault:BackupVaultName,Created:CreationDate,Expiry:ExpiryDate,Status:Status}", limit)
	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource",
		"--region", region,
		"--resource-arn", arn,
		"--query", jsonQuery,
		"--output", "json",
	)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" || out == "[]" {
		return nil, nil
	}
	var rps []types.RecoveryPoint
	if err := json.Unmarshal([]byte(out), &rps); err != nil {
		return nil, err
	}
	return rps, nil
}

// DisplayRecoveryPoints renders recovery points with expiry calculation
func DisplayRecoveryPoints(l *zap.SugaredLogger, rps []types.RecoveryPoint) {
	if len(rps) == 0 {
		l.Infof("   ⚠️  No recovery points found")
		return
	}

	// Calculate dynamic column widths based on actual data
	maxVaultLen := 10   // minimum width for "Vault" header
	maxCreatedLen := 10 // minimum width for "Created" header
	maxExpiryLen := 10  // minimum width for "Expiry" header
	maxStatusLen := 10  // minimum width for "Status" header

	// Find maximum lengths for each column
	for _, rp := range rps {
		if len(rp.Vault) > maxVaultLen {
			maxVaultLen = len(rp.Vault)
		}
		if len(rp.Created) > maxCreatedLen {
			maxCreatedLen = len(rp.Created)
		}
		if len(rp.Expiry) > maxExpiryLen {
			maxExpiryLen = len(rp.Expiry)
		}
		if len(rp.Status) > maxStatusLen {
			maxStatusLen = len(rp.Status)
		}
	}

	// Apply reasonable maximum limits to prevent extremely wide tables
	const maxVaultWidth = 50
	const maxCreatedWidth = 40
	const maxExpiryWidth = 40
	const maxStatusWidth = 15

	if maxVaultLen > maxVaultWidth {
		maxVaultLen = maxVaultWidth
	}
	if maxCreatedLen > maxCreatedWidth {
		maxCreatedLen = maxCreatedWidth
	}
	if maxExpiryLen > maxExpiryWidth {
		maxExpiryLen = maxExpiryWidth
	}
	if maxStatusLen > maxStatusWidth {
		maxStatusLen = maxStatusWidth
	}

	// Calculate total table width
	totalWidth := maxVaultLen + maxCreatedLen + maxExpiryLen + maxStatusLen + 12 // 12 for spacing and borders

	l.Infof("   %s", strings.Repeat("-", totalWidth))
	l.Infof("   |                EFS Recovery Points            |")
	l.Infof("   %s", strings.Repeat("-", totalWidth))

	// Dynamic header formatting
	headerFormat := fmt.Sprintf("   %%-%ds %%-%ds %%-%ds %%-%ds", maxVaultLen, maxCreatedLen, maxExpiryLen, maxStatusLen)
	l.Infof(headerFormat, "Vault", "Created", "Expiry", "Status")
	l.Infof("   %s", strings.Repeat("-", totalWidth))

	// Dynamic row formatting
	rowFormat := fmt.Sprintf("   %%-%ds %%-%ds %%-%ds %%-%ds", maxVaultLen, maxCreatedLen, maxExpiryLen, maxStatusLen)

	for _, rp := range rps {
		vault := rp.Vault
		if len(vault) > maxVaultLen {
			vault = vault[:maxVaultLen-3] + "..."
		}

		created := rp.Created
		if len(created) > maxCreatedLen {
			created = created[:maxCreatedLen-3] + "..."
		}

		expiry := rp.Expiry
		if len(expiry) > maxExpiryLen {
			expiry = expiry[:maxExpiryLen-3] + "..."
		}

		status := rp.Status
		if len(status) > maxStatusLen {
			status = status[:maxStatusLen-3] + "..."
		}

		l.Infof(rowFormat, vault, created, expiry, status)
	}
	l.Infof("   %s", strings.Repeat("-", totalWidth))
}
