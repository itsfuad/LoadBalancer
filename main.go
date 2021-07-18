package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"


	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"

	"loadbalancer/config"
	"loadbalancer/server"
)

type LoadBalancer struct {
	Servers []*server.Server
	mu      sync.Mutex
	client  *redis.Client
	ctx     context.Context
	logger  *log.Logger
}

func (lb *LoadBalancer) AddServer(url string) {
	server := server.NewServer(url, lb.client, lb.ctx, lb.logger)
	lb.Servers = append(lb.Servers, server)
	lb.client.Set(lb.ctx, url+":load", 0, 0)
	lb.client.Set(lb.ctx, url+server.HealthyKey, true, 0)
	lb.logger.Printf("Added server %s to the load balancer", url)
}

func (lb *LoadBalancer) GetLeastLoadedServer() *server.Server {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	var leastLoadedServer *server.Server
	for _, server := range lb.Servers {
		load, err := lb.client.Get(lb.ctx, server.URL+":load").Int()
		if err != nil {
			lb.logger.Printf("Error getting load for server %s: %v\n", server.URL, err)
			continue
		}

		healthy, err := lb.client.Get(lb.ctx, server.URL+server.HealthyKey).Bool()
		if err != nil {
			lb.logger.Printf("Error getting health status for server %s: %v\n", server.URL, err)
			continue
		}

		if healthy && (leastLoadedServer == nil || load < leastLoadedServer.Load) {
			server.Load = load
			leastLoadedServer = server
		}
	}

	if leastLoadedServer != nil {
		lb.logger.Printf("Selected server %s with load %d\n", leastLoadedServer.URL, leastLoadedServer.Load)
	}

	return leastLoadedServer
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server := lb.GetLeastLoadedServer()
	if server != nil {
		server.HandleRequest(w, r)
	} else {
		http.Error(w, "No servers available", http.StatusServiceUnavailable)
		lb.logger.Println("No servers available to handle the request")
	}
}

func (lb *LoadBalancer) StartHealthChecks(interval time.Duration) {
	for _, server := range lb.Servers {
		go func(s *server.Server) {
			for {
				s.CheckHealth()
				time.Sleep(interval)
			}
		}(server)
	}
}

func (lb *LoadBalancer) GracefulShutdown() {
	// Code to gracefully shutdown servers and connections
	lb.logger.Println("Shutting down load balancer gracefully")
	// Implement any necessary cleanup or final logging
}



func LoadConfig(filename string) (*config.Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &config.Config{}
	if err := json.NewDecoder(file).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

func main() {
	// Setup logger
	logger := log.New(os.Stdout, "load-balancer: ", log.LstdFlags)

	// Load configuration and initialize Redis
	config, err := LoadConfig("servers.json")
	if err != nil {
		logger.Fatalf("Error loading configuration: %v\n", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     config.LoadBalancer.RedisAddress,
		Password: config.LoadBalancer.RedisPassword,
		DB:       config.LoadBalancer.RedisDB,
	})
	ctx := context.Background()

	// Create load balancer
	lb := &LoadBalancer{
		client: client,
		ctx:    ctx,
		logger: logger,
	}

	// Add servers from configuration
	for _, url := range config.Servers.URLs {
		lb.AddServer(url)
	}

	// Start health checks
	healthCheckInterval := time.Duration(config.LoadBalancer.HealthCheckIntervalSeconds) * time.Second
	lb.StartHealthChecks(healthCheckInterval)

	// Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stop
		lb.GracefulShutdown()
		os.Exit(0)
	}()

	// Start the load balancer
	port := config.LoadBalancer.Port
	logger.Printf("Load balancer is running on port %d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), lb); err != nil {
		logger.Fatalf("Load balancer failed: %v\n", err)
	}
}
