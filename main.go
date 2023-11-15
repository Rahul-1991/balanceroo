package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

var (
	backendServers = []string{"localhost:3000", "localhost:4000", "localhost:5000", "localhost:6000"}
	activeServers  []string
	proxyMap       sync.Map
)

func isServerInActiveList(server string) bool {
	for i := 0; i < len(activeServers); i++ {
		if activeServers[i] == server {
			return true
		}
	}
	return false
}

func removeServerFromActiveList(server string) {
	for i := 0; i < len(activeServers); i++ {
		if activeServers[i] == server {
			activeServers = append(activeServers[:i], activeServers[i+1:]...)
		}
	}
}

func healthChecker() {
	for {
		currentTimestamp := time.Now().Unix()
		if currentTimestamp%10 == 0 {
			for i := 0; i < len(backendServers); i++ {
				conn, err := net.Dial("tcp", backendServers[i])
				if err == nil {
					if !isServerInActiveList(backendServers[i]) {
						activeServers = append(activeServers, backendServers[i])
					}
				} else {
					removeServerFromActiveList(backendServers[i])
					proxyMap.Delete(backendServers[i])
				}
				// Close the TCP connection after performing operations
				if conn != nil {
					err = conn.Close()
					if err != nil {
						fmt.Println("Error closing connection:", err)
						return
					}
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}

// Custom response writer to capture status code
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func main() {
	var counter int64 // Atomic counter variable of type int64

	// Create a custom handler function
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Increment the counter atomically
		atomic.AddInt64(&counter, 1)
		if int(counter) == len(activeServers)+1 {
			counter = 1
		}
		selectedServer := activeServers[counter-1]
		fmt.Println("Forwarding request to backend server...", selectedServer)

		// Set the addresses of the backend servers
		backendURL, err := url.Parse("http://" + selectedServer)
		if err != nil {
			panic(err)
		}

		// Get or create reverse proxies for backend server
		proxy, ok := proxyMap.Load(selectedServer)
		if !ok {
			backendProxy := httputil.NewSingleHostReverseProxy(backendURL)
			proxyMap.Store(selectedServer, backendProxy)
			proxy = backendProxy
		}

		// Capture the start time
		startTime := time.Now()

		// Serve HTTP using the reverse proxy
		recorder := &responseRecorder{ResponseWriter: w}
		if backendProxy, ok := proxy.(*httputil.ReverseProxy); ok {
			backendProxy.ServeHTTP(recorder, r)
			// Calculate the duration taken
			duration := time.Since(startTime)
			fmt.Printf("Host: %s StatusCode: %d ResponseTime: %v\n", selectedServer, recorder.status, duration)
		} else {
			http.Error(w, "Failed to handle request", http.StatusInternalServerError)
		}
	})

	go healthChecker()
	// Start the HTTP server on port 8080
	fmt.Println("Starting server on :80...")
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		panic(err)
	}
}
