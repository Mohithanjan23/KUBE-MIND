package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	pb "kubemind/inference/api"
)

var (
	queueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "triton_queue_depth",
		Help: "Current depth of the internal inference queue",
	})
	requestsProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "grpc_server_handled_total",
		Help: "Total number of completed RPCs",
	})
)

func init() {
	prometheus.MustRegister(queueDepth)
	prometheus.MustRegister(requestsProcessed)
}

type inferenceServer struct {
	pb.UnimplementedInferenceServer
	activeRequests int64
}

func (s *inferenceServer) Predict(ctx context.Context, req *pb.InferenceRequest) (*pb.InferenceResponse, error) {
	// Increment active queue depth
	atomic.AddInt64(&s.activeRequests, 1)
	queueDepth.Inc()
	defer func() {
		atomic.AddInt64(&s.activeRequests, -1)
		queueDepth.Dec()
		requestsProcessed.Inc()
	}()

	start := time.Now()

	// Simulate heavy GPU Inference time (100ms per request)
	// If activeRequests is high, it simulates queue wait time as well.
	// We'll keep it simple: it just sleeps to represent compute time.
	time.Sleep(100 * time.Millisecond)

	elapsed := time.Since(start).Milliseconds()

	return &pb.InferenceResponse{
		Result:           "inference_success",
		QueueDepth:       atomic.LoadInt64(&s.activeRequests),
		ProcessingTimeMs: elapsed,
	}, nil
}

func main() {
	// Start Prometheus Metrics Server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Metrics server listening on :9090")
		if err := http.ListenAndServe(":9090", nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	// Start gRPC Server
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterInferenceServer(s, &inferenceServer{})
	log.Printf("Inference server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
