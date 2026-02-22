#!/bin/bash

echo "🚀 KUBE-MIND: Bootstrapping Local Environment Loophole Demo"

# 1. Start Kind Cluster
echo "📦 Starting Kind Cluster..."
kind create cluster --name kube-mind

# 2. Build Docker Images
echo "🐳 Building Mock Inference and Load Generator images..."
docker build -t kubemind/inference:latest -f src/inference/Dockerfile .
docker build -t kubemind/loadgen:latest -f src/loadgen/Dockerfile .

# 3. Load Images into Kind
echo "📥 Loading images into Kind node..."
kind load docker-image kubemind/inference:latest --name kube-mind
kind load docker-image kubemind/loadgen:latest --name kube-mind

# 4. Apply The Loophole Architecture
echo "⚙️ Applying Base Kubernetes Architecture (The Loophole)..."
kubectl apply -f k8s/base/inference.yaml

echo "⏳ Waiting for inference server to be ready..."
kubectl wait --for=condition=available --timeout=60s deployment/inference-server -n kube-mind-system
kubectl wait --for=condition=ready pod -l app=inference-server -n kube-mind-system --timeout=60s

# 5. Execute The Loophole Load Generator
echo "💥 Starting the Load Generator (100 multiplexed HTTP/2 streams over 1 connection)..."
kubectl apply -f k8s/base/loadgen-loophole.yaml

echo "🔍 Fetching logs from Load Generator to show latency explosion..."
sleep 5
LOADGEN_POD=$(kubectl get pods -n kube-mind-system -l job-name=loadgen-loophole -o jsonpath='{.items[0].metadata.name}')
kubectl logs -f $LOADGEN_POD -n kube-mind-system
