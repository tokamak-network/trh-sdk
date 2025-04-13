package constants

var DockerImageTag = map[string]struct {
	OpGethImageTag      string
	ThanosStackImageTag string
}{
	Testnet: {OpGethImageTag: "193382ee", ThanosStackImageTag: "56ed30e3"},
	Mainnet: {OpGethImageTag: "c61a056f", ThanosStackImageTag: "011bec4a"},
}
