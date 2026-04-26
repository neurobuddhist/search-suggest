package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/neurobuddhist/search-suggest/internal/api"
	"github.com/neurobuddhist/search-suggest/internal/corpus"
	"github.com/neurobuddhist/search-suggest/internal/suggest"
	"github.com/neurobuddhist/search-suggest/web"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP server address")
	flag.Parse()

	items := corpus.Items()
	registry := suggest.NewRegistry(items, 20)

	mux := http.NewServeMux()
	api.New(registry).Mount(mux)
	staticFiles, err := web.Files()
	if err != nil {
		log.Fatalf("load static files: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFiles)))

	server := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("loaded %d suggestions", len(items))
	log.Printf("listening on http://%s", displayAddr(*addr))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func displayAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err == nil {
		if host == "" || host == "0.0.0.0" || host == "::" {
			host = "localhost"
		}
		return net.JoinHostPort(host, port)
	}

	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}
