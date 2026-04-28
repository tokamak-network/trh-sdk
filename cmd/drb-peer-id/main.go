package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: drb-peer-id <mnemonic> <role>")
		fmt.Fprintln(os.Stderr, "  role: leader | regular-1 | regular-2 | regular-3")
		os.Exit(1)
	}
	mnemonic := os.Args[1]
	role := os.Args[2]

	peerID, keyBytes, err := thanos.DerivePeerID(mnemonic, role)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("PeerID: %s\n", peerID)
	fmt.Printf("KeyB64: %s\n", base64.StdEncoding.EncodeToString(keyBytes))
}
