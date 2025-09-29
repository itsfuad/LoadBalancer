package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	RedisHost     string
	RedisPort     string
	RedisPassword string
	LoadBalancer  struct {
		Port                       int    `json:"port"`
		HealthCheckIntervalSeconds int    `json:"health_check_interval_seconds"`
		Strategy                   string `json:"strategy"` // "least_active" or "round_robin"
	} `json:"load_balancer"`
	Servers struct {
		URLs []string `json:"urls"`
	} `json:"servers"`
}

func LoadConfig() (*Config, error) {

	filename := "servers.json"

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	if err := json.NewDecoder(file).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}
