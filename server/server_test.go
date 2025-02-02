package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const URL = "http://example.com"

func TestNewServer(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)
	server := NewServer(URL, logger)

	if server.URL != URL {
		t.Errorf("expected URL to be 'http://example.com', got %s", server.URL)
	}
	if !server.Healthy {
		t.Errorf("expected server to be healthy")
	}
	if server.logger != logger {
		t.Errorf("expected logger to be set")
	}
}

func TestHandleRequest(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)
	server := NewServer(URL, logger)

	req := httptest.NewRequest(http.MethodGet, URL, nil)
	w := httptest.NewRecorder()

	err := server.HandleRequest(w, req)
	if err == nil {
		t.Errorf("expected error due to server being unreachable")
	}

	server.Healthy = false
	err = server.HandleRequest(w, req)
	if err == nil {
		t.Errorf("expected error due to server being unhealthy")
	}
}

func TestUpdateResponseTime(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)
	server := NewServer(URL, logger)

	duration := 100 * time.Millisecond
	server.updateResponseTime(duration)

	if len(server.ResponseTimes) != 1 {
		t.Errorf("expected 1 response time, got %d", len(server.ResponseTimes))
	}
	if server.ResponseTimes[0] != duration {
		t.Errorf("expected response time to be %v, got %v", duration, server.ResponseTimes[0])
	}
}

func TestPrintState(t *testing.T) {
	load := 5
	finished := true

	expected := fmt.Sprintf("Finished request on server \033[32m%s\033[0m. Current load: \033[33m5\033[0m\n", URL)
	result := PrintState(URL, load, finished)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}

	finished = false
	expected = fmt.Sprintf("Handling request on server \033[32m%s\033[0m. Current load: \033[33m5\033[0m\n", URL)
	result = PrintState(URL, load, finished)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestCheckHealth(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)
	server := NewServer("http://example.com", logger)

	server.CheckHealth()
	if !server.Healthy {
		t.Errorf("expected server to be healthy")
	}
}