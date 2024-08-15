package server

import (
	"io"
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/context"
)

const HealthyKey = ":healthy"

type Server struct {
	URL     string
	Load    int
	Healthy bool
	mu      sync.Mutex
	Ctx     context.Context
	logger  *log.Logger
}

func NewServer(url string, ctx context.Context, logger *log.Logger) *Server {
	return &Server{
		URL:     url,
		Healthy: true,
		Ctx:     ctx,
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
    s.logger.Printf("Handling request on server %s. Current load: %d\n", s.URL, s.Load)

    // Create a new request based on the original request
    req, err := http.NewRequest(r.Method, s.URL+r.RequestURI, r.Body)
    if err != nil {
        http.Error(w, "Error occurred while creating request to target server", http.StatusInternalServerError)
        s.logger.Printf("Error occurred while creating request to target server: %v", err)
        s.Load--
        return
    }

    // Copy the original request headers to the new request
    for key, values := range r.Header {
        for _, value := range values {
            req.Header.Add(key, value)
        }
    }

    // Make the request to the target server
    client := &http.Client{}
    res, err := client.Do(req)
    if err != nil || res.StatusCode != http.StatusOK {
        http.Error(w, "Error occurred while making request to target server", http.StatusInternalServerError)
        s.logger.Printf("Error occurred while making request to target server: %v", err)
        s.Load--
        return
    }
    defer res.Body.Close()

    // Relay back the response
    for k, v := range res.Header {
        w.Header()[k] = v
    }
    w.WriteHeader(res.StatusCode)
    if _, err := io.Copy(w, res.Body); err != nil {
        s.logger.Printf("Failed to relay response body from target server: %v", err)
    }

    s.Load--
    s.logger.Printf("Finished request on server %s. Current load: %d\n", s.URL, s.Load)
}


func (s *Server) CheckHealth() {
	resp, err := http.Get(s.URL + "/")
	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil || resp.StatusCode != http.StatusOK {
		s.Healthy = false
		s.logger.Printf("Server %s is unhealthy\n", s.URL)
	} else {
		s.Healthy = true
		s.logger.Printf("Server %s is healthy\n", s.URL)
	}
}