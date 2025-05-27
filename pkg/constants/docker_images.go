package constants

var DockerImageTag = map[string]struct {
	OpGethImageTag      string
	ThanosStackImageTag string
}{
	Testnet: {OpGethImageTag: "193382ee", ThanosStackImageTag: "56ed30e3"},
	Mainnet: {OpGethImageTag: "a7c74c7e", ThanosStackImageTag: "49e37d47"},
}
