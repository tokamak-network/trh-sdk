package constants

var DockerImageTag = map[string]struct {
	OpGethImageTag      string
	ThanosStackImageTag string
}{
	Testnet: {OpGethImageTag: "latest", ThanosStackImageTag: "latest"},
	Mainnet: {OpGethImageTag: "latest", ThanosStackImageTag: "latest"},
}
