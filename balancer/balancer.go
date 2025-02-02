package balancer

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	sv "loadbalancer/server"
	"loadbalancer/utils"
)

type LoadBalancer struct {
	Servers     []*sv.Server
	mu          sync.RWMutex
	Logger      *log.Logger
	wg          sync.WaitGroup
	shutdown    bool
	metrics     *Metrics
	maxRetries  int
}

type Metrics struct {
	TotalRequests uint64
	FailedRequests uint64
	ActiveConnections int64
}

func NewLoadBalancer(logger *log.Logger) *LoadBalancer {
	if logger == nil {
		logger = log.New(io.Discard, "", log.LstdFlags)
	}
	return &LoadBalancer{
		Logger: logger,
		metrics: &Metrics{},
		maxRetries: 3,
	}
}

func (lb *LoadBalancer) AddServer(url string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	// Validate server URL
	if url == "" {
		return errors.New("server URL cannot be empty")
	}
	
	// Check for duplicate servers
	for _, server := range lb.Servers {
		if server.URL == url {
			return fmt.Errorf("server %s already exists", url)
		}
	}
	
	server := sv.NewServer(url, lb.Logger)
	lb.Servers = append(lb.Servers, server)
	lb.Logger.Println(utils.Colorize("Added server "+url+" to the load balancer", utils.GREEN))
	return nil
}

func (lb *LoadBalancer) RemoveServer(url string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	for i, server := range lb.Servers {
		if server.URL == url {
			// Wait for active connections to finish
			for server.Load > 0 {
				lb.mu.Unlock()
				time.Sleep(100 * time.Millisecond)
				lb.mu.Lock()
			}
			
			// Remove server
			lb.Servers = append(lb.Servers[:i], lb.Servers[i+1:]...)
			lb.Logger.Println(utils.Colorize("Removed server "+url, utils.YELLOW))
			return nil
		}
	}
	
	return fmt.Errorf("server %s not found", url)
}

func (lb *LoadBalancer) GetMetrics() *Metrics {
	return &Metrics{
		TotalRequests: atomic.LoadUint64(&lb.metrics.TotalRequests),
		FailedRequests: atomic.LoadUint64(&lb.metrics.FailedRequests),
		ActiveConnections: atomic.LoadInt64(&lb.metrics.ActiveConnections),
	}
}

func (lb *LoadBalancer) GetLeastLoadedServer() *sv.Server {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var leastLoadedServer *sv.Server
	for _, server := range lb.Servers {
		healthy := server.Healthy
		load := server.Load

		if healthy && (leastLoadedServer == nil || load < leastLoadedServer.Load) {
			server.Load = load
			leastLoadedServer = server
		}
	}

	if leastLoadedServer != nil {
		lb.Logger.Println(utils.Colorize("Selected server "+leastLoadedServer.URL+" with load "+fmt.Sprint(leastLoadedServer.Load), utils.BLUE))
	}

	return leastLoadedServer
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Increment metrics
	atomic.AddUint64(&lb.metrics.TotalRequests, 1)
	atomic.AddInt64(&lb.metrics.ActiveConnections, 1)
	defer atomic.AddInt64(&lb.metrics.ActiveConnections, -1)

	// Check if we're shutting down
	if lb.shutdown {
		http.Error(w, "Service is shutting down", http.StatusServiceUnavailable)
		return
	}

	// Try multiple servers if needed
	var err error
	for retry := 0; retry < lb.maxRetries; retry++ {
		server := lb.GetLeastLoadedServer()
		if server == nil {
			continue
		}

		err = server.HandleRequest(w, r)
		if err == nil {
			return
		}

		lb.Logger.Printf("Request failed on server %s, attempt %d: %v", server.URL, retry+1, err)
	}

	// All retries failed
	atomic.AddUint64(&lb.metrics.FailedRequests, 1)
	http.Error(w, "All servers failed to process the request", http.StatusServiceUnavailable)
}

func (lb *LoadBalancer) StartHealthChecks(interval time.Duration) {
	for _, server := range lb.Servers {
		go func(s *sv.Server) {
			for {
				s.CheckHealth()
				time.Sleep(interval)
			}
		}(server)
	}
}

func (lb *LoadBalancer) GracefulShutdown() {
	// Notify about the shutdown process
	lb.Logger.Println(utils.Colorize("Shutting down load balancer gracefully", utils.YELLOW))

	// Stop accepting new requests
	lb.mu.Lock()
	lb.shutdown = true
	lb.mu.Unlock()

	// Wait for ongoing requests to complete
	lb.wg.Wait()

	lb.Logger.Println(utils.Colorize("All servers have been shut down, and connections are closed.", utils.YELLOW))
}
