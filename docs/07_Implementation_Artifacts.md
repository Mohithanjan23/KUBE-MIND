# PHASE 7: IMPLEMENTATION ARTIFACTS

This section provides the implementation manifests demonstrating the difference in architecture.

## 1. Gateway Sidecar Configuration (Envoy L7 Routing)
*Required to break multiplexed gRPC connections into per-request load balancing.*

```yaml
# proxy-config.yaml
static_resources:
  listeners:
  - address:
      socket_address:
        address: 0.0.0.0
        port_value: 8080
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: grpc_router
          codec_type: HTTP2
          route_config:
            name: local_route
            virtual_hosts:
            - name: backend
              domains: ["*"]
              routes:
              - match: { prefix: "/" }
                route: 
                  cluster: ai_inference_cluster
                  # Fail fast if unavailable
                  timeout: 2.0s
          http_filters:
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
  clusters:
  - name: ai_inference_cluster
    connect_timeout: 0.25s
    type: STRICT_DNS
    # Explicit HTTP2 ensures gRPC compatibility
    typed_extension_protocol_options:
      envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
        "@type": type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
        explicit_http_config:
          http2_protocol_options: {}
    # L7 Least Request algorithm replacing L4 IPVS
    lb_policy: LEAST_REQUEST
    load_assignment:
      cluster_name: ai_inference_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: ai-inference-service.ai-namespace.svc.cluster.local
                port_value: 9000
    circuit_breakers:
      thresholds:
      - max_pending_requests: 100
        max_requests: 100
```

## 2. KEDA ScaledObject Definition
*Autoscale based on the in-flight metrics provided by the Envoy sidecar, bypassing the host CPU metric.*

```yaml
# keda-scaler.yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: triton-inference-scaler
  namespace: ai-namespace
spec:
  scaleTargetRef:
    name: triton-inference-deployment
  minReplicaCount: 2
  maxReplicaCount: 20
  cooldownPeriod: 300 # Prevent rapid scale-down of expensive models
  triggers:
  - type: prometheus
    metadata:
      serverAddress: http://prometheus-server.observability.svc.cluster.local:9090
      # Sum of total active upstream requests across Envoy pods
      metricName: envoy_cluster_upstream_rq_active
      # We target ~16 active requests per inference pod before scaling up
      threshold: "16"
      query: sum(envoy_cluster_upstream_rq_active{cluster_name="ai_inference_cluster"})
```

## 3. Event Flow Automation (Pseudocode)
*Simulating the orchestrator that feeds requests to the load balancer.*

```go
package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	pb "kubemind/inference/api"
)

// The orchestrator multiplexes all requests onto this single L4 connection.
// Envoy intercepts this at the other end and performs L7 dispatching.
func main() {
	conn, err := grpc.Dial("envoy-gateway:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewInferenceClient(conn)

	// Simulate event stream bursting
	for i := 0; i < 50000; i++ {
		go func(id int) {
			// Without Envoy, all these land on Pod A queue.
			// With Envoy, these are spread per-call based on LeastRequest.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			
			_, err := client.Predict(ctx, &pb.InferenceRequest{Data: "tensor_payload"})
			if err != nil {
                // If Circuit Breaker trips, Envoy returns UNAVAILABLE immediately
				log.Printf("Req %d failed/breaker tripped: %v", id, err)
			}
		}(i)
	}
}
```
