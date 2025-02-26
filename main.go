package jcompiler

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
)

// Request represents the incoming request structure
type Request struct {
	Req string `json:"req"`
}

// Response represents the outgoing response structure
type Response struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Cached    bool   `json:"cached"`
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

	// Define API endpoints
	http.HandleFunc("/api/process", handleRequest)
	http.HandleFunc("/health", healthCheck)

	// Start the server
	log.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	// Parse request
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	// Create cache key from request body (using raw body as cache key)
	cacheKey := string(body)

	// Try to get response from cache
	cachedResponse, err := redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache hit - return cached response
		log.Printf("Cache hit for: %s", cacheKey)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(cachedResponse))
		return
	}

	// Cache miss - process the request
	log.Printf("Cache miss for: %s", cacheKey)
	response := processRequest(req)
	response.Cached = false

	// Serialize the response
	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error creating response", http.StatusInternalServerError)
		return
	}

	// Store in cache (expire after 1 hour)
	err = redisClient.Set(ctx, cacheKey, responseJSON, time.Hour).Err()
	if err != nil {
		log.Printf("Failed to cache response: %v", err)
	}

	// Return the response
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}

// processRequest simulates processing the request
// In a real application, this could be a more complex operation
func processRequest(req Request) Response {
	// Simulate some processing time
	time.Sleep(500 * time.Millisecond)

	return Response{
		Message:   fmt.Sprintf("Processed: %s", req.Req),
		Timestamp: time.Now().Format(time.RFC3339),
		Cached:    false,
	}
}
