#!/usr/bin/env bash
# Builds the four service images, makes sure the secrets exist, and applies
# the kustomize tree. Docker Desktop's Kubernetes shares the local Docker
# image store, so images built here are visible to the cluster (that's why
# the manifests use imagePullPolicy: IfNotPresent and no registry).
set -euo pipefail

cd "$(dirname "$0")/../.."   # backend/

for svc in auth-service ticket-service notification-service api-gateway; do
  docker build -t "helpdesk/${svc}:local" "./${svc}"
done

k8s/scripts/gen-secrets.sh

kubectl kustomize --load-restrictor=LoadRestrictionsNone k8s | kubectl apply -f -

# Images are tagged :local, so a rebuilt image is not noticed by an
# unchanged Deployment spec — restart the app pods to pick it up.
for svc in auth-service ticket-service notification-service api-gateway; do
  kubectl -n helpdesk rollout restart "deployment/${svc}" >/dev/null 2>&1 || true
done
