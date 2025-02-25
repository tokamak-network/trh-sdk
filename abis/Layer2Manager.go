package abis

const Layer2ManagerABI = `[
	{
    "inputs": [
      { "internalType": "address", "name": "rollupConfig", "type": "address" },
      { "internalType": "uint256", "name": "amount", "type": "uint256" },
      { "internalType": "bool", "name": "flagTon", "type": "bool" },
      { "internalType": "string", "name": "memo", "type": "string" }
    ],
    "name": "registerCandidateAddOn",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]`