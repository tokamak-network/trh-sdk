package constants

const (
	LocalDevnet = "local-devnet"
	Testnet     = "testnet"
	Mainnet     = "mainnet"
)

var SupportedNetworks = map[string]bool{
	"local-devnet": true,
	"testnet":      true,
	"mainnet":      true,
}
