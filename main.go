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

// Our RerverseProxy object
type rproxy struct {
	proxy *httputil.ReverseProxy // instance of Go ReverseProxy that will do the job for us
}

// factory
func NewAuthProxy(target *url.URL) *rproxy {
	return &rproxy{proxy: httputil.NewSingleHostReverseProxy(target)}
}

// Make Prox adhere to the Handler interface
func (p *rproxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		w.Header().Add("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Missing basic auth credentials"))
		return
	}
	if user != *authuser || pass != *authpass {
		w.Header().Add("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Incorrect auth credentials"))
		return
	}
	// Set header and let the request propagate
	log.Println("Setting header as", "X-HTTP-USER", *authuser)
	r.Header.Set("X-HTTP-USER", *authuser)
	log.Println("Letting req through after setting header")
	p.proxy.ServeHTTP(w, r)
}

func main() {
	kingpin.Parse()

	log.Println("Setting up handlers")
	rproxy := NewAuthProxy(*backend)
	logger := handlers.LoggingHandler(os.Stdout, rproxy)

	log.Println("Starting reverse proxy server for", (*backend).String(), "on", *listen)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *listen), logger)

	log.Println("Reverse proxy server exiting")
	if err != nil {
		log.Println("Error:", err)
	}
}
