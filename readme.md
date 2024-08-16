# Load Balancer

## Overview
This project is a Go-based load balancer designed to distribute incoming HTTP requests to a pool of backend servers based on their current load and health status. It uses Redis for storing server state information such as load and health status, allowing for a robust and scalable architecture.

## Features
- **Dynamic Load Balancing:** Distributes requests to the least loaded server.
- **Health Checks:** Regularly checks the health of each server and only routes traffic to healthy servers.
- **Graceful Shutdown:** Ensures that ongoing requests are handled before shutting down the load balancer.
- **Configuration via JSON:** Server details and settings are loaded from a JSON configuration file.
- **Detailed Logging:** Provides comprehensive logging for monitoring and debugging.

## Prerequisites
- Go 1.16 or later
- Redis server (locally or remotely accessible)

## Installation

1. **Clone the Repository:**
```sh
git clone https://github.com/itsfuad/LoadBalancer.git
cd LoadBalancer
```
2. **Build the Load Balancer:**
```sh
go build -o LoadBalancer main.go
```

## Configuration
**1. Create a JSON Configuration File:**
Create a servers.json file in the root directory with the following structure:
```json
{
    "load_balancer": {
        "port": 8080, // Port on which the load balancer will listen
        "health_check_interval_seconds": 10 // Interval in seconds for health checks
    },
    "servers": {
        "urls": [
            "http://localhost:9001", // server 1
            "http://localhost:9002", // server 2
            "http://localhost:9003"  // server 3
        ]
    }
}
```
+ port: The port on which the load balancer will listen.
+ health_check_interval_seconds: Interval in seconds for health checks.
+ urls: List of backend server URLs.

**2. Running the Load Balancer:**
Start the load balancer by executing:
```sh
./LoadBalancer
```
The load balancer will start on the specified port and begin routing traffic to the least loaded server.

## Usage
**Sending Requests:**
Send HTTP requests to the load balancer's address (e.g., http://localhost:8080). The load balancer will forward the request to the backend server with the least load.

**Server Health Check:**
The load balancer periodically checks each server's health by sending a GET request to the /health endpoint of each server. If a server does not respond with a status code of 200, it is marked as unhealthy and temporarily removed from the load balancer's pool.

**Graceful Shutdown:**
The load balancer handles OS signals (SIGINT, SIGTERM) to perform a graceful shutdown, allowing in-progress requests to complete before terminating.

## Extending the Load Balancer
You can extend this load balancer by:

+ Adding more sophisticated load balancing algorithms.
+ Implementing server auto-scaling features.
+ Adding support for HTTPS requests.
+ Creating a user interface for monitoring server status and load in real-time.
## Contributing
Feel free to contribute by opening issues or submitting pull requests. Make sure to follow the existing coding style and include relevant tests for any new features or bug fixes.

## License
This project is licensed under the GNU License. See the LICENSE file for more information.
