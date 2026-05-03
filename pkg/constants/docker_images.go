package constants

var DockerImageTag = map[string]struct {
	OpGethImageTag      string
	ThanosStackImageTag string
	DRBNodeImageTag     string
}{
	Testnet: {OpGethImageTag: "nightly", ThanosStackImageTag: "d48fda92", DRBNodeImageTag: "sha-8c37f63"},
	Mainnet: {OpGethImageTag: "nightly", ThanosStackImageTag: "d48fda92", DRBNodeImageTag: "sha-8c37f63"},
}
