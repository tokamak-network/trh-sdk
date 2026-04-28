package thanos

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// maybeFundAAAdmin injects genesis balance for the AA admin wallet when AA setup is needed.
// AA setup is required for all presets when fee token is not TON.
// Funded amount: 2 × DefaultEntryPointDeposit (1× for EntryPoint deposit, 1× for setup gas).
func maybeFundAAAdmin(genesisPath, preset, feeToken, adminPrivKey string) error {
	if !constants.NeedsAASetup(preset, feeToken) {
		return nil
	}
	adminAddr, err := utils.GetAddressFromPrivateKey(adminPrivKey)
	if err != nil {
		return fmt.Errorf("derive admin address for AA genesis funding: %w", err)
	}
	return patchGenesisWithAAAdminFunding(genesisPath, adminAddr)
}

// patchGenesisWithAAAdminFunding sets the AA admin's genesis balance to
// 2 × DefaultEntryPointDeposit wei, preserving all other alloc entries and fields.
func patchGenesisWithAAAdminFunding(genesisPath string, adminAddr common.Address) error {
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		return fmt.Errorf("failed to read genesis file: %w", err)
	}

	var genesis map[string]json.RawMessage
	if err := json.Unmarshal(data, &genesis); err != nil {
		return fmt.Errorf("failed to parse genesis JSON: %w", err)
	}

	var alloc map[string]json.RawMessage
	if err := json.Unmarshal(genesis["alloc"], &alloc); err != nil {
		return fmt.Errorf("failed to parse alloc section: %w", err)
	}

	has0xPrefix := false
	for key := range alloc {
		if strings.HasPrefix(key, "0x") || strings.HasPrefix(key, "0X") {
			has0xPrefix = true
		}
		break
	}

	addrHex := strings.ToLower(adminAddr.Hex())
	if !has0xPrefix {
		addrHex = strings.TrimPrefix(addrHex, "0x")
	}

	// 2 × DefaultEntryPointDeposit = 2e18 wei
	funding := new(big.Int).Mul(constants.DefaultEntryPointDeposit, big.NewInt(2))
	balanceHex := json.RawMessage(`"0x` + funding.Text(16) + `"`)

	if existing, ok := alloc[addrHex]; ok {
		var entry map[string]json.RawMessage
		if err := json.Unmarshal(existing, &entry); err != nil {
			return fmt.Errorf("failed to parse existing AA admin alloc: %w", err)
		}
		entry["balance"] = balanceHex
		updated, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal updated AA admin alloc: %w", err)
		}
		alloc[addrHex] = updated
	} else {
		newEntry, err := json.Marshal(map[string]interface{}{
			"balance": "0x" + funding.Text(16),
		})
		if err != nil {
			return fmt.Errorf("failed to marshal AA admin alloc: %w", err)
		}
		alloc[addrHex] = newEntry
	}

	allocJSON, err := json.Marshal(alloc)
	if err != nil {
		return err
	}
	genesis["alloc"] = allocJSON

	output, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(genesisPath, output, 0644)
}
