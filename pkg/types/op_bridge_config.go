package types

type OpBridgeConfig struct {
	OpBridge struct {
		Env struct {
			L1ChainName                   string `yaml:"l1_chain_name"`
			L1ChainID                     string `yaml:"l1_chain_id"`
			L1RPC                         string `yaml:"l1_rpc"`
			L1NativeCurrencyName          string `yaml:"l1_native_currency_name"`
			L1NativeCurrencySymbol        string `yaml:"l1_native_currency_symbol"`
			L1NativeCurrencyDecimals      int    `yaml:"l1_native_currency_decimals"`
			L1BlockExplorer               string `yaml:"l1_block_explorer"`
			L2ChainName                   string `yaml:"l2_chain_name"`
			L2ChainID                     string `yaml:"l2_chain_id"`
			L2RPC                         string `yaml:"l2_rpc"`
			L2NativeCurrencyName          string `yaml:"l2_native_currency_name"`
			L2NativeCurrencySymbol        string `yaml:"l2_native_currency_symbol"`
			L2NativeCurrencyDecimals      int    `yaml:"l2_native_currency_decimals"`
			NativeTokenL1Address          string `yaml:"native_token_l1_address"`
			L1USDCAddress                 string `yaml:"l1_usdc_address"`
			L1USDTAddress                 string `yaml:"l1_usdt_address"`
			L2USDTAddress                 string `yaml:"l2_usdt_address"`
			StandardBridgeAddress         string `yaml:"standard_bridge_address"`
			AddressManagerAddress         string `yaml:"address_manager_address"`
			L1CrossDomainMessengerAddress string `yaml:"l1_cross_domain_messenger_address"`
			OptimismPortalAddress         string `yaml:"optimism_portal_address"`
			L2OutputOracleAddress         string `yaml:"l2_output_oracle_address"`
			L1USDCBridgeAddress           string `yaml:"l1_usdc_bridge_address"`
			DisputeGameFactoryAddress     string `yaml:"dispute_game_factory_address"`
		} `yaml:"env"`
		Ingress struct {
			Enabled     bool              `yaml:"enabled"`
			ClassName   string            `yaml:"className"`
			Annotations map[string]string `yaml:"annotations"`
			TLS         struct {
				Enabled bool `yaml:"enabled"`
			} `yaml:"tls"`
		} `yaml:"ingress"`
	} `yaml:"op_bridge"`
}
