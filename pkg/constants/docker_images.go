package constants

var DockerImageTag = map[string]struct {
	OpGethImageTag      string
	ThanosStackImageTag string
	DRBNodeImageTag     string
}{
	Testnet: {OpGethImageTag: "nightly", ThanosStackImageTag: "nightly-b684fda0", DRBNodeImageTag: "latest"},
	Mainnet: {OpGethImageTag: "nightly", ThanosStackImageTag: "nightly-b684fda0", DRBNodeImageTag: "latest"},
}
