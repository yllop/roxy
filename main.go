package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gorilla/handlers"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	backend = kingpin.Flag("backend-url", "Backend URL to connect to").Required().URL()

	authuser = kingpin.Flag("username", "Basic auth username").Default("bubbles").String()
	authpass = kingpin.Flag("password", "Basic auth password").Default("bubbles").String()

	listen = kingpin.Flag("listen", "port to listen on").Default("8080").Uint64()

	// pkey = kingpin.Flag("cert", "SSL private key file path").String()
	// cert = kingpin.Flag("cert", "SSL certiticate file path").String()
)

// rproxy is the authenticated ReverseProxy
type rproxy struct {
	proxy *httputil.ReverseProxy // stdlib ReverseProxy does the proxying
}

// New returns an rproxy that acts as an authenticated reverse proxy for the given backend
func New(backend *url.URL) *rproxy {
	return &rproxy{proxy: httputil.NewSingleHostReverseProxy(backend)}
}

// Make rproxy adhere to the http.Handler interface
func (p *rproxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Prepare an "unauthorized" response
	unauthorized := func(bodyStr string) {
		w.Header().Add("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(bodyStr))
	}
	// forward the request to the actual reverse proxy
	forward := func(resp http.ResponseWriter, req *http.Request, user string) {
		// log.Println("Setting header as", "X-HTTP-USER", user)
		r.Header.Set("X-HTTP-USER", user)
		p.proxy.ServeHTTP(w, r)
	}

	user, pass, ok := r.BasicAuth()
	if !ok {
		unauthorized("Unable to extract basic auth credentials")
		return
	}
	if user == "" {
		unauthorized("Missing username")
		return
	}
	if *authuser == "*" {
		forward(w, r, user)
		return
	} else {
		if user != *authuser || pass != *authpass {
			unauthorized("Incorrect auth credentials")
			return
		} else {
			forward(w, r, user)
			return
		}
	}
}

func main() {
	kingpin.Parse()

	log.Println("Setting up handlers")
	rproxy := New(*backend)
	logger := handlers.LoggingHandler(os.Stdout, rproxy)

	log.Println("Starting reverse proxy server for", (*backend).String(), "on", *listen)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *listen), logger)

	log.Println("Reverse proxy server exiting")
	if err != nil {
		log.Println("Error:", err)
	}
}
