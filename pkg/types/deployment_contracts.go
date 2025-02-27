package types

type Contracts struct {
	AddressManager                    string `json:"AddressManager"`
	AnchorStateRegistry               string `json:"AnchorStateRegistry"`
	AnchorStateRegistryProxy          string `json:"AnchorStateRegistryProxy"`
	DelayedWETH                       string `json:"DelayedWETH"`
	DelayedWETHProxy                  string `json:"DelayedWETHProxy"`
	DisputeGameFactory                string `json:"DisputeGameFactory"`
	DisputeGameFactoryProxy           string `json:"DisputeGameFactoryProxy"`
	L1CrossDomainMessenger            string `json:"L1CrossDomainMessenger"`
	L1CrossDomainMessengerProxy       string `json:"L1CrossDomainMessengerProxy"`
	L1ERC721Bridge                    string `json:"L1ERC721Bridge"`
	L1ERC721BridgeProxy               string `json:"L1ERC721BridgeProxy"`
	L1StandardBridge                  string `json:"L1StandardBridge"`
	L1StandardBridgeProxy             string `json:"L1StandardBridgeProxy"`
	L1UsdcBridge                      string `json:"L1UsdcBridge"`
	L1UsdcBridgeProxy                 string `json:"L1UsdcBridgeProxy"`
	L2OutputOracle                    string `json:"L2OutputOracle"`
	L2OutputOracleProxy               string `json:"L2OutputOracleProxy"`
	Mips                              string `json:"Mips"`
	OptimismMintableERC20Factory      string `json:"OptimismMintableERC20Factory"`
	OptimismMintableERC20FactoryProxy string `json:"OptimismMintableERC20FactoryProxy"`
	OptimismPortal                    string `json:"OptimismPortal"`
	OptimismPortal2                   string `json:"OptimismPortal2"`
	OptimismPortalProxy               string `json:"OptimismPortalProxy"`
	PermissionedDelayedWETHProxy      string `json:"PermissionedDelayedWETHProxy"`
	PreimageOracle                    string `json:"PreimageOracle"`
	ProtocolVersions                  string `json:"ProtocolVersions"`
	ProtocolVersionsProxy             string `json:"ProtocolVersionsProxy"`
	ProxyAdmin                        string `json:"ProxyAdmin"`
	SafeProxyFactory                  string `json:"SafeProxyFactory"`
	SafeSingleton                     string `json:"SafeSingleton"`
	SuperchainConfig                  string `json:"SuperchainConfig"`
	SuperchainConfigProxy             string `json:"SuperchainConfigProxy"`
	SystemConfig                      string `json:"SystemConfig"`
	SystemConfigProxy                 string `json:"SystemConfigProxy"`
	SystemOwnerSafe                   string `json:"SystemOwnerSafe"`
}
