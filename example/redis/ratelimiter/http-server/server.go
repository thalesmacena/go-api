package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"go-api/pkg/redis"
)

var rateLimiter *redis.RateLimiter
var redisClient *redis.Client

func main() {
	// Redis configuration
	redisHost := getEnvOrDefault("REDIS_HOST", "localhost")
	redisPort := getEnvOrDefaultInt("REDIS_PORT", 6379)
	redisPassword := getEnvOrDefault("REDIS_PASSWORD", "redis_password")

	// Create Redis client
	config := redis.NewRedisConfig().
		WithHost(redisHost).
		WithPort(redisPort).
		WithPassword(redisPassword).
		WithDatabase(0)

	redisClient = redis.NewClient(config)

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Create rate limiter
	var err error
	rateLimiter, err = redis.NewRateLimiter(redisClient, "api_endpoint", redis.NewRateLimiterOptions().
		WithMaxTransactionsPerMinute(100).
		WithWaitOnLimit(false).
		WithNamespace("http_server").
		WithCacheName("http_rate_limiter"))
	if err != nil {
		log.Fatalf("Failed to create rate limiter: %v", err)
	}

	// Setup routes
	http.HandleFunc("/api/rate-limit/", rateLimitHandler)
	http.HandleFunc("/health", healthHandler)

	// Start server
	port := getEnvOrDefault("PORT", "8080")
	addr := fmt.Sprintf(":%s", port)

	log.Printf("Server starting on http://localhost%s", addr)
	log.Printf("Rate limit: http://localhost%s/api/rate-limit/{userId}", addr)
	log.Printf("Health: http://localhost%s/health", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func rateLimitHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.URL.Path[len("/api/rate-limit/"):]
	if userId == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	transactionID, err := rateLimiter.AcquireWithKey(r.Context(), userId)
	if err != nil {
		// Rate limit exceeded - return 429
		response := map[string]interface{}{
			"status":      "rejected",
			"userId":      userId,
			"error":       err.Error(),
			"timestamp":   time.Now().Unix(),
			"transaction": "",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests) // 429
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
		return
	}

	// Rate limit passed - return 200
	response := map[string]interface{}{
		"status":      "accepted",
		"userId":      userId,
		"transaction": transactionID,
		"timestamp":   time.Now().Unix(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := redisClient.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	metrics, err := rateLimiter.GetMetrics(ctx)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"metrics": metrics,
	})
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
