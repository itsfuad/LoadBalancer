package utils

import (
	"net/http"
	"testing"
)

func TestCopyHeaders(t *testing.T) {
	src := http.Header{}
	src.Add("Content-Type", "application/json")
	src.Add("X-Custom-Header", "custom-value")

	dst := http.Header{}
	CopyHeaders(dst, src)

	if dst.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type to be 'application/json', got '%s'", dst.Get("Content-Type"))
	}

	if dst.Get("X-Custom-Header") != "custom-value" {
		t.Errorf("expected X-Custom-Header to be 'custom-value', got '%s'", dst.Get("X-Custom-Header"))
	}
}

func TestGetClientIP(t *testing.T) {
	req := &http.Request{
		Header: http.Header{},
	}

	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	ip := GetClientIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("expected IP to be '192.168.1.1', got '%s'", ip)
	}

	req.Header.Del("X-Forwarded-For")
	req.Header.Set("X-Real-IP", "10.0.0.1")
	ip = GetClientIP(req)
	if ip != "10.0.0.1" {
		t.Errorf("expected IP to be '10.0.0.1', got '%s'", ip)
	}

	req.Header.Del("X-Real-IP")
	req.RemoteAddr = "127.0.0.1:8080"
	ip = GetClientIP(req)
	if ip != "127.0.0.1:8080" {
		t.Errorf("expected IP to be '127.0.0.1:8080', got '%s'", ip)
	}
}
