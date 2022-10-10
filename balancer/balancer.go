package balancer

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"

	sv "loadbalancer/server"
)


type LoadBalancer struct {
	Servers []*sv.Server
	mu      sync.Mutex
	Client  *redis.Client
	Ctx     context.Context
	Logger  *log.Logger
	wg      sync.WaitGroup
	shutdown bool
}

func (lb *LoadBalancer) AddServer(url string) {
	server := sv.NewServer(url, lb.Client, lb.Ctx, lb.Logger)
	lb.Servers = append(lb.Servers, server)
	lb.Client.Set(lb.Ctx, url+":load", 0, 0)
	lb.Client.Set(lb.Ctx, url+sv.HealthyKey, true, 0)
	lb.Logger.Printf("Added server %s to the load balancer", url)
}

func (lb *LoadBalancer) GetLeastLoadedServer() *sv.Server {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	var leastLoadedServer *sv.Server
	for _, server := range lb.Servers {
		load, err := lb.Client.Get(lb.Ctx, server.URL+":load").Int()
		if err != nil {
			lb.Logger.Printf("Error getting load for server %s: %v\n", server.URL, err)
			continue
		}

		healthy, err := lb.Client.Get(lb.Ctx, server.URL+sv.HealthyKey).Bool()
		if err != nil {
			lb.Logger.Printf("Error getting health status for server %s: %v\n", server.URL, err)
			continue
		}

		if healthy && (leastLoadedServer == nil || load < leastLoadedServer.Load) {
			server.Load = load
			leastLoadedServer = server
		}
	}

	if leastLoadedServer != nil {
		lb.Logger.Printf("Selected server %s with load %d\n", leastLoadedServer.URL, leastLoadedServer.Load)
	}

	return leastLoadedServer
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server := lb.GetLeastLoadedServer()
	if server != nil {
		server.HandleRequest(w, r)
	} else {
		http.Error(w, "No servers available", http.StatusServiceUnavailable)
		lb.Logger.Println("No servers available to handle the request")
	}
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
	lb.Logger.Println("Shutting down load balancer gracefully")

	// Stop accepting new requests
	lb.mu.Lock()
	lb.shutdown = true
	lb.mu.Unlock()

	// Wait for ongoing requests to complete
	lb.wg.Wait()

	// Close Redis client connection
	if err := lb.Client.Close(); err != nil {
		lb.Logger.Printf("Error closing Redis connection: %v", err)
	}

	lb.Logger.Println("All servers have been shut down, and connections are closed.")
}