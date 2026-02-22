#!/bin/bash

echo "🚀 KUBE-MIND: Deploying the Envoy / KEDA Fix"

# 1. Apply Envoy
echo "🛡️ Deploying Envoy L7 Proxy..."
kubectl apply -f k8s/fix/envoy-config.yaml
kubectl apply -f k8s/fix/envoy-gateway.yaml

echo "⏳ Waiting for Envoy to be ready..."
kubectl wait --for=condition=available --timeout=60s deployment/envoy-gateway -n kube-mind-system

# Note: KEDA & Prometheus installation omitted for brevity, 
# but in a real environment you would run:
# helm repo add kedacore https://kedacore.github.io/charts
# helm install keda kedacore/keda --namespace keda --create-namespace
# kubectl apply -f k8s/fix/keda-scaler.yaml

# 2. Execute The Fix Load Generator
echo "✅ Starting the Fix Load Generator (Targeting Envoy L7 Router)..."
kubectl apply -f k8s/fix/loadgen-fix.yaml

echo "🔍 Fetching logs from Fixed Load Generator to show predictable latency..."
sleep 5
LOADGEN_FIX_POD=$(kubectl get pods -n kube-mind-system -l job-name=loadgen-fix -o jsonpath='{.items[0].metadata.name}')
kubectl logs -f $LOADGEN_FIX_POD -n kube-mind-system
