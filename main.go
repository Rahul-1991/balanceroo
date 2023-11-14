package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var backendServers = []string{"http://localhost:3000", "http://localhost:4000"}

func main() {
	count := 0

	// Create a custom handler function
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		selectedServer := backendServers[count]
		fmt.Println("Forwarding request to backend server...", selectedServer)

		// Set the addresses of the backend servers
		backendURL, err := url.Parse(selectedServer)
		if err != nil {
			panic(err)
		}

		// Create reverse proxies for backend server
		proxy := httputil.NewSingleHostReverseProxy(backendURL)

		proxy.ServeHTTP(w, r)
		count += 1
		if count == 2 {
			count = 0
		}
	})

	// Start the HTTP server on port 8080
	fmt.Println("Starting server on :80...")
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		panic(err)
	}
}
