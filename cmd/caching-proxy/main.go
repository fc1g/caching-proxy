package main

import (
	"log"
	"net/url"
	"os"

	"github.com/fc1g/caching-proxy/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("load config: %v", err)
		os.Exit(1)
	}

	if err := run(cfg.Port, cfg.Origin); err != nil {
		log.Printf("Application failed: %v", err)
		os.Exit(1)
	}
}

func run(port int, origin *url.URL) error {
	log.Printf("port: %d, origin: %s", port, origin.String())
	return nil
}
