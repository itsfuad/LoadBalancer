package server

import (
	"fmt"
	"io"
	"loadbalancer/utils"
	"log"
	"net/http"
	"sync"
)

const HealthyKey = ":healthy"

type Server struct {
	URL     string
	Load    int
	Healthy bool
	mu      sync.Mutex
	logger  *log.Logger
}

func NewServer(url string, logger *log.Logger) *Server {
	return &Server{
		URL:     url,
		Healthy: true,
		logger:  logger,
	}
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.Healthy {
		http.Error(w, "Server is not healthy", http.StatusServiceUnavailable)
		s.logger.Println(utils.Colorize(fmt.Sprintf("Rejected request to %s as server is not healthy", s.URL), utils.RED))
		return
	}

	s.Load++

	s.logger.Println(PrintState(s.URL, s.Load, false))

	// Create a new request based on the original request
	req, err := http.NewRequest(r.Method, s.URL+r.RequestURI, r.Body)
	if err != nil {
		http.Error(w, "Error occurred while creating request to target server", http.StatusInternalServerError)
		s.logger.Println(utils.Colorize(fmt.Sprintf("Error occurred while creating request to target server %s: %v", s.URL, err), utils.RED))
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
	if err != nil {
		http.Error(w, "Error occurred while making request to target server", http.StatusInternalServerError)
		s.logger.Println(utils.Colorize(fmt.Sprintf("Error occurred while making request to target server %s, Error:%v, Status code: %d\n", s.URL, err, res.StatusCode), utils.RED))
		s.Load--
		return
	} else if res.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Server responded with status code %d for resource %s", res.StatusCode, r.URL.Path), res.StatusCode)
		if res.StatusCode >= 300 && res.StatusCode < 308 {
			http.Redirect(w, r, res.Header.Get("Location"), res.StatusCode)
		}
		s.logger.Println(utils.Colorize(fmt.Sprintf("Server responded with status code %d for resource %s", res.StatusCode, r.URL.Path), utils.RED))
	}
	defer res.Body.Close()

	// Relay back the response
	for k, v := range res.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(res.StatusCode)
	if _, err := io.Copy(w, res.Body); err != nil {
		s.logger.Println(utils.Colorize(fmt.Sprintf("Failed to relay response body from target server %s: %v", s.URL, err), utils.RED))
	}

	s.Load--
	//s.logger.Printf("Finished request on server %s. Current load: %d\n", s.URL, s.Load)
	s.logger.Println(PrintState(s.URL, s.Load, true))
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
