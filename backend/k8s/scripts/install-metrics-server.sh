#!/usr/bin/env bash
# The HPAs need metrics-server, which Docker Desktop's cluster doesn't ship.
# Cluster add-on (external manifest), so it's deliberately not part of the
# kustomize tree. --kubelet-insecure-tls is needed because Docker Desktop's
# kubelet serves a self-signed certificate.
set -euo pipefail

kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

if ! kubectl -n kube-system get deployment metrics-server -o jsonpath='{.spec.template.spec.containers[0].args}' | grep -q kubelet-insecure-tls; then
  kubectl -n kube-system patch deployment metrics-server --type=json \
    -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
fi

kubectl -n kube-system rollout status deployment/metrics-server --timeout=120s
