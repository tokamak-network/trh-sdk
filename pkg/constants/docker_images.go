package constants

var DockerImageTag = map[string]struct {
	OpGethImageTag      string
	ThanosStackImageTag string
}{
	// LocalTestnet uses Sepolia L1, so it shares image tags with Testnet.
	LocalTestnet: {OpGethImageTag: "f8c04dcb", ThanosStackImageTag: "c9d8d16a"},
	Testnet:      {OpGethImageTag: "f8c04dcb", ThanosStackImageTag: "c9d8d16a"},
	Mainnet:      {OpGethImageTag: "a7c74c7e", ThanosStackImageTag: "49e37d47"},
}
