package types

// StorageConfig represents a configuration that supports storage operations
type StorageConfig interface {
	GetNamespace() string
	GetChainName() string
	GetEFSFileSystemId() string
	GetHelmReleaseName() string
}

type UptimeKumaConfig struct {
	// Basic configuration
	Namespace       string
	ChainName       string
	HelmReleaseName	string

	// Storage configuration
	IsPersistenceEnable bool
	EFSFileSystemId     string
}


// Ensure UptimeKumaConfig implements StorageConfig
var _ StorageConfig = (*UptimeKumaConfig)(nil)

func (s *UptimeKumaConfig) GetNamespace() string       { return s.Namespace }
func (s *UptimeKumaConfig) GetChainName() string         { return s.ChainName }
func (s *UptimeKumaConfig) GetEFSFileSystemId() string  { return s.EFSFileSystemId }
func (s *UptimeKumaConfig) GetHelmReleaseName() string   { return s.HelmReleaseName }