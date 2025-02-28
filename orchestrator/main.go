package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	_ "github.com/go-redis/redis/v8"
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

var (
	redisClient *redis.Client
	ctx         = context.Background()
)

func main() {
	// Initialize Redis client
	redisClient = redis.NewClient(&redis.Options{
		//Addr: "localhost:6379",
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

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Printf("Error writing health check response: %v", err)
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
		} else {
			log.Printf("Error parsing MAX_REQUEST_SIZE: %v, using default size", err)
		}
	}

	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxSize))

	var req CodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate language
	if req.Language == "" {
		log.Print("Error: Language not specified in request")
		http.Error(w, "Language must be specified", http.StatusBadRequest)
		return
	}

	// Set default timeout if not provided
	if req.Timeout <= 0 {
		defaultTimeout := 10
		if timeoutEnv := os.Getenv("RUNNER_TIMEOUT"); timeoutEnv != "" {
			if timeout, err := strconv.Atoi(timeoutEnv); err == nil {
				defaultTimeout = timeout
			} else {
				log.Printf("Error parsing RUNNER_TIMEOUT: %v, using default timeout", err)
			}
		}
		req.Timeout = defaultTimeout
	}

	// Marshal request to JSON
	reqJSON, err := json.Marshal(req)
	if err != nil {
		log.Printf("Error marshalling request body: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	cacheKey := string(reqJSON)

	// Try to get response from cache
	cachedResponse, err := redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache hit - return cached response
		log.Printf("Cache hit for: %s", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		var cachedResp CodeResponse
		if err := json.Unmarshal([]byte(cachedResponse), &cachedResp); err != nil {
			log.Printf("Error unmarshalling cached response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(cachedResp); err != nil {
			log.Printf("Error encoding cache response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		return
	} else if !errors.Is(err, redis.Nil) {
		// If error is not a "key not found" error, log it
		log.Printf("Error checking cache: %v", err)
	}

	// Cache miss - process the request and forward to appropriate runner
	log.Printf("Cache miss for: %s", r.URL.Path)
	response, err := forwardToRunner(req)
	if err != nil {
		log.Printf("Error forwarding to runner: %v", err)
		http.Error(w, fmt.Sprintf("Error processing code: %s", response.Error), http.StatusInternalServerError)
		return
	}

	// Serialize the response
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if response.Error == "" {
		// Store in cache (expire after 1 hour)
		if err := redisClient.Set(ctx, cacheKey, responseJSON, time.Hour).Err(); err != nil {
			log.Printf("Failed to cache response: %v", err)
			// Continue anyway - caching failure shouldn't prevent response
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
