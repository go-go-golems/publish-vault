#!/usr/bin/env bash
set -euo pipefail

# Check the GitOps PR result and the live retro-obsidian-publish deployment.
# This script does not mutate cluster state; it only reports GitHub, Argo CD,
# Kubernetes deployment, and public endpoint status.

: "${GITOPS_REPO:=wesen/2026-03-27--hetzner-k3s}"
: "${GITOPS_PR:=97}"
: "${EXPECTED_IMAGE:=ghcr.io/go-go-golems/publish-vault:sha-e61c800}"
: "${HK3S_REPO:=/home/manuel/code/wesen/2026-03-27--hetzner-k3s}"
: "${KUBECONFIG:=$HK3S_REPO/kubeconfig-k3s-demo-1.tail879302.ts.net.yaml}"
: "${PUBLIC_URL:=https://parc.yolo.scapegoat.dev}"

export KUBECONFIG

echo "== GitOps PR =="
gh pr view "$GITOPS_PR" --repo "$GITOPS_REPO" \
  --json state,mergedAt,mergeCommit,url,title \
  | jq '{state, mergedAt, mergeCommit:.mergeCommit.oid, title, url}'

echo "== Argo CD application =="
kubectl --request-timeout=15s -n argocd get application retro-obsidian-publish \
  -o jsonpath='{.status.sync.status}{"\t"}{.status.health.status}{"\t"}{.status.sync.revision}{"\n"}'

echo "== Kubernetes deployment image =="
image=$(kubectl --request-timeout=15s -n retro-obsidian-publish get deploy retro-obsidian-publish \
  -o jsonpath='{.spec.template.spec.containers[?(@.name=="app")].image}{"\n"}')
echo "$image"
if [[ "$image" != "$EXPECTED_IMAGE" ]]; then
  echo "unexpected image: got $image expected $EXPECTED_IMAGE" >&2
  exit 1
fi

echo "== Rollout status =="
kubectl --request-timeout=15s -n retro-obsidian-publish rollout status deploy/retro-obsidian-publish --timeout=30s

echo "== Public endpoint =="
curl -sS -I "$PUBLIC_URL/" | sed -n '1,8p'
echo "-- healthz --"
curl -sS "$PUBLIC_URL/api/healthz" | jq .
