package types

type UpdateTerraformEnvConfig struct {
	L1RpcUrl            string
	L1RpcProvider       string
	L1BeaconUrl         string
	OpGethImageTag      string
	ThanosStackImageTag string
}

type TerraformEnvConfig struct {
	Namespace           string
	AwsRegion           string
	BackendBucketName   string
	SequencerKey        string
	BatcherKey          string
	ProposerKey         string
	ChallengerKey       string
	Azs                 []string
	VpcCidr             string
	VpcName             string
	EksClusterAdmins    string
	GenesisFilePath     string
	RollupFilePath      string
	PrestateFilePath    string
	PrestateHash        string
	DeploymentFilePath  string
	L1RpcUrl            string
	L1RpcProvider       string
	L1BeaconUrl         string
	OpGethImageTag      string
	ThanosStackImageTag string
	MaxChannelDuration  uint64
}

type BlockExplorerEnvs struct {
	BlockExplorerDatabaseName     string
	BlockExplorerDatabasePassword string
	BlockExplorerDatabaseUserName string
	VpcId                         string
}

type ThanosStackTerraformState struct {
	Version          int    `json:"version"`
	TerraformVersion string `json:"terraform_version"`
	Backend          struct {
		Type   string `json:"type"`
		Config struct {
			AccessKey                      *string `json:"access_key"`
			Acl                            *string `json:"acl"`
			AllowedAccountIds              *string `json:"allowed_account_ids"`
			AssumeRole                     *string `json:"assume_role"`
			AssumeRoleWithWebIdentity      *string `json:"assume_role_with_web_identity"`
			Bucket                         string  `json:"bucket"`
			CustomCaBundle                 *string `json:"custom_ca_bundle"`
			DynamodbEndpoint               *string `json:"dynamodb_endpoint"`
			DynamodbTable                  string  `json:"dynamodb_table"`
			Ec2MetadataServiceEndpoint     *string `json:"ec2_metadata_service_endpoint"`
			Ec2MetadataServiceEndpointMode *string `json:"ec2_metadata_service_endpoint_mode"`
			Encrypt                        bool    `json:"encrypt"`
			Endpoint                       *string `json:"endpoint"`
			Endpoints                      *string `json:"endpoints"`
			ForbiddenAccountIds            *string `json:"forbidden_account_ids"`
			ForcePathStyle                 *string `json:"force_path_style"`
			HttpProxy                      *string `json:"http_proxy"`
			HttpsProxy                     *string `json:"https_proxy"`
			IamEndpoint                    *string `json:"iam_endpoint"`
			Insecure                       *string `json:"insecure"`
			Key                            string  `json:"key"`
			KmsKeyId                       *string `json:"kms_key_id"`
			MaxRetries                     *string `json:"max_retries"`
			NoProxy                        *string `json:"no_proxy"`
			Profile                        *string `json:"profile"`
			Region                         string  `json:"region"`
			RetryMode                      *string `json:"retry_mode"`
			SecretKey                      *string `json:"secret_key"`
			SharedConfigFiles              *string `json:"shared_config_files"`
			SharedCredentialsFile          *string `json:"shared_credentials_file"`
			SharedCredentialsFiles         *string `json:"shared_credentials_files"`
			SkipCredentialsValidation      *string `json:"skip_credentials_validation"`
			SkipMetadataApiCheck           *string `json:"skip_metadata_api_check"`
			SkipRegionValidation           *string `json:"skip_region_validation"`
			SkipRequestingAccountId        *string `json:"skip_requesting_account_id"`
			SkipS3Checksum                 *string `json:"skip_s3_checksum"`
			SseCustomerKey                 *string `json:"sse_customer_key"`
			StsEndpoint                    *string `json:"sts_endpoint"`
			StsRegion                      *string `json:"sts_region"`
			Token                          *string `json:"token"`
			UseDualstackEndpoint           *string `json:"use_dualstack_endpoint"`
			UseFipsEndpoint                *string `json:"use_fips_endpoint"`
			UseLockfile                    *string `json:"use_lockfile"`
			UsePathStyle                   *string `json:"use_path_style"`
			WorkspaceKeyPrefix             *string `json:"workspace_key_prefix"`
		} `json:"config"`
		Hash int `json:"hash"`
	} `json:"backend"`
}
