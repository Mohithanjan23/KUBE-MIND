# PHASE 3: AI INFERENCE STACK

## 1. Model Serving Approach
The inference layer is standardized to isolate the AI framework from the infrastructure layer. 
* **Large Language Models (LLMs):** Served using `vLLM` or `Text Generation Inference (TGI)` due to highly optimized PagedAttention memory management and continuous batching.
* **Computer Vision & Embeddings:** Served using `NVIDIA Triton Inference Server`. Triton allows dynamic batching, meaning it can hold requests for a few milliseconds to batch them together before sending them to the GPU, dramatically increasing throughput.

## 2. Warm-up and Cold-Start Handling
* **The Problem:** A 70B parameter model requires ~140GB of VRAM (even quantized). Loading this from disk to network to RAM to VRAM takes 2-5 minutes.
* **Mitigation (Warm-Up):** 
  1. InitContainers pre-fetch model weights.
  2. The server boots and loads weights into VRAM.
  3. A local sidecar or startup probe fires a "warm-up" dummy tensor request through the model to initialize CUDA graphs and allocate the KV cache overhead.
  4. Only after the warm-up request succeeds does the `ReadinessProbe` return `HTTP 200`, placing the pod into the Service endpoints list.

## 3. Versioning and Routing Logic
* Models are versioned heavily: `resnet-50-v1.2`, `llama-3-8b-instruct-v0.5`.
* The `Inference Router` passes the requested model version as a header or inside the gRPC payload.
* Infrastructure routing maps generic API paths (`/api/v1/embeddings`) into specific backend model endpoints via the API Gateway rules.

## 4. Handling Short vs Long Inference Requests
* **Short Requests (Embeddings/CV):** Latency is typically < 100ms. These are processed via synchronous unary gRPC calls. Timeout contexts are aggressively set (e.g., 500ms).
* **Long Requests (LLM Text Generation):** Generating 1000 tokens can take 10-20 seconds. These are handled via **gRPC Server Streaming**. The client receives tokens as they are generated.
* **Thread/Queue Segregation:** The system guarantees that slow LLM generation requests do not block thread pools required for fast embedding requests.

## 5. Performance Trade-offs
* **Latency vs. Throughput:** Activating dynamic batching increases median latency by ~10-20ms (due to waiting for the batch window to close) but increases overall system throughput by 4x.
* **Quantization:** Moving from FP16 to INT8/AWQ quantization reduces memory usage by 50% allowing larger batches, slightly degrading mathematical precision, but significantly improving output speed. We default to INT8 dynamic quantization for text models.
