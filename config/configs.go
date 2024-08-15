package config

import (
	"encoding/json"
	"errors"
	"os"

	 "github.com/joho/godotenv"
)

type Config struct {
	RedisHost     string
	RedisPort     string
	RedisPassword string
	LoadBalancer  struct {
		Port                       int `json:"port"`
		HealthCheckIntervalSeconds int `json:"health_check_interval_seconds"`
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

	err = godotenv.Load()
	if err != nil {
	   return nil, errors.New("error reading .env")
	}

	//read environment variables
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		config.RedisHost = redisHost
	} else {
		return nil, errors.New("REDIS_HOST environment variable is not set")
	}

	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		config.RedisPort = redisPort
	} else {
		return nil, errors.New("REDIS_PORT environment variable is not set")
	}

	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		config.RedisPassword = redisPassword
	} else {
		return nil, errors.New("REDIS_PASSWORD environment variable is not set")
	}

	return config, nil
}