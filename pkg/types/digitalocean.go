package types

type DigitalOceanConfig struct {
	Token  string `json:"token"`
	Region string `json:"region"`
}

type DigitalOceanProfile struct {
	Config *DigitalOceanConfig
}
