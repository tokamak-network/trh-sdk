package types

// TODO: Check and update these default info with team if required
var MetadataGenericInfo = MetadataInfo{
	Chain: ChainInfo{
		Description: "Example rollup deployed with TRH SDK",
		Logo:        "https://example.com/logo.png",
		Website:     "https://example-l2.com",
	},
	Bridge: BridgeInfo{
		Name: "Example Bridge",
	},
	Explorer: ExplorerInfo{
		Name: "Example Explorer",
	},
	Support: SupportResources{
		StatusPageUrl:     "https://status.example-l2.com",
		SupportContactUrl: "https://discord.gg/example-support",
		DocumentationUrl:  "https://docs.example-l2.com",
		CommunityUrl:      "https://t.me/example_community",
		HelpCenterUrl:     "https://help.example-l2.com",
		AnnouncementUrl:   "https://twitter.com/example_l2",
	},
}

type RollupMetadata struct {
	L1ChainId   uint64 `json:"l1ChainId"`
	L2ChainId   uint64 `json:"l2ChainId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Logo        string `json:"logo,omitempty"`
	Website     string `json:"website,omitempty"`

	RollupType string `json:"rollupType"`
	Stack      Stack  `json:"stack"`

	RpcUrl string `json:"rpcUrl"`
	WsUrl  string `json:"wsUrl"`

	NativeToken NativeToken `json:"nativeToken"`

	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	LastUpdated string `json:"lastUpdated"`

	L1Contracts      L1Contracts      `json:"l1Contracts"`
	L2Contracts      L2Contracts      `json:"l2Contracts"`
	Bridges          []Bridge         `json:"bridges"`
	Explorers        []Explorer       `json:"explorers"`
	Sequencer        SequencerInfo    `json:"sequencer"`
	Staking          Staking          `json:"staking"`
	NetworkConfig    NetworkConfig    `json:"networkConfig"`
	WithdrawalConfig WithdrawalConfig `json:"withdrawalConfig"`
	SupportResources SupportResources `json:"supportResources"`
	Metadata         Metadata         `json:"metadata"`
}

type Stack struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Commit        string `json:"commit,omitempty"`        // Optional
	Documentation string `json:"documentation,omitempty"` // Optional
}

type NativeToken struct {
	Type        string `json:"type"`
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Decimals    int    `json:"decimals"`
	L1Address   string `json:"l1Address"`
	LogoUrl     string `json:"logoUrl,omitempty"`     // Optional
	CoingeckoId string `json:"coingeckoId,omitempty"` // Optional
}

// Updated L1Contracts to match the raw data structure
type L1Contracts struct {
	ProxyAdmin                   string `json:"ProxyAdmin"`
	SystemConfig                 string `json:"SystemConfig"`
	AddressManager               string `json:"AddressManager"`
	SuperchainConfig             string `json:"SuperchainConfig"`
	DisputeGameFactory           string `json:"DisputeGameFactory"`
	L1CrossDomainMessenger       string `json:"L1CrossDomainMessenger"`
	L1ERC721Bridge               string `json:"L1ERC721Bridge"`
	L1StandardBridge             string `json:"L1StandardBridge"`
	OptimismMintableERC20Factory string `json:"OptimismMintableERC20Factory"`
	OptimismPortal               string `json:"OptimismPortal"`
	AnchorStateRegistry          string `json:"AnchorStateRegistry"`
	DelayedWETH                  string `json:"DelayedWETH"`
	L1UsdcBridge                 string `json:"L1UsdcBridge,omitempty"`
	L2OutputOracle               string `json:"L2OutputOracle"`
	Mips                         string `json:"Mips"`
	PermissionedDelayedWETH      string `json:"PermissionedDelayedWETH"`
	PreimageOracle               string `json:"PreimageOracle"`
	ProtocolVersions             string `json:"ProtocolVersions"`
	SafeProxyFactory             string `json:"SafeProxyFactory"`
	SafeSingleton                string `json:"SafeSingleton"`
	SystemOwnerSafe              string `json:"SystemOwnerSafe"`
}

type L2Contracts struct {
	NativeToken                   string `json:"NativeToken"`
	WETH                          string `json:"WETH"`
	L2ToL1MessagePasser           string `json:"L2ToL1MessagePasser"`
	DeployerWhitelist             string `json:"DeployerWhitelist"`
	L2CrossDomainMessenger        string `json:"L2CrossDomainMessenger"`
	GasPriceOracle                string `json:"GasPriceOracle"`
	L2StandardBridge              string `json:"L2StandardBridge"`
	SequencerFeeVault             string `json:"SequencerFeeVault"`
	OptimismMintableERC20Factory  string `json:"OptimismMintableERC20Factory"`
	L1BlockNumber                 string `json:"L1BlockNumber"`
	L1Block                       string `json:"L1Block"`
	GovernanceToken               string `json:"GovernanceToken"`
	LegacyMessagePasser           string `json:"LegacyMessagePasser"`
	L2ERC721Bridge                string `json:"L2ERC721Bridge"`
	OptimismMintableERC721Factory string `json:"OptimismMintableERC721Factory"`
	ProxyAdmin                    string `json:"ProxyAdmin"`
	BaseFeeVault                  string `json:"BaseFeeVault"`
	L1FeeVault                    string `json:"L1FeeVault"`
	ETH                           string `json:"ETH"`
}

type Bridge struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	URL             string  `json:"url"`
	Status          string  `json:"status"`
	SupportedTokens []Token `json:"supportedTokens"`
}

type Token struct {
	Symbol        string `json:"symbol"`
	L1Address     string `json:"l1Address"`
	L2Address     string `json:"l2Address"`
	Decimals      int    `json:"decimals"`
	IsNativeToken bool   `json:"isNativeToken,omitempty"`
	IsWrappedETH  bool   `json:"isWrappedETH,omitempty"`
}

type Explorer struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Type   string `json:"type"`
	Status string `json:"status"`
	ApiUrl string `json:"apiUrl,omitempty"` // Optional
}

type SequencerInfo struct {
	Address           string `json:"address"`
	BatcherAddress    string `json:"batcherAddress"`
	ProposerAddress   string `json:"proposerAddress"`
	AggregatorAddress string `json:"aggregatorAddress,omitempty"` // Optional
	TrustedSequencer  bool   `json:"trustedSequencer,omitempty"`  // Optional
}

type Staking struct {
	IsCandidate           bool   `json:"isCandidate"`
	CandidateRegisteredAt string `json:"candidateRegisteredAt,omitempty"` // Optional
	CandidateStatus       string `json:"candidateStatus,omitempty"`       // Optional
	RegistrationTxHash    string `json:"registrationTxHash,omitempty"`    // Optional
	CandidateAddress      string `json:"candidateAddress,omitempty"`      // Optional
	RollupConfigAddress   string `json:"rollupConfigAddress,omitempty"`   // Optional
	StakingServiceName    string `json:"stakingServiceName,omitempty"`    // Optional
}

type NetworkConfig struct {
	BlockTime         int    `json:"blockTime"`
	GasLimit          string `json:"gasLimit"`
	BaseFeePerGas     string `json:"baseFeePerGas"`
	PriorityFeePerGas string `json:"priorityFeePerGas"`
}

// Updated WithdrawalConfig to match the raw data structure
type WithdrawalConfig struct {
	ChallengePeriod          int                    `json:"challengePeriod"`
	ExpectedWithdrawalDelay  int                    `json:"expectedWithdrawalDelay"`
	MonitoringInfo           MonitoringInfoMetadata `json:"monitoringInfo"`
	BatchSubmissionFrequency int                    `json:"batchSubmissionFrequency"`
	OutputRootFrequency      int                    `json:"outputRootFrequency"`
}

type MonitoringInfoMetadata struct {
	L2OutputOracleAddress    string `json:"l2OutputOracleAddress"`
	OutputProposedEventTopic string `json:"outputProposedEventTopic"`
}

type SupportResources struct {
	StatusPageUrl     string `json:"statusPageUrl"`
	SupportContactUrl string `json:"supportContactUrl"`
	DocumentationUrl  string `json:"documentationUrl"`
	CommunityUrl      string `json:"communityUrl"`
	HelpCenterUrl     string `json:"helpCenterUrl"`
	AnnouncementUrl   string `json:"announcementUrl"`
}

type Metadata struct {
	Version   string `json:"version"`
	Signature string `json:"signature"`
	SignedBy  string `json:"signedBy"`
}

// GitHubPR represents the structure for creating a GitHub PR
type GitHubPR struct {
	Title string `json:"title"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Body  string `json:"body"`
}

// GitHubCredentials holds GitHub authentication details
type GitHubCredentials struct {
	Username string
	Token    string
	Email    string
}

type ChainInfo struct {
	Description string `json:"description"`
	Logo        string `json:"logo"`
	Website     string `json:"website"`
}

type BridgeInfo struct {
	Name string `json:"name"`
}

type ExplorerInfo struct {
	Name string `json:"name"`
}

type MetadataInfo struct {
	Chain    ChainInfo        `json:"chain"`
	Bridge   BridgeInfo       `json:"bridge"`
	Explorer ExplorerInfo     `json:"explorer"`
	Support  SupportResources `json:"supportResources"`
}
