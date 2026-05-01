package types

// Backup related constants
const (
	DefaultBackupRetentionDays = 0
	TimeFormatISO8601          = "2006-01-02T15:04:05.000000-07:00"
	TimeFormatISO8601KST       = "2006-01-02T15:04:05.000000+09:00"
)

// BackupStatusInfo represents backup status information
type BackupStatusInfo struct {
	Region              string
	Namespace           string
	AccountID           string
	EFSID               string
	ARN                 string
	IsProtected         bool
	LatestRecoveryPoint string
	ExpectedExpiryDate  string
	BackupVaults        []string
	BackupSchedule      string
	NextBackupTime      string
}

// BackupSnapshotInfo represents backup snapshot information
type BackupSnapshotInfo struct {
	Region    string
	Namespace string
	EFSID     string
	ARN       string
	JobID     string
	Status    string
}

// BackupListInfo represents backup list information
type BackupListInfo struct {
	Region         string
	Namespace      string
	EFSID          string
	ResourceARN    string // EFS file system ARN
	Limit          string
	RecoveryPoints []RecoveryPoint
}

// RecoveryPoint represents a single recovery point
type RecoveryPoint struct {
	RecoveryPointARN string // Recovery point ARN for restoration
	Vault            string
	Created          string
	Expiry           string
	Status           string
}

// BackupRestoreInfo represents backup restore information
type BackupRestoreInfo struct {
	Region           string
	Namespace        string
	EFSID            string
	ARN              string
	RecoveryPointARN string
	NewEFSID         string
	JobID            string
	Status           string
	SuggestedEFSID   string
	SuggestedPVCs    string
	SuggestedSTSs    string
}

// BackupAttachInfo represents backup attach information
type BackupAttachInfo struct {
	Region    string
	Namespace string
	EFSID     string
	PVCs      []string
	STSs      []string
	Status    string
}

// BackupConfigInfo represents backup configuration information
type BackupConfigInfo struct {
	Region    string
	Namespace string
	Daily     string
	Keep      string
	Reset     bool
	Status    string
}
