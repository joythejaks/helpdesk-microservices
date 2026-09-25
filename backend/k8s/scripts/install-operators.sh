#!/usr/bin/env bash
# Installs the two operators phase 13 depends on. They are cluster add-ons
# (external release manifests) and must exist BEFORE the kustomize tree is
# applied, because it contains their custom resources — the CRDs have to be
# there first. Versions are pinned; bump them deliberately.
#
#   CloudNativePG          -> Postgres primary/replica with automatic failover
#   RabbitMQ Cluster Op.   -> clustered RabbitMQ with quorum queues
#   cert-manager           -> required by the RabbitMQ operator's manifest,
#                             which issues its webhook serving certificate
#                             through a cert-manager Certificate/Issuer (found
#                             the hard way: without it, applying the operator
#                             fails on "no matches for kind Certificate")
#
# Idempotent: re-running upgrades in place.
set -euo pipefail

CNPG_VERSION=1.30.1
RABBITMQ_OPERATOR_VERSION=2.23.0
CERT_MANAGER_VERSION=1.21.2

# The manifests are fetched from GitHub by kubectl itself, and a dropped
# connection mid-download is a plain network hiccup (seen in practice), so
# retry instead of failing the whole install.
retry() {
  local n
  for n in 1 2 3 4 5; do
    "$@" && return 0
    echo "attempt $n failed: $*" >&2
    sleep 5
  done
  return 1
}

# Server-side apply: CloudNativePG's CRDs are too large for the annotation
# client-side apply stores, and its docs require this.
retry kubectl apply --server-side -f \
  "https://github.com/cloudnative-pg/cloudnative-pg/releases/download/v${CNPG_VERSION}/cnpg-${CNPG_VERSION}.yaml"
kubectl -n cnpg-system rollout status deployment/cnpg-controller-manager --timeout=180s

retry kubectl apply -f \
  "https://github.com/cert-manager/cert-manager/releases/download/v${CERT_MANAGER_VERSION}/cert-manager.yaml"
for d in cert-manager cert-manager-cainjector cert-manager-webhook; do
  kubectl -n cert-manager rollout status "deployment/${d}" --timeout=180s
done

# cert-manager's webhook can take a few more seconds to serve after its
# rollout completes, so this apply is retried too.
retry kubectl apply -f \
  "https://github.com/rabbitmq/cluster-operator/releases/download/v${RABBITMQ_OPERATOR_VERSION}/cluster-operator.yml"
kubectl -n rabbitmq-system rollout status deployment/rabbitmq-cluster-operator --timeout=180s

kubectl wait --for condition=established --timeout=60s \
  crd/clusters.postgresql.cnpg.io crd/rabbitmqclusters.rabbitmq.com

echo "operators ready: cloudnative-pg ${CNPG_VERSION}, cert-manager ${CERT_MANAGER_VERSION}, rabbitmq cluster-operator ${RABBITMQ_OPERATOR_VERSION}"
