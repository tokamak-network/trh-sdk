package constants

var DockerImageTag = map[string]struct {
	OpGethImageTag      string
	ThanosStackImageTag string
}{
	Testnet: {OpGethImageTag: "f8c04dcb", ThanosStackImageTag: "80a6da51"},
	Mainnet: {OpGethImageTag: "a7c74c7e", ThanosStackImageTag: "49e37d47"},
}
