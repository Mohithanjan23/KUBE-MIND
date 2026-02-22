# PHASE 8: VALIDATION

## 1. Success Metrics
To validate that the engineering fix permanently resolves the loophole, we will measure the following Service Level Indicators (SLIs):
1. **P99 Inference Latency:** Must remain under 200ms during burst loads.
2. **Pod Request Distribution Variance:** The difference between the pod processing the most requests and the pod processing the least requests should be `< 5%`.
3. **Autoscaling Reaction Time:** The cluster must initiate a scale-up event within 15 seconds of an Envoy queue depth breach.

## 2. Validation Scenario & Tooling
We will use `ghz` (a gRPC load testing tool) to simulate the async event bursts.
* **Command:** 
  ```bash
  ghz --insecure --call inference.Inference/Predict \
      -d '{"tensor_payload": "data"}' \
      -c 200 -n 10000 \
      envoy-gateway:8080
  ```
  *(Sends 10,000 requests using 200 concurrent multiplexed HTTP/2 streams over a single connection context).*

## 3. Before vs. After Behavior

### **Before (Default Kubernetes Service + CPU HPA):**
* `ghz` reports a P99 latency of `8,200ms`.
* Prometheus `rate(grpc_requests_total)` shows Pod A receiving 10,000 reqs/sec, Pod B receiving 0 reqs/sec.
* Pod A CPU sits at `68%`. Deployment avg sits at `34%`.
* HPA does nothing (configured target `60%`).
* **Result:** Total structural failure resulting in application timeouts.

### **After (Envoy L7 + KEDA Prometheus Autoscaler):**
* `ghz` reports a P99 latency of `150ms`.
* Prometheus `rate(grpc_requests_total)` shows Pod A, Pod B receiving ~5,000 reqs/sec each.
* `envoy_cluster_upstream_rq_active` breaches the threshold of `16` per pod almost instantly.
* KEDA triggers the scale-up from 2 to 4 pods within 5 seconds.
* Envoy's Least-Request algorithm immediately starts sending new gRPC calls to Pod C & Pod D as their queues are empty, bypassing the legacy connection limitations.
* If load exceeds max capacity momentarily, Envoy enforces the `max_pending_requests: 100` circuit breaker, returning HTTP 503 instead of tying up memory in a black hole.

## 4. Performance and Cost Impact
* **Performance:** Predictable latency guarantees are restored. AI inference is highly deterministic again.
* **Cost Factor:** KEDA reacts significantly faster to queue depth than the legacy HPA did to CPU usage. This prevents latencies but causes aggressive GPU provisioning. To control costs, Kubernetes clusters must implement `ClusterAutoscaler` limits or spot-instance fallback strategies for scaling node elasticity.
