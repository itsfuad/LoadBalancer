package config


type Config struct {
	LoadBalancer struct {
		Port                       int    `json:"port"`
		RedisAddress               string `json:"redis_address"`
		RedisPassword              string `json:"redis_password"`
		RedisDB                    int    `json:"redis_db"`
		HealthCheckIntervalSeconds int    `json:"health_check_interval_seconds"`
	} `json:"load_balancer"`
	Servers struct {
		URLs []string `json:"urls"`
	} `json:"servers"`
}