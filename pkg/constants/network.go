package constants

const (
	LocalDevnet  = "local_devnet"
	LocalTestnet = "LocalTestnet"
	Testnet      = "testnet"
	Mainnet      = "mainnet"
)

var SupportedNetworks = map[string]bool{
	"local_devnet": true,
	"LocalTestnet": true,
	"testnet":      true,
	"mainnet":      true,
}
