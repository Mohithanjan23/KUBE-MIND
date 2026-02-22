# PHASE 5: FAILURE DEMONSTRATION

## 1. The Scenario
The KUBE-MIND platform is deployed. The CV inference service (Triton) has an HPA configured to scale between 3 to 10 pods based on maintaining an average CPU target of 60%. 

An upstream event-processor (written in Go) begins flushing a backlog of 50,000 images to the CV service. It opens a gRPC connection and multiplexes thousands of concurrent requests over it.

## 2. The Loophole: L4 Load Balancing + HTTP/2 Multiplexing
gRPC uses HTTP/2, which is designed to multiplex many requests over a single, persistent TCP connection. 

Standard Kubernetes `Service` objects operate at Layer 4 (iptables / IPVS). When the Go event-processor opens a connection, Kubernetes routes that TCP connection to **Pod A**. 
Because the connection remains open, *every single multiplexed request from that client goes to Pod A*. 

**Pod B** and **Pod C** receive absolutely no traffic.

## 3. False Health: Why Metrics Fail
* **Pod A** receives thousands of requests. Triton pushes as many as it can to the GPU. The GPU reaches 100% utilization. The CPU, however, is merely copying memory to the GPU and managing the queue. Pod A's CPU usage plateaus at ~70%.
* **Pod B** and **Pod C** sit completely idle. CPU usage is ~5%.
* **The HPA Calculation:** `(70 + 5 + 5) / 3 = 26% Average CPU`.
* Because 26% is far below the 60% threshold, **Kubernetes refuses to scale the deployment.** The HPA takes no action.

## 4. User-Visible Impact
1. **Massive Latency:** The internal queue in Pod A grows exponentially. Requests that should take 50ms are taking 8,000ms.
2. **Timeouts:** Clients eventually hit their gRPC context deadline and drop the requests, resulting in cascading failures.
3. **No Rescheduling:** Pod A does not crash. It does not run out of system memory. Liveness probes (which are simple HTTP pings) return instantly because the web-server thread isn't blocked, only the inference queue is. Kubernetes does not restart the pod.

## 5. Root Cause Analysis
This is fundamentally a misunderstanding of how network traffic behaves in modern asynchronous systems.
1. **L4 vs L7 Routing:** Kubernetes Services balance *connections*. gRPC requires balancing *requests*.
2. **Wrong Scaling Metric:** Using CPU to scale an AI workload is a severe anti-pattern. AI workloads are bound by GPU compute, VRAM bandwidth, and internal locking—which rarely saturate the host CPU proportionally. HPA relies on an invalid proxy metric.
3. **Implicit Asynchrony:** Event-driven clients generating bursts of traffic easily overwhelm single persistent connections without triggering standard infrastructure alarms.
