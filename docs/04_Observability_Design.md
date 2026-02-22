# PHASE 4: OBSERVABILITY DESIGN

## 1. Metrics Taxonomy
For an AI infrastructure, standard RED metrics (Rate, Errors, Duration) are insufficient. We expand to AI-aware RED+USE metrics.

* **Standard USE (Infrastructure):**
  * CPU/Memory Utilization (Node & Pod level)
  * Network Tx/Rx
* **GPU Metrics (DCGM Exporter):**
  * `DCGM_FI_DEV_GPU_UTIL`: Overall GPU Streaming Multiprocessor utilization.
  * `DCGM_FI_DEV_FB_USED`: Framebuffer (VRAM) used.
  * `DCGM_FI_PROF_SM_ACTIVE`: Tensor core activity.
* **Application RED (Inference Servers):**
  * `grpc_server_handled_total`: Total successful/failed requests.
  * `grpc_server_handling_seconds`: True processing latency.
  * `inference_queue_depth`: Number of requests waiting in the model server's internal queue.
  * `kv_cache_usage_percent`: LLM-specific memory saturation metric.

## 2. Logs and Tracing Strategy
* **Logging:** Structured JSON logs only. FluentBit deployed as a DaemonSet ships logs to an aggregator (e.g., Loki or Elastic). We log the start, end, and prompt token count, but NOT the payload itself (to comply with PII/Security requirements).
* **Distributed Tracing (OpenTelemetry):**
  * Context propagation injected at the API Gateway.
  * Spans created for: `Router Processing` -> `gRPC Network Time` -> `Triton Internal Queue` -> `GPU Execution Return`.
  * Tracing is heavily sampled (e.g., 5%) to avoid observability overhead, except for errors which are captured at 100%.

## 3. Latency and Queue Monitoring
The most critical blind spot in AI inference is queue time versus computation time.
* If a request takes 2000ms:
  * Network transfer: 10ms
  * Inference execution (GPU time): 150ms
  * Internal Queue time (waiting for other requests to finish): 1840ms.
* Exposing internal queue depth via Prometheus metrics (`/metrics` endpoint on the inference pod) is mandatory to understand if the system is actually backlogged.

## 4. What Kubernetes Sees vs What Users Experience
This sets up the core vulnerability.
* **Kubernetes Control Plane sees:** Average CPU at 40%, Memory at 60%. Pods are returning `HTTP 200` to readiness probes. No pods are repeatedly crashing or OOMing. Kubernetes believes the system is **perfectly healthy**.
* **The User sees:** Intermittent 5 to 10-second latencies, gRPC `DEADLINE_EXCEEDED` errors, and a degraded application experience. 
* There is a disconnect between node-level resource metrics and application-level flow metrics.
