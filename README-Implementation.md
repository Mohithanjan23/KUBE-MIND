# KUBE-MIND: Local Bootable Implementation

This directory contains the functional code to demonstrate the production loophole outlined in the architectural docs.

## Repository Structure

* `src/inference/`: A mock Go gRPC Server that artificially delays responses to simulate slow GPU inference. It exports its internal `queue_depth` as a Prometheus metric.
* `src/loadgen/`: A minimal Go gRPC Client that blasts highly concurrent requests over a **single TCP connection**, simulating a misconfigured or bursty async event worker.
* `k8s/base/`: The naive "Loophole" architecture. It maps the gRPC Service directly via Kubernetes L4 iptables, resulting in all requests queuing on a single pod.
* `k8s/fix/`: The "Fixed" architecture. Introduces Envoy for L7 request buffering and circuit breaking.

## How to Run the Demonstration

### Prerequisites
* Docker
* [Kind](https://kind.sigs.k8s.io/) (Kubernetes IN Docker)
* kubectl

### Step 1: Prove the Failure (The Loophole)
Run the loophole deployment script. This will spin up a Kind cluster, build the images, and trigger the load burst directly against the L4 service.
```bash
chmod +x deploy-loophole.sh
./deploy-loophole.sh
```
*Observe the logs:* You will see massive latency spikes on the load generator because all 100 concurrent streams were pinned to a single inference pod. CPU usage on the cluster will remain low, preventing Kubernetes from scaling.

### Step 2: Prove the Fix
Run the fix deployment script. This deploys Envoy and routes the load generator through the Service Mesh.
```bash
chmod +x deploy-fix.sh
./deploy-fix.sh
```
*Observe the logs:* You will see latency remain flat and predictable. Envoy will aggressively round-robin the requests at the L7 HTTP/2 frame level across all pods, preventing the single-pod queue explosion.
