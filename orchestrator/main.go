package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type CodeRequest struct {
	Code     string            `json:"code"`
	Timeout  int               `json:"timeout"`
	Language string            `json:"language"`
	Args     []string          `json:"args,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
}

type Code struct {
	Code string `json:"code"`
}

type CodeResponse struct {
	Stdout        string `json:"stdout"`
	Stderr        string `json:"stderr"`
	ExitCode      int    `json:"exitCode"`
	ExecutionTime int64  `json:"executionTime"`
	Error         string `json:"error,omitempty"`
}

// Request represents the incoming request structure
type Request struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

// Response represents the outgoing response structure
type Response struct {
	Output    string `json:"output"`
	Timestamp string `json:"timestamp"`
	Cached    bool   `json:"cached"`
	Error     string `json:"error"`
}

var (
	redisClient *redis.Client
	ctx         = context.Background()
)

func main() {
	// Initialize Redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password
		DB:       0,  // default DB
	})

	// Check if Redis connection is successful
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")

	r := mux.NewRouter()
	r.HandleFunc("/api/process", handleRequest).Methods("POST")
	r.HandleFunc("/health", healthCheck).Methods("GET")

	port := "8000"
	log.Printf("Starting orchestrator service on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling request for %s", r.URL.Path)

	// Get max request size from env or default to 100KB
	maxSize := 102400
	if sizeEnv := os.Getenv("MAX_REQUEST_SIZE"); sizeEnv != "" {
		if size, err := strconv.Atoi(sizeEnv); err == nil {
			maxSize = size
		}
	}

	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxSize))

	var req CodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate language
	if req.Language == "" {
		http.Error(w, "Language must be specified", http.StatusBadRequest)
		return
	}

	// Set default timeout if not provided
	if req.Timeout <= 0 {
		defaultTimeout := 10
		if timeoutEnv := os.Getenv("RUNNER_TIMEOUT"); timeoutEnv != "" {
			if timeout, err := strconv.Atoi(timeoutEnv); err == nil {
				defaultTimeout = timeout
			}
		}
		req.Timeout = defaultTimeout
	}

	// Marshal request to JSON
	reqBody, err := json.Marshal(r.Body)
	if err != nil {
		log.Print("Error marshalling request body")
		return
	}

	cacheKey := string(reqBody)

	// Try to get response from cache
	cachedResponse, err := redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache hit - return cached response
		log.Printf("Cache hit for: %s", cacheKey)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(cachedResponse)
		if err != nil {
			log.Print("Error encoding cache response")
			return
		}
		return
	}

	// Cache miss - process the request and forward to appropriate runner
	log.Printf("Cache miss for: %s", cacheKey)
	response, err := forwardToRunner(req)
	if err != nil {
		log.Print(response)
		log.Printf("Error forwarding to runner: %v", err)
		return
	}

	// Serialize the response
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		return
	}

	// Store in cache (expire after 1 hour)
	err = redisClient.Set(ctx, cacheKey, responseJSON, time.Hour).Err()
	if err != nil {
		log.Printf("Failed to cache response: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Printf("Error encoding response: %v", err)
		return
	}
}
