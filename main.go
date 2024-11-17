package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"loadbalancer/balancer"
	"loadbalancer/config"
	"loadbalancer/utils"
)

func main() {
	// Setup logger
	logger := log.New(os.Stdout, "load-balancer: ", log.LstdFlags)

	// Load configuration and initialize Redis
	config, err := config.LoadConfig()
	if err != nil {
		logger.Fatalf("Error loading configuration: %v\n", err)
	}

	// Create load balancer
	lb := &balancer.LoadBalancer{
		Logger: logger,
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
	logger.Println(utils.Colorize(fmt.Sprintf("Load balancer is running on port %d", port), utils.GREEN))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), lb); err != nil {
		logger.Fatalf("Load balancer failed: %v\n", err)
	}
}
