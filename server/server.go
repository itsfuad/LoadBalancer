package server

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)

const HealthyKey = ":healthy"

type Server struct {
	URL     string
	Load    int
	Healthy bool
	mu      sync.Mutex
	client  *redis.Client
	ctx     context.Context
	logger  *log.Logger
}

func NewServer(url string, client *redis.Client, ctx context.Context, logger *log.Logger) *Server {
	return &Server{
		URL:     url,
		Healthy: true,
		client:  client,
		ctx:     ctx,
		logger:  logger,
	}
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.Healthy {
		http.Error(w, "Server is not healthy", http.StatusServiceUnavailable)
		s.logger.Printf("Rejected request to %s as server is not healthy", s.URL)
		return
	}

	s.Load++
	s.client.Incr(s.ctx, s.URL+":load")
	s.logger.Printf("Handling request on server %s. Current load: %d\n", s.URL, s.Load)

	// Simulate request processing
	time.Sleep(2 * time.Second)

	s.Load--
	s.client.Decr(s.ctx, s.URL+":load")
	s.logger.Printf("Finished request on server %s. Current load: %d\n", s.URL, s.Load)
}

func (s *Server) CheckHealth() {
	resp, err := http.Get(s.URL + "/health")
	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil || resp.StatusCode != http.StatusOK {
		s.Healthy = false
		s.client.Set(s.ctx, s.URL+healthyKey, false, 0)
		s.logger.Printf("Server %s is unhealthy\n", s.URL)
	} else {
		s.Healthy = true
		s.client.Set(s.ctx, s.URL+healthyKey, true, 0)
		s.logger.Printf("Server %s is healthy\n", s.URL)
	}
}