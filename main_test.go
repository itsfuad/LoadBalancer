package main

import (
	"encoding/json"
	"fmt"
	"io"
	"loadbalancer/balancer"
	"loadbalancer/config"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// MockServer represents a test HTTP server
type MockServer struct {
	server *httptest.Server
	URL    string
}

// setupMockServers creates test backend servers
func setupMockServers(count int) []MockServer {
	var servers []MockServer
	for i := 0; i < count; i++ {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Mock Server %d", i+1)
		})
		server := httptest.NewServer(handler)
		servers = append(servers, MockServer{
			server: server,
			URL:    server.URL,
		})
	}
	return servers
}

// createTestConfig creates a temporary config file
func createTestConfig(t *testing.T, servers []MockServer) string {
	urls := make([]string, len(servers))
	for i, s := range servers {
		urls[i] = s.URL
	}

	configData := config.Config{
		LoadBalancer: struct {
			Port                       int    "json:\"port\""
			HealthCheckIntervalSeconds int    "json:\"health_check_interval_seconds\""
			Strategy                   string "json:\"strategy\""
		}{
			Port:                       8080,
			HealthCheckIntervalSeconds: 5,
			Strategy:                   "round_robin",
		},
		Servers: struct {
			URLs []string "json:\"urls\""
		}{
			URLs: urls,
		},
	}

	file, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(configData); err != nil {
		t.Fatalf("Failed to write config data: %v", err)
	}

	return file.Name()
}

func TestLoadBalancerInitialization(t *testing.T) {
	// Setup mock servers
	mockServers := setupMockServers(2)
	defer func() {
		for _, s := range mockServers {
			s.server.Close()
		}
	}()

	// Create test config
	configFile := createTestConfig(t, mockServers)
	defer os.Remove(configFile)

	// Test configuration loading
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Servers.URLs) != 2 {
		t.Errorf("Expected 2 servers in config, got %d", len(cfg.Servers.URLs))
	}

	// Test load balancer creation
	lb := balancer.NewLoadBalancer(nil, cfg.LoadBalancer.Strategy)
	if lb == nil {
		t.Fatal("Failed to create load balancer")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	// Create a test server with the metrics endpoint
	lb := balancer.NewLoadBalancer(nil, "round_robin")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			metrics := lb.GetMetrics()
			json.NewEncoder(w).Encode(metrics)
		}
	}))
	defer server.Close()

	// Test metrics endpoint
	resp, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	var metrics balancer.Metrics
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		t.Fatalf("Failed to decode metrics: %v", err)
	}
}

func TestGracefulShutdown(t *testing.T) {

	// Create test load balancer
	lb := balancer.NewLoadBalancer(nil, "round_robin")

	// Create shutdown channel
	shutdown := make(chan struct{})
	done := make(chan struct{})

	// Start test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-shutdown:
			// Simulate graceful shutdown
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Add test server to load balancer
	err := lb.AddServer(server.URL)
	if err != nil {
		t.Fatalf("Failed to add server: %v", err)
	}

	// Start shutdown process in goroutine
	go func() {
		close(shutdown)
		lb.GracefulShutdown()
		close(done)
	}()

	// Wait for shutdown with timeout
	select {
	case <-done:
		// Shutdown completed successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown timeout")
	}

	// Verify server is not accepting new requests
	resp, err := http.Get(server.URL)
	if err == nil && resp.StatusCode != http.StatusServiceUnavailable {
		t.Error("Expected server to reject requests after shutdown")
	}
}

func TestLoadBalancerRequestHandling(t *testing.T) {
	// Setup mock backend servers
	mockServers := setupMockServers(2)
	defer func() {
		for _, s := range mockServers {
			s.server.Close()
		}
	}()

	// Create load balancer
	lb := balancer.NewLoadBalancer(nil, "round_robin")
	for _, s := range mockServers {
		if err := lb.AddServer(s.URL); err != nil {
			t.Fatalf("Failed to add server: %v", err)
		}
	}

	// Create test server with the load balancer
	server := httptest.NewServer(lb)
	defer server.Close()

	// Test request handling
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if len(body) == 0 {
		t.Error("Expected non-empty response body")
	}
}
