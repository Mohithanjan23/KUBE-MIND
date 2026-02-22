# PHASE 6: ENGINEERING FIX

## 1. Architectural Changes
To solve the multiplexed queuing vulnerability and naive autoscaling, we migrate from L4 to L7 routing and from infrastructure metrics to application-aware telemetry.

1. **L7 Load Balancing (Service Mesh / Envoy):** 
   * A service mesh (e.g., Linkerd or Istio) or a standalone Envoy proxy is introduced.
   * Instead of managing TCP connections, Envoy intercepts HTTP/2 frames, reads the gRPC headers, and balances *individual requests* across all active internal endpoints using round-robin or least-request algorithms.
2. **Custom Metrics Autoscaling (KEDA):**
   * The legacy Kubernetes HPA is retired.
   * **KEDA (Kubernetes Event-driven Autoscaling)** is deployed.
   * A `ScaledObject` is configured to scale based on a custom Prometheus query rather than CPU/Memory.

## 2. Design of AI-Aware Metrics
The exact formula for autoscaling is now based on active queue depth.
* **Target Metric:** `envoy_cluster_upstream_rq_active` (How many requests are actively in flight to the pod).
* Alternatively, for model servers that export their own queue: `triton_queue_depth`.
* **Formula Example:** If our optimal batch size is 8, we want no more than 16 items in-flight per pod (8 processing, 8 in queue). KEDA is instructed to scale up when `sum(upstream_rq_active) / pods > 16`.

## 3. Self-Healing Mechanisms
* **Circuit Breakers (Envoy):**
  * `max_requests_per_connection`: Limit to enforce periodic connection recycling just in case.
  * `max_pending_requests`: If the queue exceeds a specific depth, Envoy starts returning HTTP 503 (or gRPC `UNAVAILABLE`). This "Fail Fast" mechanism prevents cascading system failures and protects the user experience, allowing the gateway to retry on another cluster or return an immediate error.
* **Aggressive Readiness Probing:**
  * If a pod's inference queue exceeds 50, it artificially fails its readiness probe, immediately dropping out of the Envoy cluster until it drains its queue.

## 4. Improving Routing and Load Distribution
* **Least-Request Load Balancing:** Round-robin assigns requests equally. However, generating 50 LLM tokens is faster than 2000 tokens. Round-robin will eventually send long requests to the same pod. Envoy is configured to use the `Least-Request` algorithm, ensuring pods with empty queues receive the next task.

## 5. Architectural Trade-offs
* **Latency Overhead:** The Envoy proxy adds ~1-2ms to every hop. For <50ms CV requests, this is a minor but acceptable penalty in exchange for systemic stability. For LLMs, 1ms is completely imperceptible.
* **Complexity:** Introducing KEDA and Envoy increases the operational burden. Operators must now monitor the service mesh control plane as well as Prometheus adapter functionality.
* **Cost:** KEDA introduces rapid scaling. Over-provisioning GPU instances during bursts will inflate cloud bills faster than the sluggish CPU scaling did. Cost limits (Max Replicas) must be rigorously defined.
