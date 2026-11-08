package config

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
)

const (
	minPort = 1
	maxPort = 65535
)

type Config struct {
	Port   int
	Origin *url.URL
}

func Load() (*Config, error) {
	port := flag.Int("port", 0, "is the port on which the caching proxy server will run")
	origin := flag.String("origin", "", "is the URL of the server to which the requests will be forwarded")

	flag.Parse()

	if err := validatePort(*port); err != nil {
		return nil, err
	}

	originURL, err := parseOrigin(*origin)
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:   *port,
		Origin: originURL,
	}, nil
}

func validatePort(port int) error {
	if port < minPort || port > maxPort {
		return fmt.Errorf("invalid port %d: must be between %d and %d", port, minPort, maxPort)
	}

	return nil
}

func parseOrigin(origin string) (*url.URL, error) {
	if origin == "" {
		return nil, errors.New("--origin is required")
	}

	u, err := url.ParseRequestURI(origin)
	if err != nil {
		return nil, fmt.Errorf("invalid origin %q: %w", origin, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid origin %q: scheme must be http or https", origin)
	}

	if u.Host == "" {
		return nil, fmt.Errorf("invalid origin %q: host is missing", origin)
	}

	return u, nil
}
