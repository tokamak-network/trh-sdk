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

// patchGenesisWithAAAdminFunding ensures the AA admin's genesis balance is at least
// 2 × DefaultEntryPointDeposit wei, preserving all other alloc entries and fields.
// If the admin already has a higher balance, it is not lowered.
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
	if alloc == nil {
		alloc = make(map[string]json.RawMessage)
	}

	// 2 × DefaultEntryPointDeposit = 2e18 wei
	minimum := new(big.Int).Mul(constants.DefaultEntryPointDeposit, big.NewInt(2))

	// Find existing entry using case/prefix-insensitive key lookup.
	// Genesis alloc keys vary: lowercase 0x, checksum (EIP-55), uppercase 0X, no prefix.
	// We match by normalizing both sides and reuse the original key to avoid duplicates.
	existingKey, found := findAllocKey(alloc, adminAddr)
	if found {
		var entry map[string]json.RawMessage
		if err := json.Unmarshal(alloc[existingKey], &entry); err != nil {
			return fmt.Errorf("failed to parse existing AA admin alloc: %w", err)
		}
		entry["balance"] = ensureMinBalance(entry["balance"], minimum)
		updated, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal updated AA admin alloc: %w", err)
		}
		alloc[existingKey] = updated
	} else {
		addrKey := newAllocKey(alloc, adminAddr)
		newEntry, err := json.Marshal(map[string]interface{}{
			"balance": "0x" + minimum.Text(16),
		})
		if err != nil {
			return fmt.Errorf("failed to marshal AA admin alloc: %w", err)
		}
		alloc[addrKey] = newEntry
	}

	allocJSON, err := json.Marshal(alloc)
	if err != nil {
		return fmt.Errorf("failed to marshal alloc: %w", err)
	}
	genesis["alloc"] = allocJSON

	output, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal genesis: %w", err)
	}
	return os.WriteFile(genesisPath, output, 0644)
}

// findAllocKey returns the existing key in alloc that matches addr (case/prefix-insensitive).
func findAllocKey(alloc map[string]json.RawMessage, addr common.Address) (string, bool) {
	target := strings.ToLower(strings.TrimPrefix(addr.Hex(), "0x"))
	for key := range alloc {
		normalized := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(key, "0x"), "0X"))
		if normalized == target {
			return key, true
		}
	}
	return "", false
}

// newAllocKey returns a new key for addr, matching the prefix convention of the existing alloc.
// Defaults to "0x"-prefixed lowercase if alloc is empty.
func newAllocKey(alloc map[string]json.RawMessage, addr common.Address) string {
	usesPrefix := true
	for key := range alloc {
		usesPrefix = strings.HasPrefix(key, "0x") || strings.HasPrefix(key, "0X")
		break
	}
	lower := strings.ToLower(addr.Hex())
	if !usesPrefix {
		return strings.TrimPrefix(lower, "0x")
	}
	return lower
}

// ensureMinBalance returns a JSON-quoted hex balance string that is at least minimum.
// If existing is nil/unparseable or smaller than minimum, returns minimum.
// If existing already meets or exceeds minimum, it is returned unchanged.
func ensureMinBalance(existing json.RawMessage, minimum *big.Int) json.RawMessage {
	fallback := json.RawMessage(`"0x` + minimum.Text(16) + `"`)
	if existing == nil {
		return fallback
	}
	var balStr string
	if err := json.Unmarshal(existing, &balStr); err != nil {
		return fallback
	}
	current := new(big.Int)
	s := strings.TrimPrefix(strings.TrimPrefix(balStr, "0x"), "0X")
	if _, ok := current.SetString(s, 16); !ok {
		return fallback
	}
	if current.Cmp(minimum) >= 0 {
		return existing
	}
	return fallback
}
