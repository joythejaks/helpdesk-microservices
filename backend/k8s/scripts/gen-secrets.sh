#!/usr/bin/env bash
# Creates the Secret objects the manifests expect, straight into the cluster,
# with random unique values — nothing is written to disk or git. Idempotent:
# a secret that already exists is left alone (regenerating JWT/internal
# secrets logs everyone out; regenerating a DB password would NOT change the
# password stored in an existing database volume, so it would lock the
# services out — rotate those with ALTER USER, or wipe the PVCs).
#
# Some credentials are needed twice, by different consumers: the apps read
# db-secrets / rabbitmq-secret, while the CloudNativePG clusters bootstrap
# from *-db-credentials (basic-auth) and the RabbitMQ operator adopts
# rabbitmq-default-user. The second set is always derived from the first
# (never freshly generated), so the two cannot drift apart.
#
# Bootstrap admin (BOOTSTRAP_ADMIN_EMAIL/PASSWORD) is intentionally not
# created: seed the first admin deliberately, not with a default.
set -euo pipefail

NS=helpdesk
rand() { openssl rand -hex 32; }
exists() { kubectl -n "$NS" get secret "$1" >/dev/null 2>&1; }
# get_secret <secret> <key> -> decoded value
get_secret() { kubectl -n "$NS" get secret "$1" -o "jsonpath={.data.$2}" | base64 -d; }

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

# CloudNativePG bootstrap credentials, derived from db-secrets.
for pair in auth:AUTH ticket:TICKET notification:NOTIFICATION; do
  db="${pair%%:*}"
  key="${pair##*:}_DB_PASSWORD"
  if ! exists "${db}-db-credentials"; then
    kubectl -n "$NS" create secret generic "${db}-db-credentials" \
      --type=kubernetes.io/basic-auth \
      --from-literal=username="$db" \
      --from-literal=password="$(get_secret db-secrets "$key")"
  fi
done

if ! exists rabbitmq-secret; then
  user=helpdesk
  pass="$(rand)"
  kubectl -n "$NS" create secret generic rabbitmq-secret \
    --from-literal=RABBITMQ_USER="$user" \
    --from-literal=RABBITMQ_PASS="$pass" \
    --from-literal=RABBITMQ_URL="amqp://${user}:${pass}@rabbitmq:5672/"
fi

# The RabbitMQ operator adopts a pre-existing <cluster>-default-user Secret,
# but the pods mount its default_user.conf key, which the operator only
# writes when IT generates the secret. A secret with just username/password
# leaves every pod stuck in Init ("references non-existent secret key:
# default_user.conf"), so supply that key too.
#
# This must exist BEFORE the RabbitmqCluster does (apply.sh runs this script
# first). If the cluster already exists, the operator races to generate its
# own secret with different credentials, and RabbitMQ only reads the user
# once, on first boot — so never swap credentials under a running cluster:
# delete the RabbitmqCluster and its persistence-rabbitmq-server-* PVCs
# first. `apply` (not delete + create) leaves no window for the operator to
# win that race.
if [ -z "$(kubectl -n "$NS" get secret rabbitmq-default-user -o 'jsonpath={.data.default_user\.conf}' 2>/dev/null)" ]; then
  ruser="$(get_secret rabbitmq-secret RABBITMQ_USER)"
  rpass="$(get_secret rabbitmq-secret RABBITMQ_PASS)"
  kubectl -n "$NS" create secret generic rabbitmq-default-user \
    --from-literal=username="$ruser" \
    --from-literal=password="$rpass" \
    --from-literal=default_user.conf="$(printf 'default_user = %s\ndefault_pass = %s\n' "$ruser" "$rpass")" \
    --dry-run=client -o yaml | kubectl apply -f -
fi

if ! exists grafana-secret; then
  kubectl -n "$NS" create secret generic grafana-secret \
    --from-literal=GF_SECURITY_ADMIN_PASSWORD="$(rand)"
fi

echo "secrets ready in namespace $NS"
