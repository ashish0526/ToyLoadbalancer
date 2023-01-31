package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	isAlive() bool
	serve(rw http.ResponseWriter, r *http.Request)
}

type simplerServer struct {
	addr  string
	proxy httputil.ReverseProxy
}

func newSimpleServer(addr string) *simplerServer {
	serverUrl, err := url.Parse(addr)
	handleErr(err)

	return &simplerServer{
		addr:  addr,
		proxy: *httputil.NewSingleHostReverseProxy(serverUrl),
	}

}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	server          []Server
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		server:          servers,
	}
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("Error %v\n", err)
		os.Exit(1)
	}
}

func (s *simplerServer) Address() string {
	return s.addr
}

func (s *simplerServer) isAlive() bool {
	return true
}

func (s *simplerServer) serve(rw http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(rw, r)
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.server[lb.roundRobinCount%len(lb.server)]
	for !server.isAlive() {
		lb.roundRobinCount++
		server = lb.server[lb.roundRobinCount%len(lb.server)]
	}
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, r *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("Forwarding request to address %q \n", targetServer.Address())
	targetServer.serve(rw, r)
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.duckduckgo.com"),
		newSimpleServer("http://www.facebook.com"),
		newSimpleServer("http://www.primevideo.com"),
	}

	lb := NewLoadBalancer("8002", servers)
	handleRedirect := func(rw http.ResponseWriter, r *http.Request) {
		lb.serveProxy(rw, r)
	}
	http.HandleFunc("/", handleRedirect)

	fmt.Printf("Serving traffic on localhost %s \n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
