# PHASE 2: KUBERNETES ARCHITECTURE

## 1. Namespace and Isolation Strategy
To prevent noisy-neighbor problems and restrict blast radiuses, KUBE-MIND uses strict logical isolation through Namespaces, backed by NetworkPolicies and ResourceQuotas.

* `ingress-system`: Edge proxies, TLS cert-managers.
* `ai-inference-system`: Dedicated to GPU-bound model serving pods.
* `ai-orchestration`: CPU-bound Go/Rust microservices routing the traffic.
* `observability`: Prometheus, Promtail/FluentBit, Jaeger/Tempo.

**RBAC:** Strict separation of ServiceAccounts. The inference pods have no Kubernetes API access, mitigating SSRF-to-Cluster-Admin vulnerabilities.

## 2. Pod and Service Design
* **Statelessness:** All inference pods are treated as cattle. Model weights are mounted either via high-performance network file systems (e.g., Amazon FSx, NFS, or pre-fetched hostPath caching) or pulled into a shared volume via InitContainers.
* **Service Types:** ClusterIP for all internal routing. Headless Services (`clusterIP: None`) are specifically avoided for standard REST, but **are a critical failure vector for gRPC** if used improperly (which leads into our designed loophole).
* **Probes:**
  * *Liveness:* strictly checking if the inference server process is responsive.
  * *Readiness:* ensures the model weights are actually loaded into VRAM. Until weights are in memory, the pod receives no traffic.

## 3. Deployment Strategy
* **Blue/Green & Canary:** Due to the severe cost and cold-start time of AI workloads, standard RollingUpdates are risky. We utilize Canary deployments (e.g., via ArgoRollouts).
* 10% of traffic is routed to the new model version. We monitor latency and error rates. If successful, promote to 100%.

## 4. Scaling Assumptions (The Setup)
* **Initial Autoscaling:** The legacy Kubernetes Horizontal Pod Autoscaler (HPA) targets 60% CPU utilization and 70% Memory utilization.
* **Assumption:** "If a pod gets too many requests, its CPU will usage will spike, HPA will calculate the average, and spin up more pods."
* *Spoiler: For AI/gRPC architectures, this assumption is fundamentally flawed.*

## 5. Scheduling Considerations for AI Workloads
* **NodeSelectors & Taints:** GPU nodes are expensive. We use taints (`nvidia.com/gpu=true:NoSchedule`) so only inference pods land on GPU nodes.
* **Topology Spread Constraints:** Ensure pods are spread across multiple Availability Zones to survive AZ failures.
* **Pod Anti-Affinity:** Prevent multiple heavy LLM pods from scheduling on the same physical node if the scheduler miscalculates available PCIe bandwidth.
* **InitContainers for Model Fetching:** We decouple container image size from model weight size. The Docker image contains only the server (vLLM/Triton). An InitContainer downloads the `safetensors` from S3/GCS into an `emptyDir` memory/disk volume before the main container starts.
