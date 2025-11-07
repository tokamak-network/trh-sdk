package types

// StorageConfig represents a configuration that supports storage operations
type StorageConfig interface {
	GetNamespace() string
	GetChainName() string
	GetEFSFileSystemId() string
	GetHelmReleaseName() string
}

type UptimeServiceConfig struct {
	// Basic configuration
	Namespace       string
	ChainName       string
	HelmReleaseName string

	// Storage configuration
	IsPersistenceEnable bool
	EFSFileSystemId     string
}

// Ensure UptimeServiceConfig implements StorageConfig
var _ StorageConfig = (*UptimeServiceConfig)(nil)

func (s *UptimeServiceConfig) GetNamespace() string       { return s.Namespace }
func (s *UptimeServiceConfig) GetChainName() string       { return s.ChainName }
func (s *UptimeServiceConfig) GetEFSFileSystemId() string { return s.EFSFileSystemId }
func (s *UptimeServiceConfig) GetHelmReleaseName() string { return s.HelmReleaseName }
