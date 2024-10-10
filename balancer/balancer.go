package balancer

import "fmt"

import (
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/context"

	sv "loadbalancer/server"
	"loadbalancer/utils"
)


type LoadBalancer struct {
	Servers []*sv.Server
	mu      sync.Mutex
	Ctx     context.Context
	Logger  *log.Logger
	wg      sync.WaitGroup
	shutdown bool
}

func (lb *LoadBalancer) AddServer(url string) {
	server := sv.NewServer(url, lb.Ctx, lb.Logger)
	lb.Servers = append(lb.Servers, server)
	//lb.Logger.Printf("Added server %s to the load balancer", url)
	lb.Logger.Println(utils.Colorize("Added server "+url+" to the load balancer", utils.GREEN))
}

func (lb *LoadBalancer) GetLeastLoadedServer() *sv.Server {
	lb.mu.Lock()
	defer lb.mu.Unlock()

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
	server := lb.GetLeastLoadedServer()
	if server != nil {
		server.HandleRequest(w, r)
	} else {
		http.Error(w, "No servers available", http.StatusServiceUnavailable)
		lb.Logger.Println(utils.Colorize("No servers available to handle the request", utils.RED))
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
	lb.Logger.Println(utils.Colorize("Shutting down load balancer gracefully", utils.YELLOW))

	// Stop accepting new requests
	lb.mu.Lock()
	lb.shutdown = true
	lb.mu.Unlock()

	// Wait for ongoing requests to complete
	lb.wg.Wait()

	lb.Logger.Println(utils.Colorize("All servers have been shut down, and connections are closed.", utils.YELLOW))
}