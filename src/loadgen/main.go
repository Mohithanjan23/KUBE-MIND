package main

import (
	"context"
	"flag"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "kubemind/inference/api"
)

func main() {
	target := flag.String("target", "localhost:9000", "gRPC target address")
	concurrency := flag.Int("concurrency", 50, "Number of concurrent requests multiplexed on the connection")
	flag.Parse()

	log.Printf("Dialing %s...", *target)
	conn, err := grpc.Dial(*target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewInferenceClient(conn)

	log.Printf("Starting load generation with concurrency %d on a SINGLE HTTP/2 connection", *concurrency)

	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				start := time.Now()
				// 2-second timeout to simulate failure threshold
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				r, err := c.Predict(ctx, &pb.InferenceRequest{Data: "test_tensor"})
				
				if err != nil {
					log.Printf("[Worker %d] Request failed/timeout: %v", workerID, err)
				} else {
					log.Printf("[Worker %d] Latency: %s | Server Queue Depth: %d", workerID, time.Since(start), r.GetQueueDepth())
				}
				cancel()
				
				// Keep pounding the server
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Run indefinitely
	wg.Wait()
}
