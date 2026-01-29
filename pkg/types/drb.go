package types

// LeaderNodeInput represents the input for leader node configuration
type LeaderNodeInput struct {
	PrivateKey string `json:"private_key"`
}

// DeployDRBInput represents the input for DRB deployment
type DeployDRBInput struct {
	RPC             string             `json:"rpc"`
	ChainID         uint64             `json:"chain_id"`
	PrivateKey      string             `json:"private_key"`
	LeaderNodeInput *LeaderNodeInput   `json:"leader_node_input"`
	DatabaseConfig  *DRBDatabaseConfig `json:"database_config"`
}

// DeployDRBContractsOutput represents the output from DRB contract deployment
type DeployDRBContractsOutput struct {
	ContractAddress          string `json:"contract_address"` // CommitReveal2L2 address
	ContractName             string `json:"contract_name"`    // CommitReveal2L2
	ChainID                  uint64 `json:"chain_id"`
	ConsumerExampleV2Address string `json:"consumer_example_v2_address,omitempty"` // ConsumerExampleV2 address (optional)
}

// DeployDRBApplicationOutput represents the output from DRB application deployment
type DeployDRBApplicationOutput struct {
	LeaderNodeURL string `json:"leader_node_url"`
}

// DeployDRBOutput represents the complete output from DRB deployment
type DeployDRBOutput struct {
	DeployDRBContractsOutput   *DeployDRBContractsOutput   `json:"deploy_drb_contracts_output"`
	DeployDRBApplicationOutput *DeployDRBApplicationOutput `json:"deploy_drb_application_output"`
}

// DRBDatabaseConfig represents the database configuration for DRB nodes
type DRBDatabaseConfig struct {
	Type          string `json:"type"` // "rds" or "local"
	ConnectionURL string `json:"connection_url"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	DatabaseName  string `json:"database_name"`
}

// DRBHelmValues represents the Helm values structure for DRB nodes
type DRBHelmValues struct {
	NameOverride     string
	FullnameOverride string
	Volume           struct {
		Enabled          bool
		ExistingClaim    string
		StorageClassName string
		Size             string
	}
}

// DRBConfig represents the configuration for DRB deployment
type DRBConfig struct {
	// Basic configuration
	Namespace       string
	ChainName       string
	HelmReleaseName string

	// Storage configuration
	IsPersistenceEnable bool
	EFSFileSystemId     string
}

// Ensure DRBConfig implements StorageConfig
var _ StorageConfig = (*DRBConfig)(nil)

func (s *DRBConfig) GetNamespace() string       { return s.Namespace }
func (s *DRBConfig) GetChainName() string       { return s.ChainName }
func (s *DRBConfig) GetEFSFileSystemId() string { return s.EFSFileSystemId }
func (s *DRBConfig) GetHelmReleaseName() string { return s.HelmReleaseName }

// DRBInfrastructureConfig represents the infrastructure configuration for DRB cluster
type DRBInfrastructureConfig struct {
	ClusterName string `json:"cluster_name"`
	Namespace   string `json:"namespace"`
	VpcID       string `json:"vpc_id"`
	Region      string `json:"region"`
}

// DRBLeaderInfo represents the leader node connection information
type DRBLeaderInfo struct {
	LeaderURL                string `json:"leader_url"`
	LeaderIP                 string `json:"leader_ip"`
	LeaderPort               int    `json:"leader_port"`
	LeaderPeerID             string `json:"leader_peer_id"`
	LeaderEOA                string `json:"leader_eoa"`
	CommitReveal2L2Address   string `json:"commit_reveal2_l2_address"`             // CommitReveal2L2 contract address
	ConsumerExampleV2Address string `json:"consumer_example_v2_address,omitempty"` // ConsumerExampleV2 contract address (optional)
	ChainID                  uint64 `json:"chain_id"`
	RPCURL                   string `json:"rpc_url"`
	DeploymentTimestamp      string `json:"deployment_timestamp"`
	ClusterName              string `json:"cluster_name"`
	Namespace                string `json:"namespace"`
}

// DRBRegularNodeInput represents the input for a regular node setup
type DRBRegularNodeInput struct {
	EnvFilePath      string `json:"env_file_path,omitempty"`
	EnvFileContent   string `json:"-"` // cached .env content when loaded from file, to avoid re-reading
	LeaderIP         string `json:"leader_ip"`
	LeaderPort       int    `json:"leader_port"`
	LeaderPeerID     string `json:"leader_peer_id"`
	LeaderEOA        string `json:"leader_eoa"` 
	NodePort         int    `json:"node_port"`
	EOAPrivateKey    string `json:"eoa_private_key"`
	NodeType         string `json:"node_type"`
	DatabaseConfig *DRBDatabaseConfig `json:"database_config"`
	DrbNodeImage   string             `json:"drb_node_image"`

	ChainID         string `json:"chain_id"`          // e.g. "11155111"
	EthRpcUrls      string `json:"eth_rpc_urls"`      // comma-separated RPC URLs
	ContractAddress string `json:"contract_address"`  // CommitReveal2L2 contract address

	Region         string `json:"region"`
	VpcID          string `json:"vpc_id,omitempty"`
	SubnetID       string `json:"subnet_id,omitempty"`
	InstanceType string `json:"instance_type"`
	KeyPairName  string `json:"key_pair_name"`
	AmiID        string `json:"ami_id,omitempty"`
	InstanceName string `json:"instance_name"`
}
