package constants

var DockerImageTag = map[string]struct {
	OpGethImageTag      string
	ThanosStackImageTag string
	DRBNodeImageTag     string
}{
	LocalTestnet: {OpGethImageTag: "nightly", ThanosStackImageTag: "nightly-b684fda0", DRBNodeImageTag: "latest"},
	Testnet:      {OpGethImageTag: "nightly", ThanosStackImageTag: "nightly-b684fda0", DRBNodeImageTag: "latest"},
	Mainnet:      {OpGethImageTag: "nightly", ThanosStackImageTag: "nightly-b684fda0", DRBNodeImageTag: "latest"},
}
