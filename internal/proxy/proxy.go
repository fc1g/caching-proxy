package proxy

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
)

type Handler struct {
	origin *url.URL
	client *http.Client
}

func NewHandler(o *url.URL, c *http.Client) *Handler {
	return &Handler{
		origin: o,
		client: c,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	shouldCache := true

	fullURL := h.origin.JoinPath(r.URL.EscapedPath())
	fullURL.RawQuery = r.URL.RawQuery

	if r.Method != http.MethodGet {
		shouldCache = false
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, fullURL.String(), r.Body)
	if err != nil {
		log.Printf("new request with context: %v", err)
		writeUpstreamError(w, err)
		return
	}
	req.ContentLength = r.ContentLength

	for key, values := range r.Header {
		if key == "Authorization" || key == "Cookie" {
			shouldCache = false
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
	defer res.Body.Close()

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

	log.Printf("shouldCache: %t\n", shouldCache)

	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(res.StatusCode)
	w.Write(data)
}

func writeUpstreamError(w http.ResponseWriter, err error) {
	w.Header().Set("X-Cache", "MISS")
	
	if netErr, ok := errors.AsType[net.Error](err); ok && netErr.Timeout() {
		http.Error(w, http.StatusText(http.StatusGatewayTimeout), http.StatusGatewayTimeout)
		return
	}

	http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	return
}
