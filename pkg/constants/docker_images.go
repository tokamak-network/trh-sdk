package constants

var DockerImageTag = map[string]struct {
	OpGethImageRepo     string // Execution client image repository override (empty = default op-geth)
	OpGethImageTag      string
	ThanosStackImageTag string
}{
	Testnet: {OpGethImageRepo: "tokamaknetwork/py-ethclient", OpGethImageTag: "latest", ThanosStackImageTag: "80a6da51"},
	Mainnet: {OpGethImageRepo: "", OpGethImageTag: "a7c74c7e", ThanosStackImageTag: "49e37d47"},
}
