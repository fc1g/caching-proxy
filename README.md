# caching-proxy

A CLI caching proxy that forwards requests to an origin server and caches its responses on disk.

Built for the [roadmap.sh caching proxy project](https://roadmap.sh/projects/caching-server).

## Requirements

Go 1.27.1 or newer.

## Quick start

Clone the project and build the app:

```sh
git clone https://github.com/fc1g/caching-proxy.git
cd caching-proxy
go build -o caching-proxy ./cmd/caching-proxy
```

Start the proxy:

```sh
./caching-proxy --port 3000 --origin http://dummyjson.com
```

To run `caching-proxy` from any folder, install it instead of building it, and make sure your Go bin directory is in your `PATH`:

```sh
go install ./cmd/caching-proxy
```

## Usage

| Flag | Description |
|---|---|
| `--port` | The port the proxy listens on, from 1 to 65535. Required to start the proxy. |
| `--origin` | The `http` or `https` URL of the origin server. Required to start the proxy. |
| `--clear-cache` | Clears the cache and exits. `--port` and `--origin` are not needed. |

The proxy keeps running in the current terminal. To clear the cache while it is running, use another terminal:

```sh
./caching-proxy --clear-cache
```

## Checking that it works

Send the same request twice:

```sh
curl -s -D - -o /dev/null "http://localhost:3000/products?limit=3" | grep -i x-cache
```

The first request reaches the origin:

```text
X-Cache: MISS
```

The next one is served from the cache, without contacting the origin:

```text
X-Cache: HIT
```

## What gets cached

A response is cached only if all of the following are true:

- The request method is `GET`.
- The request has no `Authorization` or `Cookie` header.
- The response status code is 200.
- The response has no `Set-Cookie` header.

All other requests are forwarded to the origin and always get `X-Cache: MISS`.

The cache key consists of the request method and the full origin URL, including the path and query string.

## Where the cache is stored

Each cached response is stored as a separate file in the `caching-proxy` folder inside the user cache directory:

- macOS: `~/Library/Caches/caching-proxy`
- Linux: `~/.cache/caching-proxy`, or `$XDG_CACHE_HOME/caching-proxy` if that variable is set

The cache survives proxy restarts.

## Limitations

- Cached responses never expire. They stay on disk until you run `--clear-cache`.
- `Cache-Control` and `Vary` headers are ignored.
- Response bodies are read fully into memory, so very large responses use a lot of memory.
- `HEAD` requests are not cached, so `curl -I` always gets `X-Cache: MISS`.
- `Accept-Encoding` is not forwarded, so clients always receive uncompressed responses.
- Requests to the origin time out after 8 seconds.

## Tests

```sh
go test -race ./...
```
