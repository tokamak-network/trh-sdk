#!/bin/bash

# drb_smoke.sh — Quick DRB health and activation verification
#
# Validates a deployed DRB Gaming preset within 5 seconds:
# 1. Checks CommitReveal2L2 predeploy contract exists (non-empty bytecode)
# 2. Checks 3+ operators activated via getActivatedOperators()
# 3. Checks drb-leader container is healthy
# 4. Checks drb-regular-{1,2,3} containers are healthy
#
# Usage:
#   ./drb_smoke.sh [RPC_URL] [CONTRACT_ADDR] [TIMEOUT_SEC]
#
# Default values:
#   RPC_URL: ws://localhost:8546 (local L2)
#   CONTRACT_ADDR: 0x4200000000000000000000000000000000000060 (DRB predeploy)
#   TIMEOUT_SEC: 5 (total execution time)
#
# Exit codes:
#   0 = all checks passed
#   1 = any check failed

set -euo pipefail

# Arguments
RPC_URL="${1:-ws://localhost:8546}"
CONTRACT_ADDR="${2:-0x4200000000000000000000000000000000000060}"
TIMEOUT="${3:-5}"

echo "=== DRB Smoke Test ==="
echo "RPC: $RPC_URL"
echo "Contract: $CONTRACT_ADDR"
echo "Timeout: ${TIMEOUT}s"
echo ""

# Check 1: Predeploy contract code (non-empty = deployed)
echo -n "1. Checking predeploy contract..."
if code=$(timeout 2s cast code "$CONTRACT_ADDR" --rpc-url "$RPC_URL" 2>/dev/null || echo ""); then
  if [[ "$code" != "0x" && ! -z "$code" ]]; then
    echo " ✓"
  else
    echo " ✗ (code empty or 0x)"
    exit 1
  fi
else
  echo " ✗ (timeout or RPC error)"
  exit 1
fi

# Check 2: Activated operators (count 0x addresses, need 3+)
echo -n "2. Checking activated operators..."
if ops=$(timeout 2s cast call "$CONTRACT_ADDR" "getActivatedOperators()(address[])" --rpc-url "$RPC_URL" 2>/dev/null || echo ""); then
  # Count addresses (grep -o matches each 0x<40-hex-digits>)
  addr_count=$(echo "$ops" | grep -o "0x[a-fA-F0-9]\{40\}" | wc -l)
  if [[ $addr_count -ge 3 ]]; then
    echo " ✓ ($addr_count operators)"
  else
    echo " ✗ (found $addr_count, need 3+)"
    exit 1
  fi
else
  echo " ✗ (timeout or RPC error)"
  exit 1
fi

# Check 3: drb-leader container health
echo -n "3. Checking Leader container..."
if health=$(docker inspect -f '{{.State.Health.Status}}' drb-leader 2>/dev/null || echo ""); then
  if [[ "$health" == "healthy" ]]; then
    echo " ✓"
  else
    echo " ✗ (status: $health)"
    exit 1
  fi
else
  echo " ✗ (container not found)"
  exit 1
fi

# Check 4: drb-regular-1/2/3 container health
echo -n "4. Checking Regular containers..."
all_healthy=true
for i in 1 2 3; do
  if health=$(docker inspect -f '{{.State.Health.Status}}' drb-regular-$i 2>/dev/null || echo ""); then
    if [[ "$health" != "healthy" ]]; then
      all_healthy=false
      echo -n "[$i:$health] "
    fi
  else
    all_healthy=false
    echo -n "[$i:missing] "
  fi
done

if [[ "$all_healthy" == "true" ]]; then
  echo " ✓"
else
  echo " ✗"
  exit 1
fi

echo ""
echo "=== All checks passed ✓ ==="
exit 0
