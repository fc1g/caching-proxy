package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/fc1g/caching-proxy/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeHTTP_SecondGetIsServedFromCache(t *testing.T) {
	var originHits atomic.Int64
	origin := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originHits.Add(1)
		w.Write([]byte("hello, world"))
	}))

	client := origin.Client()

	require := require.New(t)
	assert := assert.New(t)

	originURL, err := url.Parse(origin.URL)
	require.NoError(err)
	cacheStore, err := cache.NewStore(t.TempDir())
	require.NoError(err)

	handler := NewHandler(originURL, cacheStore, client)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/products?limit=3", nil)

	handler.ServeHTTP(rec, req)

	xCache := rec.Result().Header.Get(headerXCache)
	require.Equal(cacheMiss, xCache)

	newRec := httptest.NewRecorder()
	newReq := httptest.NewRequest("GET", "/products?limit=3", nil)

	handler.ServeHTTP(newRec, newReq)

	newXCache := newRec.Result().Header.Get(headerXCache)
	require.Equal(cacheHit, newXCache)

	assert.Equal(originHits.Load(), int64(1))
	assert.Equal(rec.Body.String(), newRec.Body.String())
}
