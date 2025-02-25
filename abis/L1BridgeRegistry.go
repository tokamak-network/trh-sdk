package abis


const L1BridgeRegistryABI = `[
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "rollupConfig",
				"type": "address"
			},
			{
				"internalType": "uint8",
				"name": "_type",
				"type": "uint8"
			},
			{
				"internalType": "address",
				"name": "_l2TON",
				"type": "address"
			},
			{
				"internalType": "string",
				"name": "_name",
				"type": "string"
			}
		],
		"name": "registerRollupConfig",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`