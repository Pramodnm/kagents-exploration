#!/usr/bin/env bash
# Verify a kagent + Agent Substrate install and the examples in this repo.
# Read-only except for the optional sandbox and agent checks, which create and then
# delete their own objects. Safe to rerun.
#
# Usage:
#   scripts/verify.sh            # platform health + agents
#   scripts/verify.sh --sandbox  # also create, exercise, and delete a test sandbox
set -euo pipefail

PASS=0
FAIL=0
check() {
  local name="$1"; shift
  if "$@" >/dev/null 2>&1; then
    echo "PASS  $name"; PASS=$((PASS + 1))
  else
    echo "FAIL  $name"; FAIL=$((FAIL + 1))
  fi
}

check "kubectl can reach a cluster"            kubectl cluster-info
check "kagent controller rolled out"           kubectl rollout status deployment/kagent-controller -n kagent --timeout=60s
check "Agent Substrate api-server running"     kubectl get pods -n ate-system -l app=ate-api-server --field-selector=status.phase=Running
check "atelet DaemonSet ready"                 kubectl rollout status daemonset/atelet -n ate-system --timeout=60s
check "WorkerPool kagent-default ready"        bash -c 'kubectl get workerpools -n kagent kagent-default -o jsonpath="{.status.readyReplicas}" | grep -qE "^[1-9]"'

echo "--- agents in namespace kagent"
kubectl get agent -n kagent -o custom-columns=NAME:.metadata.name,READY:.status.conditions[-1].status 2>/dev/null || echo "(no agents)"

if [[ "${1:-}" == "--sandbox" ]]; then
  echo "--- sandbox round trip (creates and deletes one sandbox)"
  if ! command -v kagent >/dev/null; then
    echo "SKIP  kagent CLI not on PATH"
  else
    kubectl port-forward -n kagent svc/kagent-controller 8083:8083 >/dev/null 2>&1 & PF=$!
    trap 'kill $PF 2>/dev/null || true' EXIT
    sleep 3
    RID="verify-$(date +%s)"
    SB=$(kagent sandbox create python --request-id "$RID" --ttl 300s -o json | jq -r '.id')
    echo "created $SB"
    printf 'print(2 + 3)\n' > /tmp/verify-sample.py
    kagent sandbox upload "$SB" /tmp/verify-sample.py /data/workspace/sample.py >/dev/null
    OUT=$(kagent sandbox exec "$SB" -- python3 /data/workspace/sample.py 2>&1 || true)
    if echo "$OUT" | grep -q "^5$"; then
      echo "PASS  sandbox ran python (output 5)"; PASS=$((PASS + 1))
    else
      echo "FAIL  sandbox python output: $OUT"; FAIL=$((FAIL + 1))
    fi
    kagent sandbox delete "$SB" >/dev/null 2>&1 && echo "deleted $SB"
  fi
fi

echo
echo "passed=$PASS failed=$FAIL"
[[ $FAIL -eq 0 ]]
