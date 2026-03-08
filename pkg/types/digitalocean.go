package types

type DigitalOceanConfig struct {
	Token           string `json:"token"`
	Region          string `json:"region"`
	SpacesAccessKey string `json:"spaces_access_key"`
	SpacesSecretKey string `json:"spaces_secret_key"`
}

type DigitalOceanProfile struct {
	Config *DigitalOceanConfig
}
