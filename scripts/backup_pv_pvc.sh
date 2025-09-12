#!/usr/bin/env bash
#
# Kubernetes PV/PVC Backup Script for TRH-SDK
#
# This script creates a comprehensive backup of Kubernetes resources related to
# PersistentVolumes and PersistentVolumeClaims used by op-geth and op-node components.
# It automatically detects the namespace and backs up the following resources:
#
# - PVC definitions for op-geth and op-node components
# - PV definitions bound to the PVCs
# - StatefulSet definitions for op-geth and op-node
# - StorageClass definitions
# - ConfigMaps in the target namespace
# - Summary file with key information (PVC, PV, StorageClass, EFS IDs)
#
# Usage:
#   NAMESPACE=<namespace> ./backup_pv_pvc.sh
#   or
#   ./backup_pv_pvc.sh  # auto-detects namespace
#
# Output:
#   Creates timestamped backup directory: ./k8s-efs-backup/<namespace>_<timestamp>/
#
set -euo pipefail

command -v kubectl >/dev/null 2>&1 || { echo "kubectl not found"; exit 1; }

NAMESPACE="${NAMESPACE:-}"
if [ -z "${NAMESPACE}" ]; then
  NAMESPACE="$(kubectl get pvc -A -o jsonpath='{range .items[*]}{.metadata.namespace}{"\t"}{.metadata.name}{"\n"}{end}' \
    | egrep 'op-(geth|node)' | awk '{print $1}' | sort -u | head -1 || true)"
fi
if [ -z "${NAMESPACE}" ]; then
  NAMESPACE="$(kubectl get pods -A --no-headers -o custom-columns=NS:.metadata.namespace,NAME:.metadata.name \
    | egrep 'op-(geth|node)' | awk '{print $1}' | sort -u | head -1 || true)"
fi
if [ -z "${NAMESPACE}" ]; then
  NAMESPACE="$(kubectl config view --minify -o jsonpath='{..namespace}' || true)"
fi
if [ -z "${NAMESPACE}" ]; then
  echo "[-] Failed to auto-detect namespace. Set NAMESPACE env and retry."
  exit 1
fi

BACKUP_DIR="${BACKUP_DIR:-./k8s-efs-backup}"
TS="$(date +%Y%m%d-%H%M%S)"
OUT="${BACKUP_DIR}/${NAMESPACE}_${TS}"
mkdir -p "$OUT"

echo "[+] Namespace: $NAMESPACE"
echo "[+] Backup dir: $OUT"

PVC_LIST="$(kubectl -n "$NAMESPACE" get pvc -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | egrep 'op-(geth|node)' || true)"
if [ -z "$PVC_LIST" ]; then
  echo "[-] No op-geth/op-node PVC found in $NAMESPACE"
  exit 0
fi

SUMMARY="$OUT/summary.txt"
touch "$SUMMARY"

while IFS= read -r PVC; do
  [ -z "$PVC" ] && continue
  PV="$(kubectl -n "$NAMESPACE" get pvc "$PVC" -o jsonpath='{.spec.volumeName}' || true)"
  if [ -z "$PV" ]; then
    echo "[-] PVC $PVC has no bound PV (skip)" | tee -a "$SUMMARY"
    continue
  fi
  kubectl -n "$NAMESPACE" get pvc "$PVC" -o yaml > "$OUT/pvc_${PVC}.yaml"
  kubectl get pv "$PV" -o yaml > "$OUT/pv_${PV}.yaml"
  SC="$(kubectl get pv "$PV" -o jsonpath='{.spec.storageClassName}' || true)"
  EFS="$(kubectl get pv "$PV" -o jsonpath='{.spec.csi.volumeHandle}' 2>/dev/null || true)"
  {
    echo "PVC: $PVC"; echo "PV : $PV"; echo "SC : $SC"; echo "EFS: $EFS"; echo "-----"
  } >> "$SUMMARY"
done <<< "$PVC_LIST"

kubectl -n "$NAMESPACE" get statefulset -o name | egrep 'op-(geth|node)' | while read -r STS; do
  NAME="${STS##*/}"
  kubectl -n "$NAMESPACE" get "$STS" -o yaml > "$OUT/sts_${NAME}.yaml"
done

kubectl get storageclass -o yaml > "$OUT/storageclasses.yaml"
kubectl -n "$NAMESPACE" get cm -o yaml > "$OUT/configmaps.yaml" || true

echo "[+] Backup completed: $OUT"
