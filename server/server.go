package server

import (
	"bytes"
	"fmt"
	"io"
	"loadbalancer/utils"
	"log"
	"net/http"
	"sync"
	"time"
)

const HealthyKey = ":healthy"

type Server struct {
	URL           string
	Load          int
	Healthy       bool
	LastChecked   time.Time
	ResponseTimes []time.Duration
	mu            sync.RWMutex
	logger        *log.Logger
}

func NewServer(url string, logger *log.Logger) *Server {
	return &Server{
		URL:     url,
		Healthy: true,
		logger:  logger,
	}
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) error {
	s.mu.Lock()
	if !s.Healthy {
		s.mu.Unlock()
		return fmt.Errorf("server %s is not healthy", s.URL)
	}
	s.Load++
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.Load--
		s.mu.Unlock()
	}()

	start := time.Now()

	// Create new request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %v", err)
	}
	
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	req, err := http.NewRequest(r.Method, s.URL+r.RequestURI, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Copy headers
	utils.CopyHeaders(req.Header, r.Header)

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	// Update response times
	duration := time.Since(start)
	s.updateResponseTime(duration)

	// Copy response
	utils.CopyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response: %v", err)
	}

	return nil
}

func (s *Server) updateResponseTime(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ResponseTimes = append(s.ResponseTimes, duration)
	if len(s.ResponseTimes) > 100 {
		s.ResponseTimes = s.ResponseTimes[1:]
	}
}

func PrintState(url string, load int, finished bool) string {
	if finished {
		return fmt.Sprintf("Finished request on server %s. Current load: %s\n", utils.Colorize(url, utils.GREEN), utils.Colorize(fmt.Sprintf("%d", load), utils.YELLOW))
	}
	return fmt.Sprintf("Handling request on server %s. Current load: %s\n", utils.Colorize(url, utils.GREEN), utils.Colorize(fmt.Sprintf("%d", load), utils.YELLOW))
}

func (s *Server) CheckHealth() {
	resp, err := http.Get(s.URL + "/")
	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil || resp.StatusCode != http.StatusOK {
		s.Healthy = false
		s.logger.Println(utils.Colorize(fmt.Sprintf("Server %s is unhealthy\n", s.URL), utils.RED))
	} else {
		s.Healthy = true
	}
}
