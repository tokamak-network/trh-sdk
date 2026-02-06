package types

type ThanosLogsHelmValues struct {
	NameOverride     string
	FullnameOverride string
	Storage          struct {
		BucketNames map[string]interface{}
		S3          struct {
			Endpoint        string
			Region          string
			SecretAccessKey string
			AccessKeyId     string
		}
	}
}

type ThanosLogsConfig struct {
	Namespace       string
	ChainName       string
	HelmReleaseName string
	GrafanaPassword string
}
