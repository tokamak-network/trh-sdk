package types

type DigitalOceanConfig struct {
	Token           string `json:"token"`
	Region          string `json:"region"`
	SpacesAccessKey string `json:"spaces_access_key"`
	SpacesSecretKey string `json:"-"` // never persisted to disk; re-prompted at destroy time
}

type DigitalOceanProfile struct {
	Config *DigitalOceanConfig
}
