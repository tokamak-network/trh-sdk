package constants

var DockerImageTag = map[string]struct {
	OpGethImageTag      string
	ThanosStackImageTag string
	DRBNodeImageTag     string
}{
	LocalTestnet: {OpGethImageTag: "nightly", ThanosStackImageTag: "nightly-b684fda0", DRBNodeImageTag: "sha-8c37f63"},
	Testnet:      {OpGethImageTag: "nightly", ThanosStackImageTag: "nightly-b684fda0", DRBNodeImageTag: "sha-8c37f63"},
	Mainnet:      {OpGethImageTag: "nightly", ThanosStackImageTag: "nightly-b684fda0", DRBNodeImageTag: "sha-8c37f63"},
}
