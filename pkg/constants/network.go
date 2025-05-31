package constants

const (
	LocalDevnet = "local_devnet"
	Testnet     = "testnet"
	Mainnet     = "mainnet"
)

var SupportedNetworks = map[string]bool{
	"local_devnet": true,
	"testnet":      true,
	"mainnet":      true,
}
