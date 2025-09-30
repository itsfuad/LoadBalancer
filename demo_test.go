package main

import (
	"fmt"
	"io"
	"loadbalancer/balancer"
	"log"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestLoadBalancerWithDemoServers spawns real demo servers and tests load balancing
func TestLoadBalancerWithDemoServers(t *testing.T) {
	// Build the demo server first
	t.Log("Building demo server...")
	buildCmd := exec.Command("go", "build", "-o", "demo_server.exe", "./demo/demo_server.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build demo server: %v", err)
	}
	defer os.Remove("demo_server.exe")

	// Define server configurations
	serverConfigs := []struct {
		id   int
		port int
	}{
		{id: 1, port: 9001},
		{id: 2, port: 9002},
		{id: 3, port: 9003},
	}

	// Start demo servers
	var processes []*exec.Cmd
	for _, cfg := range serverConfigs {
		cmd := exec.Command("./demo_server.exe",
			"-id", fmt.Sprintf("%d", cfg.id),
			"-port", fmt.Sprintf("%d", cfg.port))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start demo server %d: %v", cfg.id, err)
		}
		processes = append(processes, cmd)
		t.Logf("Started demo server %d on port %d", cfg.id, cfg.port)
	}

	// Cleanup: kill all demo servers when test completes
	defer func() {
		for i, cmd := range processes {
			if cmd.Process != nil {
				t.Logf("Stopping demo server %d", i+1)
				cmd.Process.Kill()
			}
		}
	}()

	// Wait for servers to be ready
	time.Sleep(2 * time.Second)

	// Verify servers are running
	for _, cfg := range serverConfigs {
		url := fmt.Sprintf("http://localhost:%d/health", cfg.port)
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("Server %d is not responding: %v", cfg.id, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Server %d health check failed: %v", cfg.id, resp.Status)
		}
		t.Logf("Server %d health check passed", cfg.id)
	}

	// Test Round Robin Strategy
	t.Run("RoundRobinStrategy", func(t *testing.T) {
		testStrategy(t, "round_robin", serverConfigs)
	})

	// Test Least Active Strategy
	t.Run("LeastActiveStrategy", func(t *testing.T) {
		testStrategy(t, "least_active", serverConfigs)
	})
}

func testStrategy(t *testing.T, strategy string, serverConfigs []struct{ id, port int }) {
	t.Logf("Testing %s strategy", strategy)

	logger := log.New(os.Stdout, fmt.Sprintf("[%s] ", strategy), log.LstdFlags)
	lb := balancer.NewLoadBalancer(logger, strategy)

	for _, cfg := range serverConfigs {
		url := fmt.Sprintf("http://localhost:%d", cfg.port)
		if err := lb.AddServer(url); err != nil {
			t.Fatalf("Failed to add server %d: %v", cfg.id, err)
		}
	}

	go lb.StartHealthChecks(5 * time.Second)
	time.Sleep(2 * time.Second)

	numRequests := 10
	t.Logf("Making %d requests to test load distribution", numRequests)

	serverHits, successfulRequests := sendRequestsThroughBalancer(t, lb, numRequests)

	verifyLoadDistribution(t, strategy, serverConfigs, serverHits, successfulRequests, numRequests)
}

func sendRequestsThroughBalancer(t *testing.T, lb *balancer.LoadBalancer, numRequests int) (map[string]int, int) {
	serverHits := make(map[string]int)
	successfulRequests := 0

	for i := 0; i < numRequests; i++ {
		req, err := http.NewRequest("GET", "http://test.local", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rw := &testResponseWriter{
			headers: make(http.Header),
			body:    make([]byte, 0),
		}

		lb.ServeHTTP(rw, req)

		if rw.statusCode == 0 {
			rw.statusCode = http.StatusOK
		}

		if rw.statusCode != http.StatusOK {
			t.Logf("Request %d failed with status: %d", i+1, rw.statusCode)
			continue
		}

		response := string(rw.body)
		serverHits[response]++
		successfulRequests++
		t.Logf("Request %d: %s", i+1, response)
	}
	return serverHits, successfulRequests
}

func verifyLoadDistribution(t *testing.T, strategy string, serverConfigs []struct{ id, port int }, serverHits map[string]int, successfulRequests, numRequests int) {
	t.Logf("\n=== Load Distribution Summary for %s ===", strategy)
	for response, count := range serverHits {
		t.Logf("%s: %d requests", response, count)
	}
	t.Logf("Total successful requests: %d/%d", successfulRequests, numRequests)

	if successfulRequests == 0 {
		t.Error("No successful requests - load balancer may not be working")
	}

	if strategy == "round_robin" && len(serverHits) > 1 {
		expectedHitsPerServer := float64(successfulRequests) / float64(len(serverConfigs))
		tolerance := expectedHitsPerServer * 0.5

		for response, hits := range serverHits {
			diff := abs(float64(hits) - expectedHitsPerServer)
			if diff > tolerance {
				t.Logf("Warning: Uneven distribution for %s: got %d hits, expected ~%.1f",
					response, hits, expectedHitsPerServer)
			}
		}
		t.Logf("Round robin distributed requests across %d servers", len(serverHits))
	}
}

// testResponseWriter is a custom ResponseWriter for testing
type testResponseWriter struct {
	headers    http.Header
	body       []byte
	statusCode int
}

func (w *testResponseWriter) Header() http.Header {
	return w.headers
}

func (w *testResponseWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *testResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// TestDemoServerDirectly tests demo servers directly without load balancer
func TestDemoServerDirectly(t *testing.T) {
	// Build the demo server
	t.Log("Building demo server...")
	buildCmd := exec.Command("go", "build", "-o", "demo_server_test.exe", "./demo/demo_server.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build demo server: %v", err)
	}
	defer os.Remove("demo_server_test.exe")

	// Start a single demo server
	cmd := exec.Command("./demo_server_test.exe", "-id", "99", "-port", "9099")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start demo server: %v", err)
	}
	defer cmd.Process.Kill()

	// Wait for server to be ready
	time.Sleep(1 * time.Second)

	// Test server endpoint
	resp, err := http.Get("http://localhost:9099/")
	if err != nil {
		t.Fatalf("Failed to connect to demo server: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	response := string(body)
	t.Logf("Demo server response: %s", response)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	// Verify response contains server ID
	expectedSubstring := "SERVER 99"
	if !contains(response, expectedSubstring) {
		t.Errorf("Expected response to contain '%s', got: %s", expectedSubstring, response)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
