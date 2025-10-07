package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ListRecoveryPoints queries recovery points and returns parsed entries
func ListRecoveryPoints(ctx context.Context, region, arn, limit string) ([]types.RecoveryPoint, error) {
	if strings.TrimSpace(limit) == "" {
		limit = "20"
	}
	jsonQuery := fmt.Sprintf("reverse(sort_by(RecoveryPoints,&CreationDate))[:%s].{RecoveryPointARN:RecoveryPointArn,Vault:BackupVaultName,Created:CreationDate,Expiry:ExpiryDate,Status:Status}", limit)
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

// DisplayRecoveryPoints renders recovery points in card style format
// Only shows COMPLETED recovery points that can be used for restoration
func DisplayRecoveryPoints(l *zap.SugaredLogger, rps []types.RecoveryPoint) {
	// Filter only COMPLETED recovery points
	var completedRPs []types.RecoveryPoint
	for _, rp := range rps {
		if strings.ToUpper(rp.Status) == "COMPLETED" {
			completedRPs = append(completedRPs, rp)
		}
	}

	if len(completedRPs) == 0 {
		l.Infof("")
		l.Infof("⚠️  No available recovery points found")
		l.Infof("")
		return
	}

	l.Infof("")
	l.Infof("📦 Available Recovery Points (%d)", len(completedRPs))
	l.Infof("")

	for idx, rp := range completedRPs {
		// Calculate relative times
		createdRelative := formatRelativeTime(rp.Created)

		// Display card
		l.Infof("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		l.Infof("#%-2d", idx+1)
		l.Infof("    🔑 ARN      : %s", rp.RecoveryPointARN)
		l.Infof("    🗄️  Vault    : %s", rp.Vault)
		l.Infof("    📅 Created  : %s %s", rp.Created, createdRelative)

		// Handle expiry date - show "Never" if no expiry is set
		if strings.TrimSpace(rp.Expiry) == "" {
			l.Infof("    ⏰ Expires  : Never")
		} else {
			expiryRelative := formatRelativeTime(rp.Expiry)
			l.Infof("    ⏰ Expires  : %s %s", rp.Expiry, expiryRelative)
		}

		l.Infof("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Add spacing between cards (except for the last one)
		if idx < len(completedRPs)-1 {
			l.Infof("")
		}
	}

	l.Infof("")
}

// formatRelativeTime formats a timestamp to show relative time (e.g., "2 days ago")
func formatRelativeTime(timestamp string) string {
	if strings.TrimSpace(timestamp) == "" {
		return ""
	}

	// Try parsing with different time formats
	var t time.Time
	var err error

	// ISO8601 formats to try
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000000-07:00",
		"2006-01-02T15:04:05.000000Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		t, err = time.Parse(format, timestamp)
		if err == nil {
			break
		}
	}

	if err != nil {
		return ""
	}

	now := time.Now()
	duration := now.Sub(t)

	// If time is in the future
	if duration < 0 {
		duration = -duration
		days := int(duration.Hours() / 24)
		hours := int(duration.Hours()) % 24

		if days > 0 {
			return fmt.Sprintf("(in %d days)", days)
		} else if hours > 0 {
			return fmt.Sprintf("(in %d hours)", hours)
		} else {
			minutes := int(duration.Minutes())
			return fmt.Sprintf("(in %d minutes)", minutes)
		}
	}

	// If time is in the past
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24

	if days > 0 {
		return fmt.Sprintf("(%d days ago)", days)
	} else if hours > 0 {
		return fmt.Sprintf("(%d hours ago)", hours)
	} else {
		minutes := int(duration.Minutes())
		if minutes <= 0 {
			return "(just now)"
		}
		return fmt.Sprintf("(%d minutes ago)", minutes)
	}
}
