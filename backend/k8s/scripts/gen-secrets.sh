#!/usr/bin/env bash
# Creates the Secret objects the manifests expect, straight into the cluster,
# with random unique values — nothing is written to disk or git. Idempotent:
# a secret that already exists is left alone (regenerating JWT/internal
# secrets logs everyone out; regenerating a DB password would NOT change the
# password stored in an existing database volume, so it would lock the
# services out — rotate those with ALTER USER, or wipe the PVCs).
#
# Bootstrap admin (BOOTSTRAP_ADMIN_EMAIL/PASSWORD) is intentionally not
# created: seed the first admin deliberately, not with a default.
set -euo pipefail

NS=helpdesk
rand() { openssl rand -hex 32; }
exists() { kubectl -n "$NS" get secret "$1" >/dev/null 2>&1; }

kubectl get namespace "$NS" >/dev/null 2>&1 || kubectl create namespace "$NS"

if ! exists app-secrets; then
  kubectl -n "$NS" create secret generic app-secrets \
    --from-literal=JWT_SECRET="$(rand)" \
    --from-literal=INTERNAL_SHARED_SECRET="$(rand)"
fi

if ! exists db-secrets; then
  kubectl -n "$NS" create secret generic db-secrets \
    --from-literal=AUTH_DB_PASSWORD="$(rand)" \
    --from-literal=TICKET_DB_PASSWORD="$(rand)" \
    --from-literal=NOTIFICATION_DB_PASSWORD="$(rand)"
fi

if ! exists rabbitmq-secret; then
  user=helpdesk
  pass="$(rand)"
  kubectl -n "$NS" create secret generic rabbitmq-secret \
    --from-literal=RABBITMQ_USER="$user" \
    --from-literal=RABBITMQ_PASS="$pass" \
    --from-literal=RABBITMQ_URL="amqp://${user}:${pass}@rabbitmq:5672/"
fi

if ! exists grafana-secret; then
  kubectl -n "$NS" create secret generic grafana-secret \
    --from-literal=GF_SECURITY_ADMIN_PASSWORD="$(rand)"
fi

echo "secrets ready in namespace $NS"
