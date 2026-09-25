#!/usr/bin/env bash
# Builds the four service images, makes sure the secrets exist, and applies
# the kustomize tree. There is no registry: the manifests use
# imagePullPolicy: IfNotPresent on helpdesk/<svc>:local.
#
# In Docker Desktop's kind mode the cluster node has its own containerd, and
# it only picks up a :local image the FIRST time — after a rebuild it keeps
# running the old one (a stale image ran for a whole debugging session
# before this was noticed). So each rebuilt image is imported into the node
# explicitly. In kubeadm mode the store is shared and the node container
# doesn't exist, so the import is skipped.
set -euo pipefail

cd "$(dirname "$0")/../.."   # backend/

NODE=desktop-control-plane

for svc in auth-service ticket-service notification-service api-gateway; do
  docker build -t "helpdesk/${svc}:local" "./${svc}"
  if docker inspect "$NODE" >/dev/null 2>&1; then
    docker save "helpdesk/${svc}:local" | docker exec -i "$NODE" ctr -n k8s.io images import --all-platforms - >/dev/null
  fi
done

k8s/scripts/gen-secrets.sh

kubectl kustomize --load-restrictor=LoadRestrictionsNone k8s | kubectl apply -f -

# Images are tagged :local, so a rebuilt image is not noticed by an
# unchanged Deployment spec — restart the app pods to pick it up.
for svc in auth-service ticket-service notification-service api-gateway; do
  kubectl -n helpdesk rollout restart "deployment/${svc}" >/dev/null 2>&1 || true
done
