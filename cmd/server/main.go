package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"autocomplete/internal/api"
	"autocomplete/internal/corpus"
	"autocomplete/internal/suggest"
	"autocomplete/web"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP server address")
	flag.Parse()

	items := corpus.Items()
	registry := suggest.NewRegistry(items, 20)

	mux := http.NewServeMux()
	api.New(registry).Mount(mux)
	mux.Handle("/", http.FileServer(http.FS(web.Files())))

	server := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("loaded %d suggestions", len(items))
	log.Printf("listening on http://localhost%s", *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
