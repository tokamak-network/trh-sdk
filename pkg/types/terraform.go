package types

type TerraformEnvConfig struct {
	ThanosStackName   string
	AwsRegion         string
	BackendBucketName string
	SequencerKey      string
	BatcherKey        string
	ProposerKey       string
	ChallengerKey     string
	Azs               string
	VpcCidr           string
	VpcName           string
	EksClusterAdmins  string
	GenesisFilePath   string
	RollupFilePath    string
	PrestateFilePath  string
	PrestateHash      string
	DeploymentsPath   string
	L1RpcUrl          string
	L1RpcProvider     string
	L1BeaconUrl       string
}
