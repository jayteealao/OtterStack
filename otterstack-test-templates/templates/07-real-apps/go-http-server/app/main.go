package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

// Config holds application configuration
type Config struct {
	Port       string
	AppName    string
	Version    string
	APIKey     string
	ServerAddr string
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    float64   `json:"uptime"`
	Version   string    `json:"version"`
}

// InfoResponse represents the info endpoint response
type InfoResponse struct {
	App        string            `json:"app"`
	Version    string            `json:"version"`
	GoVersion  string            `json:"go_version"`
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	Memory     map[string]uint64 `json:"memory"`
	Goroutines int               `json:"goroutines"`
}

// EchoRequest represents the echo endpoint request
type EchoRequest struct {
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

// EchoResponse represents the echo endpoint response
type EchoResponse struct {
	Received  EchoRequest `json:"received"`
	Timestamp time.Time   `json:"timestamp"`
}

// EnvResponse represents the environment endpoint response
type EnvResponse struct {
	AppName   string `json:"app_name"`
	Version   string `json:"version"`
	Port      string `json:"port"`
	APIKeySet bool   `json:"api_key_set"`
}

var (
	config    Config
	startTime time.Time
)

func init() {
	startTime = time.Now()
	config = Config{
		Port:    getEnv("PORT", "8080"),
		AppName: getEnv("APP_NAME", "OtterStack Test Go Server"),
		Version: getEnv("VERSION", "1.0.0"),
		APIKey:  getEnv("API_KEY", ""),
	}
	config.ServerAddr = fmt.Sprintf("0.0.0.0:%s", config.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    time.Since(startTime).Seconds(),
		Version:   config.Version,
	}
	respondJSON(w, http.StatusOK, response)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	response := map[string]interface{}{
		"message": fmt.Sprintf("Welcome to %s", config.AppName),
		"version": config.Version,
		"endpoints": map[string]string{
			"health": "/health",
			"info":   "/info",
			"echo":   "/echo",
			"env":    "/env",
		},
	}
	respondJSON(w, http.StatusOK, response)
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	response := InfoResponse{
		App:        config.AppName,
		Version:    config.Version,
		GoVersion:  runtime.Version(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Goroutines: runtime.NumGoroutine(),
		Memory: map[string]uint64{
			"alloc_mb":       m.Alloc / 1024 / 1024,
			"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
			"sys_mb":         m.Sys / 1024 / 1024,
		},
	}
	respondJSON(w, http.StatusOK, response)
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EchoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := EchoResponse{
		Received:  req,
		Timestamp: time.Now(),
	}
	respondJSON(w, http.StatusOK, response)
}

func envHandler(w http.ResponseWriter, r *http.Request) {
	response := EnvResponse{
		AppName:   config.AppName,
		Version:   config.Version,
		Port:      config.Port,
		APIKeySet: config.APIKey != "",
	}
	respondJSON(w, http.StatusOK, response)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	}
}

func main() {
	// Setup routes
	http.HandleFunc("/", loggingMiddleware(rootHandler))
	http.HandleFunc("/health", loggingMiddleware(healthHandler))
	http.HandleFunc("/info", loggingMiddleware(infoHandler))
	http.HandleFunc("/echo", loggingMiddleware(echoHandler))
	http.HandleFunc("/env", loggingMiddleware(envHandler))

	// Setup graceful shutdown
	server := &http.Server{
		Addr:         config.ServerAddr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("%s v%s starting on %s", config.AppName, config.Version, config.ServerAddr)
		log.Printf("Health check: http://%s/health", config.ServerAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")
	os.Exit(0)
}
