package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

//go:embed static/*
var assets embed.FS

// Local diagnostic UI; never expose it as part of the production deployment.
func main() {
	target := flag.String("api", "http://localhost:8080", "Backend origin")
	flag.Parse()
	backend, err := url.Parse(*target)
	if err != nil || backend.Host == "" || (backend.Scheme != "http" && backend.Scheme != "https") {
		log.Fatal("invalid API origin")
	}
	files, err := fs.Sub(assets, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", httputil.NewSingleHostReverseProxy(backend))
	mux.Handle("/", http.FileServer(http.FS(files)))
	server := &http.Server{Addr: "127.0.0.1:8081", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Print("Push debug: http://localhost:8081 (local only)")
	log.Fatal(server.ListenAndServe())
}
