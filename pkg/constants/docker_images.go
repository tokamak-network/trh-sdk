package constants

var DockerImageTag = map[string]struct {
	OpGethImageTag      string
	ThanosStackImageTag string
	DRBNodeImageTag     string
}{
	Testnet: {OpGethImageTag: "nightly", ThanosStackImageTag: "af052710", DRBNodeImageTag: "sha-8c37f63"},
	Mainnet: {OpGethImageTag: "nightly", ThanosStackImageTag: "af052710", DRBNodeImageTag: "sha-8c37f63"},
}
