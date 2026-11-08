package proxy

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/fc1g/caching-proxy/internal/cache"
)

const (
	headerXCache = "X-Cache"
	cacheHit     = "HIT"
	cacheMiss    = "MISS"
)

type Handler struct {
	origin *url.URL
	cache  *cache.Store
	client *http.Client
}

func NewHandler(o *url.URL, cs *cache.Store, c *http.Client) *Handler {
	return &Handler{
		origin: o,
		cache:  cs,
		client: c,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	shouldCache := true

	fullURL := h.origin.JoinPath(r.URL.EscapedPath())
	fullURL.RawQuery = r.URL.RawQuery

	cacheKey := cache.GenerateCacheKey(r.Method, fullURL.String())

	if r.Method != http.MethodGet {
		shouldCache = false
	}

	for key := range r.Header {
		if key == "Authorization" || key == "Cookie" {
			shouldCache = false
		}
	}

	if shouldCache {
		if e, err := h.cache.Get(cacheKey); err == nil {
			for key, values := range e.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}

			w.Header().Set(headerXCache, cacheHit)
			w.WriteHeader(e.StatusCode)
			if _, err := w.Write(e.Body); err != nil {
				log.Printf("write response: %v", err)
			}

			return
		}
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, fullURL.String(), r.Body)
	if err != nil {
		log.Printf("new request with context: %v", err)
		writeUpstreamError(w, err)
		return
	}
	req.ContentLength = r.ContentLength

	for key, values := range r.Header {
		if key == "Accept-Encoding" {
			continue
		}

		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	res, err := h.client.Do(req)
	if err != nil {
		log.Printf("network error: %v", err)
		writeUpstreamError(w, err)
		return
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		shouldCache = false
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("read response body: %v", err)
		writeUpstreamError(w, err)
		return
	}

	for key, values := range res.Header {
		if key == "Set-Cookie" {
			shouldCache = false
		}

		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	if shouldCache {
		if err := h.cache.Set(cacheKey, cache.Entry{
			Body:       data,
			StatusCode: res.StatusCode,
			Header:     res.Header,
		}); err != nil {
			log.Printf("set cache: %v", err)
		}
	}

	w.Header().Set(headerXCache, cacheMiss)
	w.WriteHeader(res.StatusCode)
	if _, err := w.Write(data); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeUpstreamError(w http.ResponseWriter, err error) {
	w.Header().Set(headerXCache, cacheMiss)

	if netErr, ok := errors.AsType[net.Error](err); ok && netErr.Timeout() {
		http.Error(w, http.StatusText(http.StatusGatewayTimeout), http.StatusGatewayTimeout)
		return
	}

	http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
}
