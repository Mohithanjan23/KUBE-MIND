# KUBE-MIND: Cloud-Agnostic Kubernetes AI Platform

**KUBE-MIND** is a portfolio-grade, research-focused system architecture modeling a cloud-agnostic Kubernetes-based AI inference platform. It demonstrates deep practical knowledge in distributed microservices, gRPC telemetry, observability, and the nuances of horizontal AI autoscaling.

## The Core Concept
This project outlines a high-throughput hybrid AI architecture (LLM + CV + Embeddings). Importantly, it details a **real-world production loophole** found in naive Kubernetes AI inference deployments—specifically, how L4 load balancing of HTTP/2 multiplexed gRPC connections circumvents traditional CPU/Memory autoscaling rules, causing silent "false positive" failures (green metrics, destroyed user experience).

The documentation moves step-by-step from structural planning through failure analysis, terminating in an engineering fix utilizing Envoy Service Mesh and KEDA Custom Metrics scaling.

## Project Phases & Architecture Documentation

Please review the documentation in the `/docs/` directory.

1. **[Phase 1: System Design](docs/01_System_Design.md)** - High-level goals, boundaries, and tiering.
2. **[Phase 2: Kubernetes Architecture](docs/02_Kubernetes_Architecture.md)** - Isolation, pod statelessness, and scheduling restrictions.
3. **[Phase 3: AI Inference Stack](docs/03_AI_Inference_Stack.md)** - Model serving (vLLM/Triton), cold-start handling, and load profiling.
4. **[Phase 4: Observability Design](docs/04_Observability_Design.md)** - RED+USE metrics for AI infrastructure. The disconnect between infrastructure metrics and application truth.
5. **[Phase 5: Failure Demonstration (The Loophole)](docs/05_Failure_Demonstration.md)** - Dissecting the catastrophic failure caused by default L4 load balancing + multiplexed gRPC + naive CPU scaling.
6. **[Phase 6: Engineering Fix](docs/06_Engineering_Fix.md)** - Introduction of L7 routing and custom Prometheus/queue-depth scaling criteria.
7. **[Phase 7: Implementation Artifacts](docs/07_Implementation_Artifacts.md)** - Complete YAML manifests (KEDA, Envoy) and orchestrator setup.
8. **[Phase 8: Validation](docs/08_Validation.md)** - Load testing strategies, comparing "Before vs. After", and defining acceptable SLIs.

---

## Local Bootable Implementation

The repository also contains the functional code to demonstrate the production loophole outlined in the architectural docs.

### Repository Structure

* `src/inference/`: A mock Go gRPC Server that artificially delays responses to simulate slow GPU inference. It exports its internal `queue_depth` as a Prometheus metric.
* `src/loadgen/`: A minimal Go gRPC Client that blasts highly concurrent requests over a **single TCP connection**, simulating a misconfigured or bursty async event worker.
* `k8s/base/`: The naive "Loophole" architecture. It maps the gRPC Service directly via Kubernetes L4 iptables, resulting in all requests queuing on a single pod.
* `k8s/fix/`: The "Fixed" architecture. Introduces Envoy for L7 request buffering and circuit breaking.
* `helm/kube-mind/`: A production-ready parameterizable Helm chart packaging the fix utilizing Envoy and KEDA.

### How to Run the Demonstration

#### Prerequisites
* Docker
* [Kind](https://kind.sigs.k8s.io/) (Kubernetes IN Docker)
* `kubectl`
* `helm` (for advanced deployment)

#### Step 1: Prove the Failure (The Loophole)
Run the loophole deployment script. This will spin up a Kind cluster, build the images, and trigger the load burst directly against the L4 service.
```bash
chmod +x deploy-loophole.sh
./deploy-loophole.sh
```
*Observe the logs:* You will see massive latency spikes on the load generator because all 100 concurrent streams were pinned to a single inference pod. CPU usage on the cluster will remain low, preventing Kubernetes from scaling.

#### Step 2: Prove the Fix
Run the fix deployment script. This deploys Envoy and routes the load generator through the Service Mesh.
```bash
chmod +x deploy-fix.sh
./deploy-fix.sh
```
*Observe the logs:* You will see latency remain flat and predictable. Envoy will aggressively round-robin the requests at the L7 HTTP/2 frame level across all pods, preventing the single-pod queue explosion.

---
*Authored by Antigravity AI — Principal Software Engineer & AI Infrastructure Architect perspective.*
